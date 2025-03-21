package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type UserStatus struct {
	UserID     int       `json:"user_id"`
	Status     string    `json:"status"`
	LastActive time.Time `json:"last_active"`
}

func UpdateUserStatusHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var status struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&status); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		_, err = db.Exec(`
			INSERT INTO user_status (user_id, status, last_active)
			VALUES ($1, $2, CURRENT_TIMESTAMP)
			ON CONFLICT (user_id)
			DO UPDATE SET
				status = $2,
				last_active = CURRENT_TIMESTAMP
		`, userID, status.Status)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func GetUserStatusHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		userID := vars["id"]

		var status UserStatus
		err := db.QueryRow(`
			SELECT user_id, status, last_active
			FROM user_status
			WHERE user_id = $1
		`, userID).Scan(&status.UserID, &status.Status, &status.LastActive)

		if err == sql.ErrNoRows {
			status = UserStatus{
				Status:     "offline",
				LastActive: time.Now(),
			}
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Check if user has been inactive for more than 1 minute
		if time.Since(status.LastActive) > 1*time.Minute {
			status.Status = "offline"

			// Update status in database
			_, err = db.Exec(`
				UPDATE user_status
				SET status = 'offline'
				WHERE user_id = $1 AND status = 'online'
			`, userID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	}
}
