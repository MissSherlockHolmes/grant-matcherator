package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// Predefined arrays for consistent test data
var sectors = []string{
	"Education", "Healthcare", "Environment", "Arts & Culture",
	"Social Services", "Technology", "Economic Development",
	"Youth Development", "Community Development", "Research",
}

var targetGroups = []string{
	"Children", "Youth", "Elderly", "Veterans", "Immigrants",
	"Low-income", "Disabilities", "Women", "Minorities",
	"LGBTQ+", "Students", "Unemployed",
}

var states = []string{
	"AL", "AK", "AZ", "AR", "CA", "CO", "CT", "DE", "FL", "GA",
	"HI", "ID", "IL", "IN", "IA", "KS", "KY", "LA", "ME", "MD",
	"MA", "MI", "MN", "MS", "MO", "MT", "NE", "NV", "NH", "NJ",
	"NM", "NY", "NC", "ND", "OH", "OK", "OR", "PA", "RI", "SC",
	"SD", "TN", "TX", "UT", "VT", "VA", "WA", "WV", "WI", "WY",
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
			email := gofakeit.Email()
			organizationName := gofakeit.Company()

			// Hash a simple password
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte("testpass123"), bcrypt.DefaultCost)
			if err != nil {
				continue
			}

			// Randomly assign role (provider or recipient)
			role := "recipient"
			if gofakeit.Bool() {
				role = "provider"
			}

			// Insert user
			var userID int
			err = tx.QueryRow(`
				INSERT INTO users (email, password_hash, role)
				VALUES ($1, $2, $3)
				RETURNING id
			`, email, string(hashedPassword), role).Scan(&userID)
			if err != nil {
				continue
			}

			// Generate random sectors and target groups
			numSectors := gofakeit.Number(1, 3)
			numTargetGroups := gofakeit.Number(1, 3)
			selectedSectors := make([]string, numSectors)
			selectedTargetGroups := make([]string, numTargetGroups)

			for j := 0; j < numSectors; j++ {
				selectedSectors[j] = sectors[gofakeit.Number(0, len(sectors)-1)]
			}
			for j := 0; j < numTargetGroups; j++ {
				selectedTargetGroups[j] = targetGroups[gofakeit.Number(0, len(targetGroups)-1)]
			}

			// Create profile for the user
			_, err = tx.Exec(`
				INSERT INTO profiles (
					user_id, organization_name, mission_statement,
					state, city, zip_code, sectors, target_groups,
					website_url, contact_email
				)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			`,
				userID,
				organizationName,
				gofakeit.Sentence(10),
				states[gofakeit.Number(0, len(states)-1)],
				gofakeit.City(),
				gofakeit.Zip(),
				pq.Array(selectedSectors),
				pq.Array(selectedTargetGroups),
				fmt.Sprintf("https://www.%s.org", gofakeit.DomainName()),
				email,
			)
			if err != nil {
				continue
			}

			// Create provider or recipient specific data
			if role == "provider" {
				_, err = tx.Exec(`
					INSERT INTO provider_data (
						user_id, funding_type, amount_offered,
						region_scope, location_notes, eligibility_notes,
						deadline, application_link
					)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
				`,
					userID,
					gofakeit.Word(),
					gofakeit.Price(10000, 1000000),
					gofakeit.State(),
					gofakeit.Sentence(5),
					gofakeit.Sentence(5),
					gofakeit.DateRange(time.Now(), time.Now().AddDate(0, 3, 0)),
					fmt.Sprintf("https://www.%s.org/apply", gofakeit.DomainName()),
				)
			} else {
				_, err = tx.Exec(`
					INSERT INTO recipient_data (
						user_id, needs, budget_requested,
						team_size, timeline, prior_funding
					)
					VALUES ($1, $2, $3, $4, $5, $6)
				`,
					userID,
					pq.Array(selectedTargetGroups), // Using target groups as needs
					gofakeit.Price(5000, 500000),
					gofakeit.Number(1, 100),
					gofakeit.Word(),
					gofakeit.Bool(),
				)
			}
			if err != nil {
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
