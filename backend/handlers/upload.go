package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func UploadProfilePictureHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get user ID from token
		userID, err := GetUserIDFromToken(r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}

		// Parse multipart form
		err = r.ParseMultipartForm(10 << 20) // 10 MB max
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to parse form"})
			return
		}

		file, handler, err := r.FormFile("file")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "No file uploaded"})
			return
		}
		defer file.Close()

		// Create file path
		filename := fmt.Sprintf("%d_%s", userID, handler.Filename)
		filepath := filepath.Join("uploads", filename)

		// Create the file
		dst, err := os.Create(filepath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create file"})
			return
		}
		defer dst.Close()

		// Copy the uploaded file to the created file
		if _, err := io.Copy(dst, file); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to save file"})
			return
		}

		// Update profile picture URL in database
		fileURL := fmt.Sprintf("/uploads/%s", filename)
		_, err = db.Exec(`
			UPDATE profiles 
			SET profile_picture_url = $1 
			WHERE user_id = $2
		`, fileURL, userID)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update profile"})
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"url": fileURL})
	}
}
