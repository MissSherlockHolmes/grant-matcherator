package main

import (
	"database/sql"
	"log"
	"matchme/handlers"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
	"golang.org/x/exp/rand"
)

func UpdateUserStatusMiddleware(db *sql.DB) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := handlers.GetUserIDFromToken(r)
			if err == nil {
				// Update user status in the database
				_, err = db.Exec(`
					INSERT INTO user_status (user_id, status, last_active)
					VALUES ($1, 'online', CURRENT_TIMESTAMP)
					ON CONFLICT (user_id) DO UPDATE SET
						status = 'online',
						last_active = CURRENT_TIMESTAMP
				`, userID)
				if err != nil {
					//log.printf("Failed to update user status: %v", err)
				}
			}

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// Function to update inactive users' status to "offline"
func updateInactiveUsersStatus(db *sql.DB) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Set users' status to 'offline' if last active time is older than 1 minute
			_, err := db.Exec(`
				UPDATE user_status
				SET status = 'offline'
				WHERE last_active < NOW() - INTERVAL '1 minute' AND status = 'online'
			`)
			if err != nil {
				//log.printf("Error updating user status: %v", err)
			}
		}
	}
}

func main() {
	// Initialize random seed
	rand.Seed(uint64(time.Now().UnixNano()))

	connStr := "user=postgres dbname=matchme password=postgres sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	go updateInactiveUsersStatus(db)

	r := mux.NewRouter()
	r.Use(UpdateUserStatusMiddleware(db))

	// Create uploads directory if it doesn't exist
	os.MkdirAll("uploads", 0755)

	// Serve static files
	fs := http.FileServer(http.Dir("uploads"))
	r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", fs))

	// User routes
	r.HandleFunc("/api/auth/signup", handlers.SignupHandler(db)).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/auth/login", handlers.LoginHandler(db)).Methods("POST", "OPTIONS")

	// Basic user info routes
	r.HandleFunc("/api/users/{id}", handlers.GetUserHandler(db)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/users/{id}/profile", handlers.GetUserProfileHandler(db)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/users/{id}/bio", handlers.GetUserBioHandler(db)).Methods("GET", "OPTIONS")

	// Me routes (shortcuts for authenticated user)
	r.HandleFunc("/api/me", handlers.GetMyBasicInfoHandler(db)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/me/profile", handlers.GetMyProfileHandler(db)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/me/profile", handlers.UpdateProfileHandler(db)).Methods("PUT", "OPTIONS")
	r.HandleFunc("/api/me/bio", handlers.GetMyBioHandler(db)).Methods("GET", "OPTIONS")

	// Recommendations and Connections routes
	r.HandleFunc("/api/recommendations", handlers.GetRecommendationsHandler(db)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/connections", handlers.GetConnectionsHandler(db)).Methods("GET", "OPTIONS")

	// Test data route
	r.HandleFunc("/api/test/generate-users", handlers.GenerateTestDataHandler(db)).Methods("POST", "OPTIONS")

	// Upload route
	r.HandleFunc("/api/upload/profile-picture", handlers.UploadProfilePictureHandler(db)).Methods("POST", "OPTIONS")

	// Match routes
	r.HandleFunc("/api/matches", handlers.GetMatchesHandler(db)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/matches/potential", handlers.GetPotentialMatchesHandler(db)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/matches/like/{id}", handlers.LikeUserHandler(db)).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/matches/dismiss/{id}", handlers.DismissMatchHandler(db)).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/matches/disconnect/{id}", handlers.DisconnectMatchHandler(db)).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/matches/reject/{id}", handlers.RejectMatchHandler(db)).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/matches/accept/{id}", handlers.AcceptMatchHandler(db)).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/matches/preferences", handlers.SetMatchPreferenceHandler(db)).Methods("GET", "OPTIONS", "POST")

	// Notification routes
	r.HandleFunc("/api/notifications", handlers.GetNotificationsHandler(db)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/notifications/mark-matches-read", handlers.MarkMatchesAsReadHandler(db)).Methods("POST")

	// WebSocket route
	r.HandleFunc("/ws/chat/{matchId}", handlers.HandleWebSocket(db))
	r.HandleFunc("/ws/notifications", handlers.HandleNotificationWebSocket())

	// Chat routes
	r.HandleFunc("/api/matches/chats", handlers.GetChatsHandler(db)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/matches/{id}/messages", handlers.GetChatMessagesHandler(db)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/matches/{id}/messages/read", handlers.MarkMessagesAsReadHandler(db)).Methods("POST", "OPTIONS")

	// Status routes
	r.HandleFunc("/api/status/update", handlers.UpdateUserStatusHandler(db)).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/status/{id}", handlers.GetUserStatusHandler(db)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/notifications/mark-read", handlers.MarkMessagesAsReadHandler(db)).Methods("POST", "OPTIONS")

	// CORS configuration
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:8080", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		Debug:            false,
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, c.Handler(r)))
}
