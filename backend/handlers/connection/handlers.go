package connection

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"matcherator/backend/handlers/auth"
	"matcherator/backend/services/matches"
)

// GetConnectionsHandler returns all connections for the authenticated user
func GetConnectionsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, err := auth.GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		rows, err := db.Query(GetConnectionsQuery, userID)
		if err != nil {
			log.Printf("Error querying connections: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var connections []Connection
		for rows.Next() {
			var conn Connection
			var otherUserPicture sql.NullString
			err := rows.Scan(
				&conn.ID,
				&conn.InitiatorID,
				&conn.TargetID,
				&conn.CreatedAt,
				&conn.UpdatedAt,
				&conn.OtherUserName,
				&otherUserPicture,
				&conn.ConnectionType,
			)
			if err != nil {
				log.Printf("Error scanning connection: %v", err)
				http.Error(w, "Error scanning connection", http.StatusInternalServerError)
				return
			}
			conn.OtherUserPicture = otherUserPicture.String
			if conn.InitiatorID == userID {
				conn.ConnectionType = "follower"
			} else {
				conn.ConnectionType = "following"
			}
			connections = append(connections, conn)
		}

		if err = rows.Err(); err != nil {
			log.Printf("Error iterating over rows: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if err := json.NewEncoder(w).Encode(connections); err != nil {
			log.Printf("Error encoding response: %v", err)
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
			return
		}
	}
}

// CreateConnectionHandler creates a new connection
func CreateConnectionHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, err := auth.GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req ConnectionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Error decoding request body: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Check if connection already exists
		var exists bool
		err = db.QueryRow(CheckConnectionExistsQuery, userID, req.TargetID).Scan(&exists)
		if err != nil {
			log.Printf("Error checking if connection exists: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if exists {
			http.Error(w, "Connection already exists", http.StatusConflict)
			return
		}

		// Create new connection
		var conn Connection
		err = db.QueryRow(CreateConnectionQuery, userID, req.TargetID, "following").Scan(
			&conn.ID,
			&conn.CreatedAt,
			&conn.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error creating connection: %v", err)
			http.Error(w, "Failed to create connection", http.StatusInternalServerError)
			return
		}

		// Remove the matched user from temp_matches (both directions)
		_, err = db.Exec("DELETE FROM temp_matches WHERE (user_id = $1 AND match_id = $2) OR (user_id = $2 AND match_id = $1)", userID, req.TargetID)
		if err != nil {
			log.Printf("Error removing match from temp_matches: %v", err)
			// Don't return error here as the connection was still created successfully
		}

		conn.InitiatorID = userID
		conn.TargetID = req.TargetID
		conn.ConnectionType = "following"

		if err := json.NewEncoder(w).Encode(conn); err != nil {
			log.Printf("Error encoding response: %v", err)
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
			return
		}
	}
}

// DeleteConnectionHandler handles deleting a connection
func DeleteConnectionHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, err := auth.GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		targetIDStr := vars["id"]
		targetID, err := strconv.Atoi(targetIDStr)
		if err != nil {
			http.Error(w, "Invalid target ID", http.StatusBadRequest)
			return
		}

		result, err := db.Exec(DeleteConnectionQuery, targetID, userID)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if rowsAffected == 0 {
			http.Error(w, "Connection not found", http.StatusNotFound)
			return
		}

		// Get user's role and recalculate matches
		var role string
		err = db.QueryRow("SELECT role FROM users WHERE id = $1", userID).Scan(&role)
		if err != nil {
			log.Printf("Error getting user role: %v", err)
			// Don't return error here as the connection was still deleted successfully
		} else {
			err = matches.CalculateAndStoreMatches(db, int64(userID), role)
			if err != nil {
				log.Printf("Error recalculating matches: %v", err)
				// Don't return error here as the connection was still deleted successfully
			}
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// GetPotentialMatchesHandler returns potential matches based on grant criteria
func GetPotentialMatchesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, err := auth.GetUserIDFromToken(r)
		if err != nil {
			log.Printf("Error getting user ID from token: %v", err)
			http.Error(w, "Unauthorized: Invalid or missing token", http.StatusUnauthorized)
			return
		}

		log.Printf("Fetching potential matches for user %d", userID)

		// Get user's role
		var role string
		err = db.QueryRow("SELECT role FROM users WHERE id = $1", userID).Scan(&role)
		if err != nil {
			log.Printf("Error getting user role: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Recalculate matches for the current user
		if err := matches.CalculateAndStoreMatches(db, int64(userID), role); err != nil {
			log.Printf("Error recalculating matches: %v", err)
			http.Error(w, "Error recalculating matches", http.StatusInternalServerError)
			return
		}

		// Get pre-calculated matches
		potentialMatches, err := matches.GetStoredMatches(db, int64(userID))
		if err != nil {
			log.Printf("Error fetching potential matches: %v", err)
			http.Error(w, fmt.Sprintf("Error fetching potential matches: %v", err), http.StatusInternalServerError)
			return
		}

		log.Printf("Found %d potential matches for user %d", len(potentialMatches), userID)
		if len(potentialMatches) > 0 {
			log.Printf("First match: %+v", potentialMatches[0])
		}

		if err := json.NewEncoder(w).Encode(potentialMatches); err != nil {
			log.Printf("Error encoding response: %v", err)
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
			return
		}
	}
}

// RecalculateMatchesHandler triggers a recalculation of matches for the current user
func RecalculateMatchesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, err := auth.GetUserIDFromToken(r)
		if err != nil {
			log.Printf("Error getting user ID from token: %v", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		log.Printf("Recalculating matches for user %d", userID)

		// Get user's role
		var role string
		err = db.QueryRow("SELECT role FROM users WHERE id = $1", userID).Scan(&role)
		if err != nil {
			log.Printf("Error getting user role: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Calculate and store matches
		err = matches.CalculateAndStoreMatches(db, int64(userID), role)
		if err != nil {
			log.Printf("Error calculating matches: %v", err)
			http.Error(w, "Error calculating matches", http.StatusInternalServerError)
			return
		}

		log.Printf("Successfully recalculated matches for user %d", userID)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Matches recalculated successfully"})
	}
}

// DismissMatchHandler handles dismissing a potential match
func DismissMatchHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, err := auth.GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		targetIDStr := vars["id"]
		targetID, err := strconv.Atoi(targetIDStr)
		if err != nil {
			http.Error(w, "Invalid target ID", http.StatusBadRequest)
			return
		}

		// Start transaction
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Error starting transaction: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// Create dismissed_matches table if it doesn't exist
		_, err = tx.Exec(`
			CREATE TABLE IF NOT EXISTS dismissed_matches (
				user_id BIGINT NOT NULL,
				match_id BIGINT NOT NULL,
				dismissed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY (user_id, match_id)
			)
		`)
		if err != nil {
			log.Printf("Error creating dismissed_matches table: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Add the match to dismissed_matches
		_, err = tx.Exec("INSERT INTO dismissed_matches (user_id, match_id) VALUES ($1, $2)", userID, targetID)
		if err != nil {
			log.Printf("Error adding to dismissed_matches: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Remove the match from temp_matches
		result, err := tx.Exec("DELETE FROM temp_matches WHERE user_id = $1 AND match_id = $2", userID, targetID)
		if err != nil {
			log.Printf("Error removing match from temp_matches: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Printf("Error getting rows affected: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if rowsAffected == 0 {
			http.Error(w, "Match not found", http.StatusNotFound)
			return
		}

		// Commit transaction
		if err = tx.Commit(); err != nil {
			log.Printf("Error committing transaction: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
