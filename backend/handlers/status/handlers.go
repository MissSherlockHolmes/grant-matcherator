package status

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"matcherator/backend/handlers/auth"

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
				u.status,
				CURRENT_TIMESTAMP as last_update
			FROM users u
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
				u.status,
				CURRENT_TIMESTAMP as last_update
			FROM users u
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
			u.status,
			CURRENT_TIMESTAMP as last_update
		FROM users u
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
