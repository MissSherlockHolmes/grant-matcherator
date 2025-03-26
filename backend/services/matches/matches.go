package matches

import (
	"database/sql"
	"fmt"
	"log"
)

// CalculateAndStoreMatches calculates and stores matches for a user
func CalculateAndStoreMatches(db *sql.DB, userID int64, userRole string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}
	defer tx.Rollback()

	// Drop and recreate the temp_matches table to ensure a clean state
	_, err = tx.Exec(`DROP TABLE IF EXISTS temp_matches`)
	if err != nil {
		return fmt.Errorf("error dropping temp table: %v", err)
	}

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
		return fmt.Errorf("error creating dismissed_matches table: %v", err)
	}

	// Create temporary table for matches
	_, err = tx.Exec(`
		CREATE TABLE temp_matches (
			user_id BIGINT NOT NULL,
			match_id BIGINT NOT NULL,
			match_score FLOAT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, match_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating temp table: %v", err)
	}

	// Build the match calculation query based on user role
	var query string
	if userRole == "provider" {
		query = `
			INSERT INTO temp_matches (user_id, match_id, match_score)
			SELECT 
				$1 as user_id,
				u.id as match_id,
				(
					-- Sector match score (30%)
					COALESCE(
						(
							SELECT COUNT(*) 
							FROM UNNEST(p1.sectors) s
							WHERE s = ANY(p2.sectors)
						)::float / 
						NULLIF(
							(
								SELECT COUNT(*) 
								FROM UNNEST(p2.sectors) s
							), 0
						),
						0
					) * 30 +
					-- Target group match score (30%)
					COALESCE(
						(
							SELECT COUNT(*) 
							FROM UNNEST(p1.target_groups) t
							WHERE t = ANY(p2.target_groups)
						)::float / 
						NULLIF(
							(
								SELECT COUNT(*) 
								FROM UNNEST(p2.target_groups) t
							), 0
						),
						0
					) * 30
				) as match_score
			FROM users u
			JOIN profiles p1 ON u.id = p1.user_id
			JOIN profiles p2 ON p2.user_id = $1
			JOIN recipient_data r ON u.id = r.user_id
			WHERE u.role = 'recipient'
			AND u.status = 'active'
			AND NOT EXISTS (
				SELECT 1 FROM dismissed_matches dm
				WHERE dm.user_id = $1 AND dm.match_id = u.id
			)
			AND NOT EXISTS (
				SELECT 1 FROM connections c
				WHERE (c.initiator_id = $1 AND c.target_id = u.id)
				   OR (c.initiator_id = u.id AND c.target_id = $1)
			)
			AND (
				-- Sector match (if both have sectors)
				(p1.sectors IS NOT NULL AND p2.sectors IS NOT NULL AND p1.sectors && p2.sectors)
				OR
				-- Target group match (if both have target groups)
				(p1.target_groups IS NOT NULL AND p2.target_groups IS NOT NULL AND p1.target_groups && p2.target_groups)
			)
			AND (
				-- Sector match score
				COALESCE(
					(
						SELECT COUNT(*) 
						FROM UNNEST(p1.sectors) s
						WHERE s = ANY(p2.sectors)
					)::float / 
					NULLIF(
						(
							SELECT COUNT(*) 
							FROM UNNEST(p2.sectors) s
						), 0
					),
					0
				) * 30 +
				-- Target group match score
				COALESCE(
					(
						SELECT COUNT(*) 
						FROM UNNEST(p1.target_groups) t
						WHERE t = ANY(p2.target_groups)
					)::float / 
					NULLIF(
						(
							SELECT COUNT(*) 
							FROM UNNEST(p2.target_groups) t
						), 0
					),
					0
				) * 30
			) >= 30  -- At least 30% match on sectors and target groups combined
		`
	} else {
		query = `
			INSERT INTO temp_matches (user_id, match_id, match_score)
			SELECT 
				$1 as user_id,
				u.id as match_id,
				(
					-- Sector match score (30%)
					COALESCE(
						(
							SELECT COUNT(*) 
							FROM UNNEST(p1.sectors) s
							WHERE s = ANY(p2.sectors)
						)::float / 
						NULLIF(
							(
								SELECT COUNT(*) 
								FROM UNNEST(p2.sectors) s
							), 0
						),
						0
					) * 30 +
					-- Target group match score (30%)
					COALESCE(
						(
							SELECT COUNT(*) 
							FROM UNNEST(p1.target_groups) t
							WHERE t = ANY(p2.target_groups)
						)::float / 
						NULLIF(
							(
								SELECT COUNT(*) 
								FROM UNNEST(p2.target_groups) t
							), 0
						),
						0
					) * 30
				) as match_score
			FROM users u
			JOIN profiles p1 ON u.id = p1.user_id
			JOIN profiles p2 ON p2.user_id = $1
			JOIN provider_data p ON u.id = p.user_id
			WHERE u.role = 'provider'
			AND u.status = 'active'
			AND NOT EXISTS (
				SELECT 1 FROM dismissed_matches dm
				WHERE dm.user_id = $1 AND dm.match_id = u.id
			)
			AND NOT EXISTS (
				SELECT 1 FROM connections c
				WHERE (c.initiator_id = $1 AND c.target_id = u.id)
				   OR (c.initiator_id = u.id AND c.target_id = $1)
			)
			AND (
				-- Sector match (if both have sectors)
				(p1.sectors IS NOT NULL AND p2.sectors IS NOT NULL AND p1.sectors && p2.sectors)
				OR
				-- Target group match (if both have target groups)
				(p1.target_groups IS NOT NULL AND p2.target_groups IS NOT NULL AND p1.target_groups && p2.target_groups)
			)
			AND (
				-- Sector match score
				COALESCE(
					(
						SELECT COUNT(*) 
						FROM UNNEST(p1.sectors) s
						WHERE s = ANY(p2.sectors)
					)::float / 
					NULLIF(
						(
							SELECT COUNT(*) 
							FROM UNNEST(p2.sectors) s
						), 0
					),
					0
				) * 30 +
				-- Target group match score
				COALESCE(
					(
						SELECT COUNT(*) 
						FROM UNNEST(p1.target_groups) t
						WHERE t = ANY(p2.target_groups)
					)::float / 
					NULLIF(
						(
							SELECT COUNT(*) 
							FROM UNNEST(p2.target_groups) t
						), 0
					),
					0
				) * 30
			) >= 30  -- At least 30% match on sectors and target groups combined
		`
	}

	// Execute the match calculation query
	_, err = tx.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("error calculating matches: %v", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %v", err)
	}

	return nil
}

// GetStoredMatches retrieves pre-calculated matches for a user
func GetStoredMatches(db *sql.DB, userID int64) ([]Match, error) {
	query := `
		SELECT 
			tm.match_id,
			tm.match_score,
			u.email,
			p.organization_name,
			p.profile_picture_url
		FROM temp_matches tm
		JOIN users u ON u.id = tm.match_id
		LEFT JOIN profiles p ON p.user_id = tm.match_id
		WHERE tm.user_id = $1
		ORDER BY tm.match_score DESC
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("error querying matches: %v", err)
	}
	defer rows.Close()

	var matches []Match
	for rows.Next() {
		var match Match
		err := rows.Scan(
			&match.ID,
			&match.Score,
			&match.Email,
			&match.OrganizationName,
			&match.ProfilePictureURL,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning match: %v", err)
		}
		matches = append(matches, match)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating matches: %v", err)
	}

	return matches, nil
}

// RecalculateMatchesForAllUsers recalculates matches for all users
func RecalculateMatchesForAllUsers(db *sql.DB) error {
	rows, err := db.Query("SELECT id, role FROM users WHERE status = 'active'")
	if err != nil {
		return fmt.Errorf("error querying users: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID int64
		var role string
		if err := rows.Scan(&userID, &role); err != nil {
			log.Printf("Error scanning user: %v", err)
			continue
		}

		if err := CalculateAndStoreMatches(db, userID, role); err != nil {
			log.Printf("Error calculating matches for user %d: %v", userID, err)
			continue
		}
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("error iterating users: %v", err)
	}

	return nil
}

// Match represents a match between users
type Match struct {
	ID                int64          `json:"id"`
	Score             float64        `json:"score"`
	Email             string         `json:"email"`
	OrganizationName  string         `json:"organization_name"`
	ProfilePictureURL sql.NullString `json:"profile_picture_url"`
}
