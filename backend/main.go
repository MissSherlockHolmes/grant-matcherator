package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
	"golang.org/x/exp/rand"

	"matchme/backend/handlers"
	"matchme/backend/handlers/auth"
	"matchme/backend/handlers/chat"
	"matchme/backend/handlers/connection"
	"matchme/backend/handlers/media"
	"matchme/backend/handlers/notifications"
	"matchme/backend/handlers/profile"
	"matchme/backend/handlers/status"
	"matchme/backend/handlers/user"
)

func main() {
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
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	})

	// Auth routes
	r.HandleFunc("/api/auth/signup", auth.SignupHandler(db)).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/auth/login", auth.LoginHandler(db)).Methods("POST", "OPTIONS")

	// User routes
	r.HandleFunc("/api/users/{id}", user.GetUserHandler(db)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/users/{id}/full", user.GetFullUserHandler(db)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/users/{id}/profile", profile.GetUserProfileHandler(db)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/users/{id}/bio", profile.GetUserBioHandler(db)).Methods("GET", "OPTIONS")

	// Me routes
	r.HandleFunc("/api/me", user.GetMyBasicInfoHandler(db)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/me/profile", profile.GetUserProfileHandler(db)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/me/profile", profile.UpdateProfileHandler(db)).Methods("PUT", "OPTIONS")
	r.HandleFunc("/api/me/bio", profile.GetMyBioHandler(db)).Methods("GET", "OPTIONS")

	// Upload routes
	r.HandleFunc("/api/upload/profile-picture", media.UploadProfilePictureHandler(db)).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/upload/profile-picture", media.DeleteProfilePictureHandler(db)).Methods("DELETE", "OPTIONS")

	// Connections and Matching routes
	r.HandleFunc("/api/connections", connection.GetConnectionsHandler(db)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/connections", connection.CreateConnectionHandler(db)).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/connections/{id}", connection.DeleteConnectionHandler(db)).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/api/potential-matches", connection.GetPotentialMatchesHandler(db)).Methods("GET", "OPTIONS")

	// Test data route
	r.HandleFunc("/api/test/generate-users", handlers.GenerateTestDataHandler(db)).Methods("POST", "OPTIONS")

	// Notification routes
	r.HandleFunc("/api/notifications", notifications.GetNotificationsHandler(db)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/notifications/read", notifications.MarkNotificationsAsReadHandler(db)).Methods("POST", "OPTIONS")
	r.HandleFunc("/ws/notifications", notifications.HandleNotificationWebSocket())

	// Chat routes
	r.HandleFunc("/api/chat/preferences", chat.GetChatPreferencesHandler(db)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/chat/preferences", chat.UpdateChatPreferencesHandler(db)).Methods("PUT", "OPTIONS")
	r.HandleFunc("/api/chat", chat.GetChatsHandler(db)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/chat/{id}/messages", chat.GetChatMessagesHandler(db)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/chat/{id}/messages/read", chat.MarkMessagesAsReadHandler(db)).Methods("POST", "OPTIONS")
	r.HandleFunc("/ws/chat/{matchId}", chat.HandleWebSocket(db))

	// Status routes
	r.HandleFunc("/api/status/{id}", status.GetStatusHandler(db)).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/status", status.GetMyStatusHandler(db)).Methods("GET", "OPTIONS")

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, c.Handler(r)))
}
