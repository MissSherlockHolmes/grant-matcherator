package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/lib/pq"
)

type Profile struct {
	UserID            int      `json:"user_id"`
	Name              string   `json:"name"`
	Bio               string   `json:"bio"`
	Interests         []string `json:"interests"`
	Location          string   `json:"location"`
	LookingFor        string   `json:"looking_for"`
	Age               int      `json:"age"`
	Occupation        string   `json:"occupation"`
	ProfilePictureURL string   `json:"profile_picture_url"`
	Email             string   `json:"email"` // Add this field
}

func GetMyProfileHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, err := GetUserIDFromToken(r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}

		var profile Profile
		var email string

		// Fetch email from the users table
		err = db.QueryRow(`
			SELECT email
			FROM users
			WHERE id = $1
		`, userID).Scan(&email)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
			return
		}

		// Fetch profile data from the profiles table
		err = db.QueryRow(`
			SELECT user_id, name, bio, interests, location, looking_for, age, occupation, profile_picture_url
			FROM profiles WHERE user_id = $1
		`, userID).Scan(
			&profile.UserID, &profile.Name, &profile.Bio, pq.Array(&profile.Interests),
			&profile.Location, &profile.LookingFor, &profile.Age,
			&profile.Occupation, &profile.ProfilePictureURL,
		)

		if err == sql.ErrNoRows {
			// Return an empty profile with the email for new users
			profile = Profile{UserID: userID, Email: email}
			json.NewEncoder(w).Encode(profile)
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
			return
		}

		// Include the email in the response
		profile.Email = email

		json.NewEncoder(w).Encode(profile)
	}
}

func UpdateProfileHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, err := GetUserIDFromToken(r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}

		var profile Profile
		if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}

		// Ensure the profile belongs to the authenticated user
		profile.UserID = userID

		// Start transaction
		tx, err := db.Begin()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
			return
		}
		defer tx.Rollback()

		// Update or insert profile
		_, err = tx.Exec(`
			INSERT INTO profiles (
				user_id, name, bio, interests, location, looking_for, 
				age, occupation, profile_picture_url
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (user_id) DO UPDATE SET
				name = EXCLUDED.name,
				bio = EXCLUDED.bio,
				interests = EXCLUDED.interests,
				location = EXCLUDED.location,
				looking_for = EXCLUDED.looking_for,
				age = EXCLUDED.age,
				occupation = EXCLUDED.occupation,
				profile_picture_url = COALESCE(EXCLUDED.profile_picture_url, profiles.profile_picture_url),
				updated_at = CURRENT_TIMESTAMP
		`,
			userID, profile.Name, profile.Bio, pq.Array(profile.Interests),
			profile.Location, profile.LookingFor, profile.Age,
			profile.Occupation, profile.ProfilePictureURL,
		)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update profile"})
			return
		}

		if err = tx.Commit(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to commit transaction"})
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"message": "Profile updated successfully"})
	}
}
