package profile

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"matcherator/backend/handlers/auth"
	"matcherator/backend/handlers/user_status"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

// GetUserProfileHandler returns a user's profile information
func GetUserProfileHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var userID string
		vars := mux.Vars(r)
		if id := vars["id"]; id != "" {
			userID = id
		} else {
			// If no ID in URL, get from token (for /api/me/profile)
			tokenUserID, err := auth.GetUserIDFromToken(r)
			if err != nil {
				log.Printf("Error getting user ID from token: %v", err)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			userID = strconv.Itoa(tokenUserID)
		}

		log.Printf("Fetching profile for user ID: %s", userID)

		var response ProfileResponse
		var sectorsJSON, targetGroupsJSON string
		err := db.QueryRow(SelectProfileQuery, userID).Scan(
			&response.ID,
			&response.OrganizationName,
			&response.ProfilePictureURL,
			&response.MissionStatement,
			&response.State,
			&response.City,
			&response.ZipCode,
			&response.EIN,
			&response.Language,
			&response.ApplicantType,
			&sectorsJSON,
			&targetGroupsJSON,
			&response.ProjectStage,
			&response.WebsiteURL,
			&response.ContactEmail,
			&response.ChatOptIn,
			&response.Location,
			&response.Role,
			&response.Status,
		)

		if err == sql.ErrNoRows {
			log.Printf("Profile not found for user ID: %s", userID)
			http.Error(w, "Profile not found", http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("Database error fetching profile for user ID %s: %v", userID, err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		log.Printf("Raw sectors JSON: %s", sectorsJSON)
		log.Printf("Raw target groups JSON: %s", targetGroupsJSON)

		// Parse JSON arrays into string slices
		if err := json.Unmarshal([]byte(sectorsJSON), &response.Sectors); err != nil {
			log.Printf("Error parsing sectors JSON: %v", err)
			http.Error(w, "Error parsing sectors", http.StatusInternalServerError)
			return
		}
		if err := json.Unmarshal([]byte(targetGroupsJSON), &response.TargetGroups); err != nil {
			log.Printf("Error parsing target groups JSON: %v", err)
			http.Error(w, "Error parsing target groups", http.StatusInternalServerError)
			return
		}

		// Send response
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding response: %v", err)
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
			return
		}
	}
}

// GetUserBioHandler returns a user's biographical information
func GetUserBioHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		userID := vars["id"]

		var response BioResponse
		err := db.QueryRow(SelectBioQuery, userID).Scan(
			&response.ID,
			&response.Location,
			&response.WebsiteURL,
		)

		if err == sql.ErrNoRows {
			http.Error(w, "Bio not found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetMyBioHandler returns the authenticated user's biographical information
func GetMyBioHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get user ID from token
		tokenUserID, err := auth.GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var response BioResponse
		err = db.QueryRow(SelectBioQuery, tokenUserID).Scan(
			&response.ID,
			&response.Location,
			&response.WebsiteURL,
		)

		if err == sql.ErrNoRows {
			http.Error(w, "Bio not found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(response)
	}
}

// UpdateProfileHandler handles updating a user's profile information
func (h *Handler) UpdateProfileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, err := auth.GetUserIDFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// First, get the existing profile
	var existingProfile ProfileResponse
	var sectorsJSON, targetGroupsJSON string
	err = h.db.QueryRow(SelectProfileQuery, userID).Scan(
		&existingProfile.ID,
		&existingProfile.OrganizationName,
		&existingProfile.ProfilePictureURL,
		&existingProfile.MissionStatement,
		&existingProfile.State,
		&existingProfile.City,
		&existingProfile.ZipCode,
		&existingProfile.EIN,
		&existingProfile.Language,
		&existingProfile.ApplicantType,
		&sectorsJSON,
		&targetGroupsJSON,
		&existingProfile.ProjectStage,
		&existingProfile.WebsiteURL,
		&existingProfile.ContactEmail,
		&existingProfile.ChatOptIn,
		&existingProfile.Location,
		&existingProfile.Role,
		&existingProfile.Status,
	)

	if err != nil {
		log.Printf("Error fetching existing profile: %v", err)
		http.Error(w, "Error fetching existing profile", http.StatusInternalServerError)
		return
	}

	// Parse JSON arrays into string slices
	if err := json.Unmarshal([]byte(sectorsJSON), &existingProfile.Sectors); err != nil {
		log.Printf("Error parsing existing sectors: %v", err)
		http.Error(w, "Error parsing sectors", http.StatusInternalServerError)
		return
	}
	if err := json.Unmarshal([]byte(targetGroupsJSON), &existingProfile.TargetGroups); err != nil {
		log.Printf("Error parsing existing target groups: %v", err)
		http.Error(w, "Error parsing target groups", http.StatusInternalServerError)
		return
	}

	// Parse the update request
	var updateRequest struct {
		OrganizationName  *string  `json:"organization_name,omitempty"`
		ProfilePictureURL *string  `json:"profile_picture_url,omitempty"`
		MissionStatement  *string  `json:"mission_statement,omitempty"`
		State             *string  `json:"state,omitempty"`
		City              *string  `json:"city,omitempty"`
		ZipCode           *string  `json:"zip_code,omitempty"`
		EIN               *string  `json:"ein,omitempty"`
		Language          *string  `json:"language,omitempty"`
		ApplicantType     *string  `json:"applicant_type,omitempty"`
		Sectors           []string `json:"sectors,omitempty"`
		TargetGroups      []string `json:"target_groups,omitempty"`
		ProjectStage      *string  `json:"project_stage,omitempty"`
		WebsiteURL        *string  `json:"website_url,omitempty"`
		ContactEmail      *string  `json:"contact_email,omitempty"`
		ChatOptIn         *bool    `json:"chat_opt_in,omitempty"`
		Location          *string  `json:"location,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateRequest); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Merge updates with existing profile
	if updateRequest.OrganizationName != nil {
		existingProfile.OrganizationName = *updateRequest.OrganizationName
	}
	if updateRequest.ProfilePictureURL != nil {
		existingProfile.ProfilePictureURL = updateRequest.ProfilePictureURL
	}
	if updateRequest.MissionStatement != nil {
		existingProfile.MissionStatement = *updateRequest.MissionStatement
	}
	if updateRequest.State != nil {
		existingProfile.State = *updateRequest.State
	}
	if updateRequest.City != nil {
		existingProfile.City = *updateRequest.City
	}
	if updateRequest.ZipCode != nil {
		existingProfile.ZipCode = *updateRequest.ZipCode
	}
	if updateRequest.EIN != nil {
		existingProfile.EIN = *updateRequest.EIN
	}
	if updateRequest.Language != nil {
		existingProfile.Language = *updateRequest.Language
	}
	if updateRequest.ApplicantType != nil {
		existingProfile.ApplicantType = *updateRequest.ApplicantType
	}
	if updateRequest.Sectors != nil {
		existingProfile.Sectors = updateRequest.Sectors
	}
	if updateRequest.TargetGroups != nil {
		existingProfile.TargetGroups = updateRequest.TargetGroups
	}
	if updateRequest.ProjectStage != nil {
		existingProfile.ProjectStage = *updateRequest.ProjectStage
	}
	if updateRequest.WebsiteURL != nil {
		existingProfile.WebsiteURL = *updateRequest.WebsiteURL
	}
	if updateRequest.ContactEmail != nil {
		existingProfile.ContactEmail = *updateRequest.ContactEmail
	}
	if updateRequest.ChatOptIn != nil {
		existingProfile.ChatOptIn = *updateRequest.ChatOptIn
	}
	if updateRequest.Location != nil {
		existingProfile.Location = *updateRequest.Location
	}

	// Start transaction
	tx, err := h.db.Begin()
	if err != nil {
		log.Printf("Failed to start transaction: %v", err)
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Update profile with merged data
	result, err := tx.Exec(`
		UPDATE profiles
		SET organization_name = $1,
			profile_picture_url = $2,
			mission_statement = $3,
			state = $4,
			city = $5,
			zip_code = $6,
			ein = $7,
			language = $8,
			applicant_type = $9,
			sectors = $10::text[],
			target_groups = $11::text[],
			project_stage = $12,
			website_url = $13,
			contact_email = $14,
			chat_opt_in = $15,
			location = $16
		WHERE user_id = $17
	`, existingProfile.OrganizationName,
		existingProfile.ProfilePictureURL,
		existingProfile.MissionStatement,
		existingProfile.State,
		existingProfile.City,
		existingProfile.ZipCode,
		existingProfile.EIN,
		existingProfile.Language,
		existingProfile.ApplicantType,
		pq.Array(existingProfile.Sectors),
		pq.Array(existingProfile.TargetGroups),
		existingProfile.ProjectStage,
		existingProfile.WebsiteURL,
		existingProfile.ContactEmail,
		existingProfile.ChatOptIn,
		existingProfile.Location,
		userID)

	if err != nil {
		log.Printf("Failed to update profile: %v", err)
		http.Error(w, fmt.Sprintf("Failed to update profile: %v", err), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error checking rows affected: %v", err)
	} else {
		log.Printf("Rows affected by update: %d", rowsAffected)
	}

	// Update user status
	if err := user_status.UpdateUserStatus(tx, strconv.Itoa(userID)); err != nil {
		http.Error(w, "Failed to update user status", http.StatusInternalServerError)
		return
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(existingProfile)
}

// UpdateProfileHandler handles updating a user's profile information
func UpdateProfileHandler(db *sql.DB) http.HandlerFunc {
	h := NewHandler(db)
	return h.UpdateProfileHandler
}
