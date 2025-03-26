package connection

import (
	"database/sql"
	"fmt"
	"log"
)

// CalculateAndStoreMatches calculates potential matches for a user and stores them in a temporary table
func CalculateAndStoreMatches(db *sql.DB, userID int, userRole string) error {
	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Error starting transaction for match calculation: %v", err)
		return err
	}
	defer tx.Rollback()

	// Create temporary table for storing matches
	_, err = tx.Exec(`
		CREATE TEMP TABLE IF NOT EXISTS temp_matches (
			user_id INTEGER NOT NULL,
			match_id INTEGER NOT NULL,
			match_score FLOAT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, match_id)
		)
	`)
	if err != nil {
		log.Printf("Error creating temporary table: %v", err)
		return err
	}

	// Clear existing matches for this user
	_, err = tx.Exec("DELETE FROM temp_matches WHERE user_id = $1", userID)
	if err != nil {
		log.Printf("Error clearing existing matches: %v", err)
		return err
	}

	// Calculate and store matches based on user role
	var query string
	if userRole == "provider" {
		query = `
			INSERT INTO temp_matches (user_id, match_id, match_score)
			SELECT 
				$1 as user_id,
				r.recipient_id as match_id,
				(sector_score * 0.3 + target_group_score * 0.3 + budget_score * 0.2 + 
				timeline_score * 0.1 + stage_score * 0.1) as match_score
			FROM (
				${GetPotentialMatchesQuery}
			) r
			WHERE r.provider_id = $1
		`
	} else {
		query = `
			INSERT INTO temp_matches (user_id, match_id, match_score)
			SELECT 
				$1 as user_id,
				p.provider_id as match_id,
				(sector_score * 0.3 + target_group_score * 0.3 + budget_score * 0.2 + 
				timeline_score * 0.1 + stage_score * 0.1) as match_score
			FROM (
				${GetPotentialMatchesQuery}
			) p
			WHERE p.recipient_id = $1
		`
	}

	// Execute the match calculation query
	_, err = tx.Exec(query, userID)
	if err != nil {
		log.Printf("Error calculating matches: %v", err)
		return err
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		log.Printf("Error committing match calculation: %v", err)
		return err
	}

	log.Printf("Successfully calculated and stored matches for user %d", userID)
	return nil
}

// GetStoredMatches retrieves the pre-calculated matches for a user
func GetStoredMatches(db *sql.DB, userID int) ([]PotentialMatch, error) {
	var matches []PotentialMatch

	rows, err := db.Query(`
		SELECT 
			tm.match_id,
			tm.match_score,
			COALESCE(u.email, '') as email,
			COALESCE(p.organization_name, '') as organization_name,
			p.profile_picture_url
		FROM temp_matches tm
		JOIN users u ON u.id = tm.match_id
		LEFT JOIN profiles p ON p.user_id = tm.match_id
		WHERE tm.user_id = $1
		ORDER BY tm.match_score DESC
	`, userID)

	if err != nil {
		log.Printf("Error executing query for stored matches: %v", err)
		return nil, fmt.Errorf("failed to query matches: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var match PotentialMatch
		err := rows.Scan(
			&match.ID,
			&match.Score,
			&match.Email,
			&match.OrganizationName,
			&match.ProfilePictureURL,
		)
		if err != nil {
			log.Printf("Error scanning match row: %v", err)
			return nil, fmt.Errorf("failed to scan match data: %v", err)
		}
		matches = append(matches, match)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating through match rows: %v", err)
		return nil, fmt.Errorf("error while processing matches: %v", err)
	}

	if len(matches) == 0 {
		log.Printf("No matches found for user %d", userID)
		return matches, nil
	}

	log.Printf("Successfully retrieved %d matches for user %d", len(matches), userID)
	return matches, nil
}

type PotentialMatch struct {
	ID                int64          `json:"id"`
	Score             float64        `json:"score"`
	Email             string         `json:"email"`
	OrganizationName  string         `json:"organization_name"`
	ProfilePictureURL sql.NullString `json:"profile_picture_url"`
}
