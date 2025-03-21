package status

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"matchme/backend/handlers/auth"

	"github.com/gorilla/mux"
)

// Status represents the current status of a user
type Status struct {
	UserID     int       `json:"user_id"`
	Role       string    `json:"role"`
	Status     string    `json:"status"`
	LastUpdate time.Time `json:"last_update"`
}

// GetStatusHandler returns the current status of a user
func GetStatusHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		vars := mux.Vars(r)
		userID := vars["id"]

		// Get user's role and status
		var status Status
		err := db.QueryRow(`
			SELECT 
				u.id,
				u.role,
				CASE 
					WHEN u.role = 'provider' THEN
						CASE 
							WHEN pd.deadline < CURRENT_TIMESTAMP THEN 'closed'
							ELSE 'active'
						END
					WHEN u.role = 'recipient' THEN
						CASE 
							WHEN p.organization_name IS NOT NULL 
								AND p.sectors IS NOT NULL 
								AND array_length(p.sectors, 1) > 0
								AND p.target_groups IS NOT NULL 
								AND array_length(p.target_groups, 1) > 0
								AND p.state IS NOT NULL 
								AND p.city IS NOT NULL 
								AND p.zip_code IS NOT NULL 
							THEN 'active'
							ELSE 'draft'
						END
				END as status,
				CURRENT_TIMESTAMP as last_update
			FROM users u
			LEFT JOIN profiles p ON u.id = p.user_id
			LEFT JOIN provider_data pd ON u.id = pd.user_id
			WHERE u.id = $1
		`, userID).Scan(&status.UserID, &status.Role, &status.Status, &status.LastUpdate)

		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(status)
	}
}

// GetMyStatusHandler returns the current status of the authenticated user
func GetMyStatusHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, err := auth.GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get user's role and status
		var status Status
		err = db.QueryRow(`
			SELECT 
				u.id,
				u.role,
				CASE 
					WHEN u.role = 'provider' THEN
						CASE 
							WHEN pd.deadline < CURRENT_TIMESTAMP THEN 'closed'
							ELSE 'active'
						END
					WHEN u.role = 'recipient' THEN
						CASE 
							WHEN p.organization_name IS NOT NULL 
								AND p.sectors IS NOT NULL 
								AND array_length(p.sectors, 1) > 0
								AND p.target_groups IS NOT NULL 
								AND array_length(p.target_groups, 1) > 0
								AND p.state IS NOT NULL 
								AND p.city IS NOT NULL 
								AND p.zip_code IS NOT NULL 
							THEN 'active'
							ELSE 'draft'
						END
				END as status,
				CURRENT_TIMESTAMP as last_update
			FROM users u
			LEFT JOIN profiles p ON u.id = p.user_id
			LEFT JOIN provider_data pd ON u.id = pd.user_id
			WHERE u.id = $1
		`, userID).Scan(&status.UserID, &status.Role, &status.Status, &status.LastUpdate)

		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(status)
	}
}

// GetUserStatus retrieves the current status of a user
func GetUserStatus(db *sql.DB, userID int) (*Status, error) {
	var status Status
	err := db.QueryRow(`
		SELECT 
			u.id,
			u.role,
			CASE 
				WHEN u.role = 'provider' THEN
					CASE 
						WHEN pd.deadline < CURRENT_TIMESTAMP THEN 'closed'
						ELSE 'active'
					END
				WHEN u.role = 'recipient' THEN
					CASE 
						WHEN p.organization_name IS NOT NULL 
							AND p.sectors IS NOT NULL 
							AND array_length(p.sectors, 1) > 0
							AND p.target_groups IS NOT NULL 
							AND array_length(p.target_groups, 1) > 0
							AND p.state IS NOT NULL 
							AND p.city IS NOT NULL 
							AND p.zip_code IS NOT NULL 
						THEN 'active'
						ELSE 'draft'
					END
			END as status,
			CURRENT_TIMESTAMP as last_update
		FROM users u
		LEFT JOIN profiles p ON u.id = p.user_id
		LEFT JOIN provider_data pd ON u.id = pd.user_id
		WHERE u.id = $1
	`, userID).Scan(&status.UserID, &status.Role, &status.Status, &status.LastUpdate)

	if err == sql.ErrNoRows {
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	return &status, nil
}
