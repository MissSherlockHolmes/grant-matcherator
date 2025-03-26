package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
	"golang.org/x/exp/rand"

	"matcherator/backend/handlers"
	"matcherator/backend/handlers/auth"
	"matcherator/backend/handlers/chat"
	"matcherator/backend/handlers/connection"
	"matcherator/backend/handlers/media"
	"matcherator/backend/handlers/notifications"
	"matcherator/backend/handlers/profile"
	"matcherator/backend/handlers/status"
	"matcherator/backend/handlers/user"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Check required environment variables
	requiredEnvVars := []string{"DATABASE_URL", "JWT_SECRET_KEY"}
	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			log.Fatalf("Required environment variable %s is not set", envVar)
		}
		log.Printf("Environment variable %s is set", envVar)
	}

	// Initialize random seed
	rand.Seed(uint64(time.Now().UnixNano()))

	// Initialize database connection
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create router
	r := mux.NewRouter()

	// CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	})

	// Public routes (no auth required)
	r.HandleFunc("/api/auth/signup", auth.SignupHandler(db)).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/auth/login", auth.LoginHandler(db)).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/test/generate-users", handlers.GenerateTestDataHandler(db)).Methods("POST", "OPTIONS")

	// Create a subrouter for protected routes
	protected := r.PathPrefix("/api").Subrouter()
	protected.Use(auth.AuthMiddleware)

	// User routes
	protected.HandleFunc("/users", user.GetUsersHandler(db)).Methods("GET", "OPTIONS")
	protected.HandleFunc("/users/{id}", user.GetUserHandler(db)).Methods("GET", "OPTIONS")
	protected.HandleFunc("/users/{id}/full", user.GetFullUserHandler(db)).Methods("GET", "OPTIONS")
	protected.HandleFunc("/users/{id}/profile", profile.GetUserProfileHandler(db)).Methods("GET", "OPTIONS")
	protected.HandleFunc("/users/{id}/bio", profile.GetUserBioHandler(db)).Methods("GET", "OPTIONS")

	// Me routes
	protected.HandleFunc("/me", user.GetMyBasicInfoHandler(db)).Methods("GET", "OPTIONS")
	protected.HandleFunc("/me/profile", profile.GetUserProfileHandler(db)).Methods("GET", "OPTIONS")
	protected.HandleFunc("/me/profile", profile.UpdateProfileHandler(db)).Methods("PUT", "OPTIONS")
	protected.HandleFunc("/me/bio", profile.GetMyBioHandler(db)).Methods("GET", "OPTIONS")

	// Upload routes
	protected.HandleFunc("/upload/profile-picture", media.UploadProfilePictureHandler(db)).Methods("POST", "OPTIONS")
	protected.HandleFunc("/upload/profile-picture", media.DeleteProfilePictureHandler(db)).Methods("DELETE", "OPTIONS")

	// Connections and Matching routes
	protected.HandleFunc("/connections", connection.GetConnectionsHandler(db)).Methods("GET", "OPTIONS")
	protected.HandleFunc("/connections", connection.CreateConnectionHandler(db)).Methods("POST", "OPTIONS")
	protected.HandleFunc("/connections/{id}", connection.DeleteConnectionHandler(db)).Methods("DELETE", "OPTIONS")
	protected.HandleFunc("/potential-matches", connection.GetPotentialMatchesHandler(db)).Methods("GET", "OPTIONS")
	protected.HandleFunc("/potential-matches/recalculate", connection.RecalculateMatchesHandler(db)).Methods("POST", "OPTIONS")
	protected.HandleFunc("/matches/dismiss/{id}", connection.DismissMatchHandler(db)).Methods("DELETE", "OPTIONS")

	// Notification routes
	protected.HandleFunc("/notifications", notifications.GetNotificationsHandler(db)).Methods("GET", "OPTIONS")
	protected.HandleFunc("/notifications/read", notifications.MarkNotificationsAsReadHandler(db)).Methods("POST", "OPTIONS")
	r.HandleFunc("/ws/notifications", notifications.HandleNotificationWebSocket())

	// Chat routes
	protected.HandleFunc("/chat/preferences", chat.GetChatPreferencesHandler(db)).Methods("GET", "OPTIONS")
	protected.HandleFunc("/chat/preferences", chat.UpdateChatPreferencesHandler(db)).Methods("PUT", "OPTIONS")
	protected.HandleFunc("/chat", chat.GetChatsHandler(db)).Methods("GET", "OPTIONS")
	protected.HandleFunc("/chat/{id}/messages", chat.GetChatMessagesHandler(db)).Methods("GET", "OPTIONS")
	protected.HandleFunc("/chat/{id}/messages/read", chat.MarkMessagesAsReadHandler(db)).Methods("POST", "OPTIONS")
	r.HandleFunc("/ws/chat/{matchId}", chat.HandleWebSocket(db))

	// Status routes
	protected.HandleFunc("/status/{id}", status.GetStatusHandler(db)).Methods("GET", "OPTIONS")
	protected.HandleFunc("/status", status.GetMyStatusHandler(db)).Methods("GET", "OPTIONS")

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, c.Handler(r)))
}
