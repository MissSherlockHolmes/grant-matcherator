package user

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"matcherator/backend/handlers/auth"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

// GetUserHandler returns basic user information
func GetUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		vars := mux.Vars(r)
		userID := vars["id"]

		// Get requesting user's ID from token
		requestingUserID, err := auth.GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check if user is authorized to view this profile
		if !IsUserAuthorized(db, requestingUserID, userID) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var user BasicUserResponse
		err = db.QueryRow(SelectBasicUserQuery, userID).Scan(
			&user.ID,
			&user.OrganizationName,
			&user.ProfilePictureURL,
		)

		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(user)
	}
}

// GetFullUserHandler returns complete user information
func GetFullUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		vars := mux.Vars(r)
		userID := vars["id"]

		// Get requesting user's ID from token
		requestingUserID, err := auth.GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check if user is authorized to view this profile
		if !IsUserAuthorized(db, requestingUserID, userID) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var user MatchingUser
		err = db.QueryRow(SelectUserQuery, userID).Scan(
			&user.Role,
			&user.ID,
			&user.Email,
			&user.OrganizationName,
			&user.ProfilePictureURL,
			&user.MissionStatement,
			&user.State,
			&user.City,
			&user.ZIPCode,
			&user.EIN,
			&user.Language,
			&user.ApplicantType,
			pq.Array(&user.Sectors),
			pq.Array(&user.TargetGroups),
			&user.ProjectStage,
			&user.WebsiteURL,
			&user.ContactEmail,
			&user.ChatOptIn,
			&user.Location,
			&user.Description,
		)

		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Get additional profile data based on user role
		if user.Role == "recipient" {
			var recipientData RecipientData
			err = db.QueryRow(SelectRecipientQuery, userID).Scan(
				pq.Array(&recipientData.Needs),
				&recipientData.BudgetRequested,
				&recipientData.TeamSize,
				&recipientData.Timeline,
				&recipientData.PriorFunding,
			)
			if err != nil && err != sql.ErrNoRows {
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			user.Description = &recipientData.Timeline
		} else if user.Role == "provider" {
			var providerData ProviderData
			err = db.QueryRow(SelectProviderQuery, userID).Scan(
				&providerData.FundingType,
				&providerData.AmountOffered,
				&providerData.RegionScope,
				&providerData.LocationNotes,
				&providerData.EligibilityNotes,
				&providerData.Deadline,
				&providerData.ApplicationLink,
			)
			if err != nil && err != sql.ErrNoRows {
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			user.Description = &providerData.EligibilityNotes
		}

		json.NewEncoder(w).Encode(user)
	}
}

// GetMyBasicInfoHandler returns the authenticated user's basic information
func GetMyBasicInfoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, err := auth.GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var user BasicUserResponse
		err = db.QueryRow(SelectBasicUserQuery, userID).Scan(
			&user.ID,
			&user.OrganizationName,
			&user.ProfilePictureURL,
		)

		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(user)
	}
}

// GetUsersHandler returns a list of all users
func GetUsersHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get user ID from token
		_, err := auth.GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		rows, err := db.Query(`
			SELECT id, email, role
			FROM users
			ORDER BY id
		`)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var users []User
		for rows.Next() {
			var user User
			err := rows.Scan(&user.ID, &user.Email, &user.Role)
			if err != nil {
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			users = append(users, user)
		}

		if err = rows.Err(); err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(users)
	}
}
