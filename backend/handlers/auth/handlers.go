package auth

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// [AI_DEPENDENCIES_START]
// DEPENDENCY_MAP:
// {
//   "external": ["database/sql", "encoding/json", "net/http", "golang.org/x/crypto/bcrypt"],
//   "internal": ["auth.GenerateToken"],
//   "usage": ["main.go"]
// }
// [AI_DEPENDENCIES_END]

// [AI_MODELS_START]
// MODELS:
// {
//   "User": {
//     "fields": ["ID", "Email", "Password", "Role", "CreatedAt"],
//     "json_tags": true,
//     "omitempty": false
//   },
//   "LoginResponse": {
//     "fields": ["ID", "Email", "Token"],
//     "json_tags": true,
//     "omitempty": false
//   }
// }
// [AI_MODELS_END]

type User struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"password,omitempty"`
	Role      string    `json:"role"` // "provider" or "recipient"
	CreatedAt time.Time `json:"created_at"`
}

type LoginResponse struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Token string `json:"token"`
}

// SignupHandler handles user registration
// Used by: /api/auth/signup
// Dependencies: GenerateToken
// Response: LoginResponse
func SignupHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var signupRequest struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			Role     string `json:"role"`
		}

		if err := json.NewDecoder(r.Body).Decode(&signupRequest); err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}

		// Validate role
		if signupRequest.Role != "provider" && signupRequest.Role != "recipient" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid role. Must be 'provider' or 'recipient'"})
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(signupRequest.Password), bcrypt.DefaultCost)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Error hashing password"})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
			return
		}
		defer tx.Rollback()

		// Insert into users table with role
		query := `INSERT INTO users (email, password_hash, role) VALUES ($1, $2, $3) RETURNING id`
		var userID int
		err = tx.QueryRow(query, signupRequest.Email, string(hashedPassword), signupRequest.Role).Scan(&userID)
		if err != nil {
			if strings.Contains(err.Error(), "unique constraint") {
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]string{"error": "Email already exists"})
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Error creating user"})
			return
		}

		// Create initial profile based on role
		if signupRequest.Role == "recipient" {
			_, err = tx.Exec(`
				INSERT INTO recipient_profiles (
					user_id, mission_statement, needs, budget_requested,
					team_size, timeline, prior_funding
				) VALUES ($1, '', '{}', 0, 0, '', false)
			`, userID)
		} else {
			_, err = tx.Exec(`
				INSERT INTO provider_profiles (
					user_id, funding_type, amount_offered, region_scope,
					location_notes, eligibility_notes, deadline,
					application_link, description
				) VALUES ($1, '', '', '', '', '', '', '', '')
			`, userID)
		}

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Error creating profile"})
			return
		}

		token, err := GenerateToken(userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Error generating token"})
			return
		}

		_, err = tx.Exec(`
			INSERT INTO tokens (user_id, token, expires_at)
			VALUES ($1, $2, $3)
		`, userID, token, time.Now().Add(time.Hour*24))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Error storing token"})
			return
		}

		_, err = tx.Exec(`
			INSERT INTO user_status (user_id, status, last_active)
			VALUES ($1, 'online', CURRENT_TIMESTAMP)
			ON CONFLICT (user_id) DO UPDATE SET
				status = 'online',
				last_active = CURRENT_TIMESTAMP
		`, userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Error initializing user status"})
			return
		}

		if err = tx.Commit(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Error completing registration"})
			return
		}

		response := LoginResponse{
			ID:    userID,
			Email: signupRequest.Email,
			Token: token,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// LoginHandler handles user authentication
// Used by: /api/auth/login
// Dependencies: GenerateToken
// Response: LoginResponse
func LoginHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var loginRequest struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&loginRequest); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var user User
		var hashedPassword string
		query := `SELECT id, email, password_hash FROM users WHERE email = $1`
		err := db.QueryRow(query, loginRequest.Email).Scan(&user.ID, &user.Email, &hashedPassword)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Invalid credentials", http.StatusUnauthorized)
				return
			}
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(loginRequest.Password))
		if err != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		token, err := GenerateToken(user.ID)
		if err != nil {
			http.Error(w, "Error generating token", http.StatusInternalServerError)
			return
		}

		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		_, err = tx.Exec(`
			INSERT INTO tokens (user_id, token, expires_at)
			VALUES ($1, $2, $3)
		`, user.ID, token, time.Now().Add(time.Hour*24))
		if err != nil {
			tx.Rollback()
			http.Error(w, "Error storing token", http.StatusInternalServerError)
			return
		}

		_, err = tx.Exec(`
			INSERT INTO user_status (user_id, status, last_active)
			VALUES ($1, 'online', CURRENT_TIMESTAMP)
			ON CONFLICT (user_id) 
			DO UPDATE SET 
				status = 'online',
				last_active = CURRENT_TIMESTAMP
		`, user.ID)
		if err != nil {
			tx.Rollback()
			http.Error(w, "Error updating user status", http.StatusInternalServerError)
			return
		}

		if err = tx.Commit(); err != nil {
			http.Error(w, "Error completing login", http.StatusInternalServerError)
			return
		}

		response := LoginResponse{
			ID:    user.ID,
			Email: user.Email,
			Token: token,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
