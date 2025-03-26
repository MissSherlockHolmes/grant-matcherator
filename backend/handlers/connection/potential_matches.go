package connection

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"matcherator/backend/handlers/auth"
	"matcherator/backend/services/matches"
)

// PotentialMatchesHandler handles fetching potential matches for a user
// Used by: /api/connections/potential-matches
// Response: []Match
func PotentialMatchesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := auth.GetUserIDFromToken(r)
		if err != nil {
			log.Printf("Error getting user ID from token: %v", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		log.Printf("Fetching potential matches for user %d", userID)

		// Get pre-calculated matches
		potentialMatches, err := matches.GetStoredMatches(db, int64(userID))
		if err != nil {
			log.Printf("Error fetching potential matches: %v", err)
			http.Error(w, "Error fetching potential matches", http.StatusInternalServerError)
			return
		}

		log.Printf("Found %d potential matches for user %d", len(potentialMatches), userID)
		if len(potentialMatches) > 0 {
			log.Printf("First match: %+v", potentialMatches[0])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(potentialMatches)
	}
}
