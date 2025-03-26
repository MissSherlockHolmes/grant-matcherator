package media

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"matcherator/backend/handlers/auth"
)

const (
	maxFileSize = 10 << 20 // 10 MB
)

var allowedTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
}

// UploadResponse represents the response for a successful upload
type UploadResponse struct {
	URL string `json:"url"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// UploadProfilePictureHandler handles profile picture uploads
func UploadProfilePictureHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get user ID from token
		userID, err := auth.GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Parse multipart form
		if err := r.ParseMultipartForm(maxFileSize); err != nil {
			json.NewEncoder(w).Encode(ErrorResponse{Error: "File too large. Maximum size is 10MB"})
			return
		}

		file, handler, err := r.FormFile("file")
		if err != nil {
			json.NewEncoder(w).Encode(ErrorResponse{Error: "No file uploaded"})
			return
		}
		defer file.Close()

		// Validate file type
		if !allowedTypes[handler.Header.Get("Content-Type")] {
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid file type. Only JPEG, PNG, and GIF are allowed"})
			return
		}

		// Generate unique filename
		filename := fmt.Sprintf("%d_%s", userID, handler.Filename)
		uploadPath := filepath.Join("uploads", "profile_pictures", filename)

		// Ensure upload directory exists
		if err := os.MkdirAll(filepath.Dir(uploadPath), 0755); err != nil {
			http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
			return
		}

		// Create the file
		dst, err := os.Create(uploadPath)
		if err != nil {
			http.Error(w, "Failed to create file", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		// Copy the uploaded file
		if _, err := io.Copy(dst, file); err != nil {
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}

		// Update profile picture URL in database
		fileURL := fmt.Sprintf("/uploads/profile_pictures/%s", filename)
		_, err = db.Exec(`
			UPDATE profiles 
			SET profile_picture_url = $1,
				updated_at = CURRENT_TIMESTAMP
			WHERE user_id = $2
		`, fileURL, userID)

		if err != nil {
			// Clean up the uploaded file if database update fails
			os.Remove(uploadPath)
			http.Error(w, "Failed to update profile", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(UploadResponse{URL: fileURL})
	}
}

// DeleteProfilePictureHandler handles profile picture deletion
func DeleteProfilePictureHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get user ID from token
		userID, err := auth.GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get current profile picture URL
		var currentURL string
		err = db.QueryRow(`
			SELECT profile_picture_url 
			FROM profiles 
			WHERE user_id = $1
		`, userID).Scan(&currentURL)

		if err != nil {
			http.Error(w, "Profile not found", http.StatusNotFound)
			return
		}

		if currentURL == "" {
			http.Error(w, "No profile picture to delete", http.StatusBadRequest)
			return
		}

		// Extract filename from URL
		filename := filepath.Base(currentURL)
		uploadPath := filepath.Join("uploads", "profile_pictures", filename)

		// Delete the file
		if err := os.Remove(uploadPath); err != nil {
			// Log the error but continue with database update
			fmt.Printf("Error deleting file: %v\n", err)
		}

		// Update database to remove profile picture URL
		_, err = db.Exec(`
			UPDATE profiles 
			SET profile_picture_url = NULL,
				updated_at = CURRENT_TIMESTAMP
			WHERE user_id = $1
		`, userID)

		if err != nil {
			http.Error(w, "Failed to update profile", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
