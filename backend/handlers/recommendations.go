package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type IDResponse struct {
	ID int `json:"id"`
}

type PreferenceUpdate struct {
	Priority string `json:"priority"`
}

func SetMatchPreferenceHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		userID, err := GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		switch r.Method {
		case http.MethodGet:
			// Fetch user preference
			var preference string
			err = db.QueryRow(`SELECT priority FROM match_preferences WHERE user_id = $1`, userID).Scan(&preference)
			if err != nil {
				preference = "none" // Default if not found
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"priority": preference})

		case http.MethodPost:
			// Update preference
			var pref PreferenceUpdate
			if err := json.NewDecoder(r.Body).Decode(&pref); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}

			// Validate input
			validPreferences := map[string]bool{"looking_for": true, "interests": true, "age": true, "none": true}
			if !validPreferences[pref.Priority] {
				http.Error(w, "Invalid priority value", http.StatusBadRequest)
				return
			}

			_, err = db.Exec(`
				INSERT INTO match_preferences (user_id, priority) 
				VALUES ($1, $2)
				ON CONFLICT (user_id) DO UPDATE 
				SET priority = EXCLUDED.priority
			`, userID, pref.Priority)

			if err != nil {
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "Preference updated"})

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// GetRecommendationsHandler returns up to 10 recommended user IDs
func GetRecommendationsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, err := GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Fetch user’s sorting preference
		var priority string
		err = db.QueryRow(`SELECT priority FROM match_preferences WHERE user_id = $1`, userID).Scan(&priority)
		if err != nil {
			priority = "none" // Default if no preference set
		}

		// Dynamic ORDER BY clause based on priority
		orderByClause := "ORDER BY RANDOM()"
		if priority == "location" {
			orderByClause = "ORDER BY (p.location = (SELECT location FROM profiles WHERE user_id = $1)) DESC"
		} else if priority == "interests" {
			orderByClause = `
				ORDER BY (
					SELECT COUNT(*) FROM jsonb_array_elements_text(p.interests) 
					WHERE jsonb_array_elements_text(p.interests) IN 
					(SELECT jsonb_array_elements_text(interests) FROM profiles WHERE user_id = $1)
				) DESC
			`
		} else if priority == "age" {
			orderByClause = "ORDER BY ABS(p.age - (SELECT age FROM profiles WHERE user_id = $1)) ASC"
		}

		// Fetch potential matches
		query := `
			SELECT u.id 
			FROM users u
			JOIN profiles p ON u.id = p.user_id
			LEFT JOIN matches m ON (m.user_id_1 = $1 AND m.user_id_2 = u.id)
				OR (m.user_id_2 = $1 AND m.user_id_1 = u.id)
			WHERE u.id != $1
			AND m.id IS NULL
			` + orderByClause + `
			LIMIT 10
		`

		rows, err := db.Query(query, userID)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var recommendations []IDResponse
		for rows.Next() {
			var rec IDResponse
			if err := rows.Scan(&rec.ID); err != nil {
				http.Error(w, "Error scanning results", http.StatusInternalServerError)
				return
			}
			recommendations = append(recommendations, rec)
		}

		json.NewEncoder(w).Encode(recommendations)
	}
}

// GetConnectionsHandler returns all connected user IDs
func GetConnectionsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, err := GetUserIDFromToken(r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}

		rows, err := db.Query(`
			SELECT CASE 
				WHEN user_id_1 = $1 THEN user_id_2
				ELSE user_id_1
			END as connected_id
			FROM matches 
			WHERE (user_id_1 = $1 OR user_id_2 = $1)
			AND status = 'connected'
		`, userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
			return
		}
		defer rows.Close()

		var connections []IDResponse
		for rows.Next() {
			var conn IDResponse
			if err := rows.Scan(&conn.ID); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "Error scanning results"})
				return
			}
			connections = append(connections, conn)
		}

		json.NewEncoder(w).Encode(connections)
	}
}
