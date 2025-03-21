package connection

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"matchme/backend/handlers/auth"
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
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var connections []Connection
		for rows.Next() {
			var conn Connection
			err := rows.Scan(
				&conn.ID,
				&conn.InitiatorID,
				&conn.TargetID,
				&conn.CreatedAt,
				&conn.UpdatedAt,
				&conn.OtherUserName,
				&conn.OtherUserPicture,
				&conn.ConnectionType,
			)
			if err != nil {
				http.Error(w, "Error scanning connection", http.StatusInternalServerError)
				return
			}
			connections = append(connections, conn)
		}

		json.NewEncoder(w).Encode(connections)
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
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Check if connection already exists
		var exists bool
		err = db.QueryRow(CheckConnectionExistsQuery, userID, req.TargetID).Scan(&exists)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if exists {
			http.Error(w, "Connection already exists", http.StatusConflict)
			return
		}

		// Create new connection
		var conn Connection
		err = db.QueryRow(CreateConnectionQuery, userID, req.TargetID).Scan(
			&conn.ID,
			&conn.CreatedAt,
			&conn.UpdatedAt,
		)
		if err != nil {
			http.Error(w, "Failed to create connection", http.StatusInternalServerError)
			return
		}

		conn.InitiatorID = userID
		conn.TargetID = req.TargetID

		json.NewEncoder(w).Encode(conn)
	}
}

// DeleteConnectionHandler removes a connection
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

		result, err := db.Exec(DeleteConnectionQuery, userID, targetID)
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

		w.WriteHeader(http.StatusNoContent)
	}
}

// PotentialMatch represents a potential match between a provider and recipient
type PotentialMatch struct {
	ProviderID       int     `json:"provider_id"`
	ProviderName     string  `json:"provider_name"`
	ProviderPicture  string  `json:"provider_picture"`
	RecipientID      int     `json:"recipient_id"`
	RecipientName    string  `json:"recipient_name"`
	RecipientPicture string  `json:"recipient_picture"`
	SectorScore      float64 `json:"sector_score"`
	TargetGroupScore float64 `json:"target_group_score"`
	BudgetScore      float64 `json:"budget_score"`
	TimelineScore    float64 `json:"timeline_score"`
	StageScore       float64 `json:"stage_score"`
	MatchScore       float64 `json:"match_score"`
}

// GetPotentialMatchesHandler returns potential matches based on grant criteria
func GetPotentialMatchesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, err := auth.GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get user's role
		var userRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = $1", userID).Scan(&userRole)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		rows, err := db.Query(GetPotentialMatchesQuery)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var matches []PotentialMatch
		for rows.Next() {
			var match PotentialMatch
			err := rows.Scan(
				&match.ProviderID,
				&match.ProviderName,
				&match.ProviderPicture,
				&match.RecipientID,
				&match.RecipientName,
				&match.RecipientPicture,
				&match.SectorScore,
				&match.TargetGroupScore,
				&match.BudgetScore,
				&match.TimelineScore,
				&match.StageScore,
				&match.MatchScore,
			)
			if err != nil {
				http.Error(w, "Error scanning match", http.StatusInternalServerError)
				return
			}

			// Filter matches based on user's role
			if userRole == "provider" && match.ProviderID == userID {
				matches = append(matches, match)
			} else if userRole == "recipient" && match.RecipientID == userID {
				matches = append(matches, match)
			}
		}

		json.NewEncoder(w).Encode(matches)
	}
}
