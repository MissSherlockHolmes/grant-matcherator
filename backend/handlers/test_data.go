// Note: To generate test data, use:
// curl -X POST "http://localhost:8080/api/test/generate-users?count=5" -H "Content-Type: application/json"

package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"matcherator/backend/handlers/auth"
	"matcherator/backend/handlers/user_status"
	"matcherator/backend/services/matches"
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

var locations = []string{
	"North America", "South America", "Europe", "Asia", "Africa",
	"Oceania", "Caribbean", "Central America", "Middle East",
	"South Asia", "East Asia", "Southeast Asia", "Sub-Saharan Africa",
	"North Africa", "Scandinavia", "Mediterranean", "Pacific Islands",
}

var fundingTypes = []string{
	"accelerator",
	"angel investors",
	"fellowship",
	"free consulting",
	"funding",
	"funding resources",
	"grant",
	"incubator",
	"loan",
	"loans",
	"pitch",
	"pitch comp",
	"resources",
	"startup program",
}

var languages = []string{
	"English", "Spanish",
}

var applicantTypes = []string{
	"Non-profit", "For-profit", "Government", "Educational Institution",
	"Research Organization", "Community Group", "Individual", "Startup",
	"Social Enterprise", "Cooperative", "Foundation", "Association",
}

var projectStages = []string{
	"Idea Stage", "Pre-seed", "Seed", "Early Stage",
	"Growth Stage", "Scale-up", "Mature", "Expansion",
	"Pilot", "MVP", "Product Development", "Market Testing",
}

var needs = []string{
	"Capital", "Mentorship", "Technical Support", "Marketing",
	"Legal Services", "HR Support", "Infrastructure", "Training",
	"Equipment", "Research", "Development", "Partnerships",
	"Market Access", "Regulatory Compliance", "Branding",
}

var timelines = []string{
	"1-3 months", "3-6 months", "6-12 months", "1-2 years",
	"2-5 years", "5+ years", "Ongoing", "Flexible",
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
			log.Printf("Error starting transaction: %v", err)
			http.Error(w, "Could not start generating", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		createdProviders := 0
		createdRecipients := 0
		failedAttempts := 0
		for i := 0; i < count; i++ {
			// Create a savepoint for this user
			_, err := tx.Exec(fmt.Sprintf("SAVEPOINT user_%d", i))
			if err != nil {
				log.Printf("[User %d] Error creating savepoint: %v", i+1, err)
				failedAttempts++
				continue
			}

			log.Printf("[User %d] Starting user creation", i+1)
			// Generate random user data
			email := gofakeit.Email()
			organizationName := gofakeit.Company()
			log.Printf("[User %d] Generated email: %s, organization: %s", i+1, email, organizationName)

			// Hash a simple password
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte("testpass123"), bcrypt.DefaultCost)
			if err != nil {
				log.Printf("[User %d] Error hashing password: %v", i+1, err)
				tx.Exec(fmt.Sprintf("ROLLBACK TO SAVEPOINT user_%d", i))
				failedAttempts++
				continue
			}

			// Randomly assign role (provider or recipient)
			role := "recipient"
			if gofakeit.Bool() {
				role = "provider"
			}
			log.Printf("[User %d] Assigned role: %s", i+1, role)

			// Insert user with initial status
			var userID int
			err = tx.QueryRow(`
				INSERT INTO users (email, password_hash, role, status)
				VALUES ($1, $2, $3, $4)
				RETURNING id
			`, email, string(hashedPassword), role, "inactive").Scan(&userID)
			if err != nil {
				log.Printf("[User %d] Error inserting user: %v", i+1, err)
				if pqErr, ok := err.(*pq.Error); ok {
					log.Printf("[User %d] Postgres error details: %s, code: %s, constraint: %s", i+1, pqErr.Detail, pqErr.Code, pqErr.Constraint)
				}
				tx.Exec(fmt.Sprintf("ROLLBACK TO SAVEPOINT user_%d", i))
				failedAttempts++
				continue
			}
			log.Printf("[User %d] Created user with ID: %d", i+1, userID)

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
					user_id, organization_name, profile_picture_url,
					mission_statement, location, state, city, zip_code,
					ein, language, applicant_type,
					sectors, target_groups, project_stage,
					website_url, contact_email, chat_opt_in
				)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
			`,
				userID,
				organizationName,
				fmt.Sprintf("https://www.%s.org/profile.jpg", gofakeit.DomainName()),
				gofakeit.Sentence(10),
				locations[gofakeit.Number(0, len(locations)-1)],
				states[gofakeit.Number(0, len(states)-1)],
				gofakeit.City(),
				gofakeit.Zip(),
				fmt.Sprintf("%d-%d", gofakeit.Number(10, 99), gofakeit.Number(1000, 9999)),
				languages[gofakeit.Number(0, len(languages)-1)],
				applicantTypes[gofakeit.Number(0, len(applicantTypes)-1)],
				pq.Array(selectedSectors),
				pq.Array(selectedTargetGroups),
				projectStages[gofakeit.Number(0, len(projectStages)-1)],
				fmt.Sprintf("https://www.%s.org", gofakeit.DomainName()),
				email,
				gofakeit.Bool())
			if err != nil {
				log.Printf("Error creating profile: %v", err)
				tx.Exec(fmt.Sprintf("ROLLBACK TO SAVEPOINT user_%d", i))
				continue
			}
			log.Printf("Created profile for user %d", userID)

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
					fundingTypes[gofakeit.Number(0, len(fundingTypes)-1)],
					gofakeit.Price(10000, 1000000),
					states[gofakeit.Number(0, len(states)-1)],
					gofakeit.Sentence(5),
					gofakeit.Sentence(5),
					gofakeit.DateRange(time.Now(), time.Now().AddDate(0, 3, 0)),
					fmt.Sprintf("https://www.%s.org/apply", gofakeit.DomainName()),
				)
			} else {
				// Generate random needs
				numNeeds := gofakeit.Number(1, 3)
				selectedNeeds := make([]string, numNeeds)
				for j := 0; j < numNeeds; j++ {
					selectedNeeds[j] = needs[gofakeit.Number(0, len(needs)-1)]
				}

				_, err = tx.Exec(`
					INSERT INTO recipient_data (
						user_id, needs, budget_requested,
						team_size, timeline, prior_funding
					)
					VALUES ($1, $2, $3, $4, $5, $6)
				`,
					userID,
					pq.Array(selectedNeeds),
					gofakeit.Price(5000, 500000),
					gofakeit.Number(1, 100),
					timelines[gofakeit.Number(0, len(timelines)-1)],
					gofakeit.Bool(),
				)
			}
			if err != nil {
				log.Printf("Error creating role-specific data: %v", err)
				tx.Exec(fmt.Sprintf("ROLLBACK TO SAVEPOINT user_%d", i))
				continue
			}
			log.Printf("Created role-specific data for user %d", userID)

			// Generate and store token for the user
			token, err := auth.GenerateToken(userID)
			if err != nil {
				log.Printf("Error generating token: %v", err)
				tx.Exec(fmt.Sprintf("ROLLBACK TO SAVEPOINT user_%d", i))
				continue
			}

			_, err = tx.Exec(`
				INSERT INTO tokens (user_id, token, expires_at)
				VALUES ($1, $2, $3)
			`, userID, token, time.Now().Add(time.Hour*24))
			if err != nil {
				log.Printf("Error storing token: %v", err)
				tx.Exec(fmt.Sprintf("ROLLBACK TO SAVEPOINT user_%d", i))
				continue
			}
			log.Printf("Generated and stored token for user %d", userID)

			// Update user status after creating all data
			if err := user_status.UpdateUserStatus(tx, strconv.Itoa(userID)); err != nil {
				log.Printf("Error updating user status: %v", err)
				tx.Exec(fmt.Sprintf("ROLLBACK TO SAVEPOINT user_%d", i))
				continue
			}
			log.Printf("Updated status for user %d", userID)

			// Release the savepoint since everything succeeded
			_, err = tx.Exec(fmt.Sprintf("RELEASE SAVEPOINT user_%d", i))
			if err != nil {
				log.Printf("Error releasing savepoint: %v", err)
				tx.Exec(fmt.Sprintf("ROLLBACK TO SAVEPOINT user_%d", i))
				continue
			}

			if role == "provider" {
				createdProviders++
			} else {
				createdRecipients++
			}
			log.Printf("[User %d] Successfully created user", i+1)
		}

		if err = tx.Commit(); err != nil {
			log.Printf("Transaction commit error: %v", err)
			log.Printf("Rolling back transaction due to error")
			tx.Rollback()
			http.Error(w, fmt.Sprintf("Could not commit transaction: %v", err), http.StatusInternalServerError)
			return
		}

		// Recalculate matches for all users
		if err := matches.RecalculateMatchesForAllUsers(db); err != nil {
			log.Printf("Error recalculating matches: %v", err)
			// Don't return error here as the users were still created successfully
		}

		log.Printf("Summary: Created %d providers and %d recipients, Failed attempts: %d", createdProviders, createdRecipients, failedAttempts)

		response := struct {
			Message      string `json:"message"`
			UsersCreated int    `json:"users_created"`
			Providers    int    `json:"providers"`
			Recipients   int    `json:"recipients"`
		}{
			Message:      "Test user(s) generated successfully",
			UsersCreated: createdProviders + createdRecipients,
			Providers:    createdProviders,
			Recipients:   createdRecipients,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding response: %v", err)
			http.Error(w, "Error generating response", http.StatusInternalServerError)
			return
		}
	}
}
