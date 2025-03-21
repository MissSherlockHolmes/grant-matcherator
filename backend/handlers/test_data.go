package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var firstNames = []string{
	"Alice", "Bob", "Charlie", "Diana", "Erik",
	"Fiona", "George", "Hannah", "Ian", "Julia",
	"Riona", "Reorge", "Rannah", "Ran", "Rulia",
	"Tiona", "Teorge", "Tannah", "Tan", "Tulia",
	"Siona", "Jeorge", "Sannah", "Jan", "Sulia",
}

var lastNames = []string{
	"Smith", "Johnson", "Williams", "Brown", "Jones",
	"Garcia", "Miller", "Davis", "Rodriguez", "Martinez",
}

var bios = []string{
	"Love to travel and explore new places!",
	"Coffee enthusiast and book lover",
	"Passionate about photography and nature",
	"Tech geek who loves coding",
	"Fitness fanatic and outdoor adventurer",
}

var interests = [][]string{
	{"Reading", "Travel", "Photography"},
	{"Gaming", "Technology", "Movies"},
	{"Sports", "Fitness", "Outdoors"},
	{"Cooking", "Music", "Art"},
	{"Dancing", "Writing", "Fashion"},
}

var locations = []string{
	"Tallinn", "Tartu", "Pärnu", "Narva", "Viljandi", "Kohtla-Järve",
	"Paide", "Rakvere", "Sillamäe", "Maardu", "Paide", "Valga", "Võru",
	"Kuressaare", "Jõhvi",
}

var lookingFor = []string{
	"Friendship", "Relationship", "Casual Dating",
	"Networking", "Chat Buddies", "Activity Partners",
}

var occupations = []string{
	"Software Developer", "Teacher", "Healthcare Professional",
	"Business Professional", "Student", "Artist/Creative",
}

func GenerateTestDataHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get count parameter, default to 10 if not provided
		count := 10
		if countParam := r.URL.Query().Get("count"); countParam != "" {
			parsedCount, err := strconv.Atoi(countParam)
			if err != nil || parsedCount < 1 || parsedCount > 150 {
				http.Error(w, "Count must be between 1 and 150", http.StatusBadRequest)
				return
			}
			count = parsedCount
		}

		// Start generating test users
		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "Could not start generating", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		createdUsers := 0
		for i := 0; i < count; i++ {
			// Generate random user data
			firstName := firstNames[rand.Intn(len(firstNames))]
			lastName := lastNames[rand.Intn(len(lastNames))]
			email := fmt.Sprintf("%s.%s.%d@test.com", firstName, lastName, time.Now().UnixNano())

			// Hash a simple password
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte("testpass123"), bcrypt.DefaultCost)
			if err != nil {
				//log.printf("Error hashing password: %v", err)
				continue
			}

			// Insert user
			var userID int
			err = tx.QueryRow(`
				INSERT INTO users (email, password_hash)
				VALUES ($1, $2)
				RETURNING id
			`, email, string(hashedPassword)).Scan(&userID)
			//log.printf("Test user: %d created", userID)
			if err != nil {
				//log.printf("Error creating test user: %v", err)
				continue
			}

			// Convert interests slice to PostgreSQL array
			selectedInterests := interests[rand.Intn(len(interests))]
			interestsArray := pq.Array(selectedInterests)

			// Create profile for the user
			_, err = tx.Exec(`
				INSERT INTO profiles (
					user_id, name, bio, interests, location,
					looking_for, age, occupation, profile_picture_url
				)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			`,
				userID,
				firstName+" "+lastName,
				bios[rand.Intn(len(bios))],
				interestsArray,
				locations[rand.Intn(len(locations))],
				lookingFor[rand.Intn(len(lookingFor))],
				rand.Intn(42)+18, // Age between 18 and 60
				occupations[rand.Intn(len(occupations))],
				"/placeholder.svg", // No profile picture for test users
			)
			if err != nil {
				//log.printf("Error creating test profile: %v", err)
				continue
			}

			createdUsers++
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "Could not commit transaction", http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"message":       "Test user(s) generated successfully",
			"users_created": createdUsers,
		}

		json.NewEncoder(w).Encode(response)
	}
}
