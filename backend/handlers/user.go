package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

type BasicUserResponse struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	ProfilePictureURL string `json:"profile_picture_url"`
}

type ProfileResponse struct {
	ID         int      `json:"id"`
	Bio        string   `json:"bio"`
	Interests  []string `json:"interests"`
	LookingFor string   `json:"looking_for"`
	Occupation string   `json:"occupation"`
}

type BioResponse struct {
	ID       int    `json:"id"`
	Age      int    `json:"age"`
	Location string `json:"location"`
}

type User struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"password,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type LoginResponse struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Token string `json:"token"`
}

// GetUserHandler returns only the user's name and profile picture URL
func GetUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get the requesting user's ID from the token
		requestingUserID, err := GetUserIDFromToken(r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}

		// Get the target user ID from the request
		vars := mux.Vars(r)
		targetUserID := vars["id"]

		// Check if the requesting user is matched with the target user
		if !isUserAuthorized(db, requestingUserID, targetUserID) {
			w.WriteHeader(http.StatusNotFound) // Pretend the user does not exist
			json.NewEncoder(w).Encode(map[string]string{"error": "User not found"})
			return
		}

		var response BasicUserResponse
		err = db.QueryRow(`
			SELECT u.id, COALESCE(p.name, ''), COALESCE(p.profile_picture_url, '')
			FROM users u
			LEFT JOIN profiles p ON u.id = p.user_id
			WHERE u.id = $1
		`, targetUserID).Scan(&response.ID, &response.Name, &response.ProfilePictureURL)

		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "User not found"})
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
			return
		}

		json.NewEncoder(w).Encode(response)
	}
}

// GetFullUserHandler returns the full user information
func GetFullUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		vars := mux.Vars(r)
		id := vars["id"]

		var user User
		err := db.QueryRow("SELECT id, email, created_at FROM users WHERE id = $1", id).Scan(&user.ID, &user.Email, &user.CreatedAt)
		if err != nil {
			if err == sql.ErrNoRows {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{"error": "User not found"})
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
			return
		}

		json.NewEncoder(w).Encode(user)
	}
}

// GetUserProfileHandler returns the user's "about me" information
func GetUserProfileHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		vars := mux.Vars(r)
		userID := vars["id"]

		var response ProfileResponse
		var interestsJSON string // Temporary variable to hold the JSON string

		err := db.QueryRow(`
		SELECT p.user_id, p.bio, array_to_json(p.interests)::text, p.looking_for, p.occupation
		FROM profiles p
		WHERE p.user_id = $1
	`, userID).Scan(&response.ID, &response.Bio, &interestsJSON, &response.LookingFor, &response.Occupation)

		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Profile not found"})
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
			return
		}

		// Unmarshal the JSON string into a []string
		var interests []string
		if err := json.Unmarshal([]byte(interestsJSON), &interests); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to parse interests"})
			return
		}

		// Assign the unmarshalled interests to the response
		response.Interests = interests

		json.NewEncoder(w).Encode(response)
	}
}

// GetUserBioHandler returns the user's biographical data
func GetUserBioHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		vars := mux.Vars(r)
		userID := vars["id"]

		var response BioResponse
		err := db.QueryRow(`
			SELECT p.user_id, p.age, p.location
			FROM profiles p
			WHERE p.user_id = $1
		`, userID).Scan(&response.ID, &response.Age, &response.Location)

		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Bio not found"})
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
			return
		}

		json.NewEncoder(w).Encode(response)
	}
}

func SignupHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var user User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
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

		query := `INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`
		err = tx.QueryRow(query, user.Email, string(hashedPassword)).Scan(&user.ID)
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

		token, err := generateToken(user.ID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Error generating token"})
			return
		}

		_, err = tx.Exec(`
			INSERT INTO tokens (user_id, token, expires_at)
			VALUES ($1, $2, $3)
		`, user.ID, token, time.Now().Add(time.Hour*24))
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
		`, user.ID)
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
			ID:    user.ID,
			Email: user.Email,
			Token: token,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

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

		token, err := generateToken(user.ID)
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

func generateToken(userID int) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	return token.SignedString([]byte("your-secret-key"))
}

func GetUserIDFromToken(r *http.Request) (int, error) {
	tokenString := r.Header.Get("Authorization")
	if tokenString == "" {
		return 0, fmt.Errorf("no token provided")
	}

	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte("your-secret-key"), nil
	})

	if err != nil {
		return 0, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return 0, fmt.Errorf("invalid token claims")
	}

	return int(claims["user_id"].(float64)), nil
}

// GetMyBasicInfoHandler returns the authenticated user's basic info
func GetMyBasicInfoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, err := GetUserIDFromToken(r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}

		var response BasicUserResponse
		err = db.QueryRow(`
			SELECT u.id, COALESCE(p.name, ''), COALESCE(p.profile_picture_url, '')
			FROM users u
			LEFT JOIN profiles p ON u.id = p.user_id
			WHERE u.id = $1
		`, userID).Scan(&response.ID, &response.Name, &response.ProfilePictureURL)

		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "User not found"})
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
			return
		}

		json.NewEncoder(w).Encode(response)
	}
}

// GetMyBioHandler returns the authenticated user's biographical data
func GetMyBioHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, err := GetUserIDFromToken(r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}

		var response BioResponse
		err = db.QueryRow(`
			SELECT p.user_id, p.age, p.location
			FROM profiles p
			WHERE p.user_id = $1
		`, userID).Scan(&response.ID, &response.Age, &response.Location)

		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Bio not found"})
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
			return
		}

		json.NewEncoder(w).Encode(response)
	}
}

func isUserAuthorized(db *sql.DB, requestingUserID int, targetUserID string) bool {
	var exists bool
	err := db.QueryRow(`
		WITH user_interests AS (
			SELECT 
				interests,
				looking_for,
				age,
				location
			FROM profiles 
			WHERE user_id = $1
		)
		SELECT EXISTS (
			SELECT 1 FROM matches 
			WHERE (user_id_1 = $1 AND user_id_2 = $2) OR (user_id_1 = $2 AND user_id_2 = $1)
			UNION
			SELECT 1 FROM profiles p
			CROSS JOIN user_interests ui
			LEFT JOIN matches m ON 
				(m.user_id_1 = p.user_id AND m.user_id_2 = $1) OR
				(m.user_id_2 = p.user_id AND m.user_id_1 = $1)
			WHERE p.user_id = $2
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
	`, requestingUserID, targetUserID).Scan(&exists)

	if err != nil {
		return false // Treat DB errors as unauthorized
	}
	return exists
}
