package profile

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

// GetUserProfileHandler returns a user's profile information
func GetUserProfileHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		userID := vars["id"]

		var response ProfileResponse
		var sectorsJSON, targetGroupsJSON string // Temporary variables to hold the JSON strings

		err := db.QueryRow(SelectProfileQuery, userID).Scan(
			&response.ID,
			&response.MissionStatement,
			&sectorsJSON,
			&targetGroupsJSON,
			&response.ProjectStage,
		)

		if err == sql.ErrNoRows {
			http.Error(w, "Profile not found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Parse JSON arrays from strings
		err = json.Unmarshal([]byte(sectorsJSON), &response.Sectors)
		if err != nil {
			http.Error(w, "Error parsing sectors", http.StatusInternalServerError)
			return
		}

		err = json.Unmarshal([]byte(targetGroupsJSON), &response.TargetGroups)
		if err != nil {
			http.Error(w, "Error parsing target groups", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
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
			&response.Website,
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
		userID := r.Context().Value("user_id").(string)

		var response BioResponse
		err := db.QueryRow(SelectBioQuery, userID).Scan(
			&response.ID,
			&response.Location,
			&response.Website,
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

// UpdateProfileHandler handles updating a user's profile information
func UpdateProfileHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get user ID from context (set by auth middleware)
		userID := r.Context().Value("user_id").(string)

		// Parse request body
		var profile struct {
			MissionStatement string   `json:"mission_statement"`
			Sectors          []string `json:"sectors"`
			TargetGroups     []string `json:"target_groups"`
			ProjectStage     string   `json:"project_stage"`
		}

		if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Start a transaction
		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// Update profile
		_, err = tx.Exec(`
			UPDATE matching_profiles
			SET mission_statement = $1,
				sectors = $2,
				target_groups = $3,
				project_stage = $4
			WHERE user_id = $5
		`, profile.MissionStatement, pq.Array(profile.Sectors), pq.Array(profile.TargetGroups), profile.ProjectStage, userID)

		if err != nil {
			http.Error(w, "Failed to update profile", http.StatusInternalServerError)
			return
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}

		// Return updated profile
		var response ProfileResponse
		err = db.QueryRow(SelectProfileQuery, userID).Scan(
			&response.ID,
			&response.MissionStatement,
			&response.Sectors,
			&response.TargetGroups,
			&response.ProjectStage,
		)

		if err != nil {
			http.Error(w, "Failed to retrieve updated profile", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(response)
	}
}
