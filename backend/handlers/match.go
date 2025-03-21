package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type Match struct {
	ID               int    `json:"id"`
	UserID1          int    `json:"user_id_1"`
	UserID2          int    `json:"user_id_2"`
	Status           string `json:"status"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
	OtherUserName    string `json:"other_user_name"`
	OtherUserPicture string `json:"other_user_picture"`
}

func GetMatchesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		rows, err := db.Query(`
			SELECT 
				m.id, 
				m.user_id_1, 
				m.user_id_2, 
				m.status, 
				m.created_at, 
				m.updated_at,
				CASE 
					WHEN m.user_id_1 = $1 THEN p2.name 
					ELSE p1.name 
				END as other_user_name,
				CASE 
					WHEN m.user_id_1 = $1 THEN p2.profile_picture_url 
					ELSE p1.profile_picture_url 
				END as other_user_picture
			FROM matches m
			JOIN profiles p1 ON m.user_id_1 = p1.user_id
			JOIN profiles p2 ON m.user_id_2 = p2.user_id
			WHERE (m.user_id_1 = $1 OR m.user_id_2 = $1) 
			AND m.status = 'connected'
		`, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var matches []Match
		for rows.Next() {
			var match Match
			err := rows.Scan(
				&match.ID,
				&match.UserID1,
				&match.UserID2,
				&match.Status,
				&match.CreatedAt,
				&match.UpdatedAt,
				&match.OtherUserName,
				&match.OtherUserPicture,
			)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			matches = append(matches, match)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(matches)
	}
}

func GetPotentialMatchesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		rows, err := db.Query(`
			WITH user_interests AS (
				SELECT 
					interests,
					looking_for,
					age,
					location
				FROM profiles 
				WHERE user_id = $1
			),
			match_calculations AS (
				SELECT 
					p.user_id,
					p.name,
					p.bio,
					array_to_json(p.interests) AS interests,
					p.location,
					p.looking_for,
					p.age,
					p.occupation,
					p.profile_picture_url,
					CASE 
						WHEN m.status = 'pending' AND m.user_id_1 != $1 THEN true 
						ELSE false 
					END as has_liked_me,
					CASE
						WHEN ui.interests && p.interests THEN (
							CARDINALITY(ARRAY(
								SELECT UNNEST(ui.interests) 
								INTERSECT 
								SELECT UNNEST(p.interests)
							))::float / 
							GREATEST(
								CARDINALITY(ui.interests),
								CARDINALITY(p.interests)
							)::float * 100
						)
						ELSE 0
					END as interest_score,
					CASE
						WHEN ABS(ui.age - p.age) <= 5 THEN 100
						WHEN ABS(ui.age - p.age) <= 10 THEN 70
						WHEN ABS(ui.age - p.age) <= 15 THEN 40
						ELSE 0
					END as age_score,
					CASE
						WHEN ui.looking_for = p.looking_for THEN 100
						ELSE 0
					END as intention_score
				FROM profiles p
				CROSS JOIN user_interests ui
				LEFT JOIN matches m ON 
					(m.user_id_1 = p.user_id AND m.user_id_2 = $1) OR
					(m.user_id_2 = p.user_id AND m.user_id_1 = $1)
				WHERE p.user_id != $1
				AND p.location = ui.location
				AND (
					m.id IS NULL 
					OR (m.status = 'pending' AND m.user_id_1 != $1)
				)
				AND NOT EXISTS (
					SELECT 1 FROM matches m2 
					WHERE m2.status = 'dismissed'
					AND (
						(m2.user_id_1 = $1 AND m2.user_id_2 = p.user_id)
						OR 
						(m2.user_id_2 = $1 AND m2.user_id_1 = p.user_id)
					)
				)
			)
			SELECT 
				user_id,
				name,
				bio,
				interests,
				location,
				looking_for,
				age,
				occupation,
				profile_picture_url,
				has_liked_me,
				interest_score,
				age_score,
				intention_score,
				(interest_score * 0.5 + age_score * 0.3 + intention_score * 0.2) as match_score
			FROM match_calculations
			WHERE (interest_score * 0.5 + age_score * 0.3 + intention_score * 0.2) >= 30
			ORDER BY has_liked_me DESC, match_score DESC
			LIMIT 10
		`, userID)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type PotentialMatch struct {
			UserID            int      `json:"user_id"`
			Name              string   `json:"name"`
			Bio               string   `json:"bio"`
			Interests         []string `json:"interests"`
			Location          string   `json:"location"`
			LookingFor        string   `json:"looking_for"`
			Age               int      `json:"age"`
			Occupation        string   `json:"occupation"`
			ProfilePictureURL string   `json:"profile_picture_url"`
			HasLikedMe        bool     `json:"has_liked_me"`
			InterestScore     float64  `json:"interest_score"`
			AgeScore          float64  `json:"age_score"`
			IntentionScore    float64  `json:"intention_score"`
			MatchScore        float64  `json:"match_score"`
		}

		var potentialMatches []PotentialMatch
		for rows.Next() {
			var match PotentialMatch
			var interestsJSON []byte

			err := rows.Scan(
				&match.UserID,
				&match.Name,
				&match.Bio,
				&interestsJSON,
				&match.Location,
				&match.LookingFor,
				&match.Age,
				&match.Occupation,
				&match.ProfilePictureURL,
				&match.HasLikedMe,
				&match.InterestScore,
				&match.AgeScore,
				&match.IntentionScore,
				&match.MatchScore,
			)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			err = json.Unmarshal(interestsJSON, &match.Interests)
			if err != nil {
				http.Error(w, "Failed to decode interests", http.StatusInternalServerError)
				return
			}

			potentialMatches = append(potentialMatches, match)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(potentialMatches)
	}
}

func LikeUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		likedUserID := vars["id"]

		// Check if users are already connected
		var existingConnection int
		err = db.QueryRow(`
			SELECT 1 
			FROM matches 
			WHERE ((user_id_1 = $1 AND user_id_2 = $2) OR (user_id_1 = $2 AND user_id_2 = $1))
			AND status = 'connected'
		`, userID, likedUserID).Scan(&existingConnection)
		
		if err != sql.ErrNoRows {
			if err == nil {
				http.Error(w, "Users are already connected", http.StatusBadRequest)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Check if there's already a match from the other user
		var existingMatch struct {
			ID     int
			Status string
		}
		err = db.QueryRow(`
			SELECT id, status 
			FROM matches 
			WHERE user_id_1 = $1 AND user_id_2 = $2
			AND status = 'pending'
		`, likedUserID, userID).Scan(&existingMatch.ID, &existingMatch.Status)

		if err != nil && err != sql.ErrNoRows {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var isMatch bool
		if err == sql.ErrNoRows {
			// No existing match, create a new pending match
			_, err = db.Exec(`
				INSERT INTO matches (user_id_1, user_id_2, status)
				VALUES ($1, $2, 'pending')
				ON CONFLICT (user_id_1, user_id_2) DO NOTHING
			`, userID, likedUserID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			isMatch = false
		} else {
			// There's a pending match from the other user, update it to connected
			_, err = db.Exec(`
				UPDATE matches 
				SET status = 'connected', updated_at = CURRENT_TIMESTAMP
				WHERE id = $1
			`, existingMatch.ID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			isMatch = true
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"isMatch": isMatch})
	}
}

func DismissMatchHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		dismissedUserID := vars["id"]

		_, err = db.Exec(`
			INSERT INTO matches (user_id_1, user_id_2, status)
			VALUES ($1, $2, 'dismissed')
			ON CONFLICT (user_id_1, user_id_2) DO NOTHING
		`, userID, dismissedUserID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func DisconnectMatchHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		matchID := vars["id"]

		_, err = db.Exec(`
			UPDATE matches 
			SET status = 'disconnected', updated_at = CURRENT_TIMESTAMP
			WHERE id = $1 AND (user_id_1 = $2 OR user_id_2 = $2)
		`, matchID, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func AcceptMatchHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		matchID := vars["id"]

		_, err = db.Exec(`
			UPDATE matches 
			SET status = 'connected', updated_at = CURRENT_TIMESTAMP
			WHERE id = $1 AND user_id_2 = $2 AND status = 'pending'
		`, matchID, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func RejectMatchHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		matchID := vars["id"]

		_, err = db.Exec(`
			UPDATE matches 
			SET status = 'rejected', updated_at = CURRENT_TIMESTAMP
			WHERE id = $1 AND user_id_2 = $2 AND status = 'pending'
		`, matchID, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
