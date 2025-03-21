package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type NotificationResponse struct {
	UnreadMessages int `json:"unreadMessages"`
	NewMatches     int `json:"newMatches"`
}

var notificationConnections = make(map[int]*websocket.Conn)
var notifLock sync.Mutex

func GetNotificationsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, err := GetUserIDFromToken(r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}

		// Get the last check times separately
		var lastMessageCheck, lastMatchCheck sql.NullTime
		err = db.QueryRow(
			`SELECT last_message_check, last_match_check FROM user_status WHERE user_id = $1`,
			userID).Scan(&lastMessageCheck, &lastMatchCheck)
		if err != nil && err != sql.ErrNoRows {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
			return
		}

		// Get unread messages count (compared to last_message_check)
		var unreadMessages int
		err = db.QueryRow(
			`SELECT COUNT(*) FROM chat_messages cm
            JOIN matches m ON cm.match_id = m.id
            WHERE (m.user_id_1 = $1 OR m.user_id_2 = $1)
            AND cm.sender_id != $1
            AND cm.read = false
            AND ($2::timestamp IS NULL OR cm.timestamp > $2)`,
			userID, lastMessageCheck).Scan(&unreadMessages)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
			return
		}

		// Get new matches count (compared to last_match_check)
		var newMatches int
		err = db.QueryRow(
			`SELECT COUNT(*) FROM matches
            WHERE (user_id_1 = $1 OR user_id_2 = $1)
            AND status = 'connected'
            AND ($2::timestamp IS NULL OR created_at > $2)`,
			userID, lastMatchCheck).Scan(&newMatches)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
			return
		}

		response := NotificationResponse{
			UnreadMessages: unreadMessages,
			NewMatches:     newMatches,
		}

		json.NewEncoder(w).Encode(response)
	}
}

func MarkNotificationsAsReadHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, err := GetUserIDFromToken(r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}

		_, err = db.Exec(
			`UPDATE user_status
			SET last_notification_check = CURRENT_TIMESTAMP
		WHERE user_id = $1`, userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"message": "Notifications marked as read"})
	}
}

func HandleNotificationWebSocket() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract token from query parameter instead of header
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "No token provided", http.StatusUnauthorized)
			return
		}

		// Create a mock request with proper Authorization header
		mockReq := &http.Request{
			Header: http.Header{
				"Authorization": []string{"Bearer " + token},
			},
		}

		userID, err := GetUserIDFromToken(mockReq)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		upgrader := websocket.Upgrader{
			CheckOrigin:     func(r *http.Request) bool { return true },
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			//log.printf("WebSocket upgrade error for user %d: %v", userID, err)
			return
		}

		// Use defer with cleanup
		defer func() {
			notifLock.Lock()
			delete(notificationConnections, userID)
			notifLock.Unlock()
			conn.Close()
		}()

		// Store connection with proper locking
		notifLock.Lock()
		notificationConnections[userID] = conn
		notifLock.Unlock()

		// Send initial connection success message
		data, _ := json.Marshal(map[string]string{"type": "connected"})
		err = conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			//log.printf("Error sending connection confirmation: %v", err)
			return
		}

		// Keep connection alive and handle messages
		for {
			messageType, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					//log.printf("WebSocket error for user %d: %v", userID, err)
				}
				break
			}

			// Respond to ping messages to keep connection alive
			if messageType == websocket.PingMessage {
				if err := conn.WriteMessage(websocket.PongMessage, nil); err != nil {
					//log.printf("Error sending pong: %v", err)
					break
				}
			}
		}
	}
}

func MarkMatchesAsReadHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, err := GetUserIDFromToken(r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}

		_, err = db.Exec(
			`UPDATE user_status
            SET last_match_check = CURRENT_TIMESTAMP
            WHERE user_id = $1`, userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"message": "Matches marked as read"})
	}
}

// Broadcast notification to a user
func SendNotification(userID int, messageType string) {
	notifLock.Lock()
	conn, exists := notificationConnections[userID]
	notifLock.Unlock()

	if exists {
		data, _ := json.Marshal(map[string]string{
			"type": messageType,
		})
		conn.WriteMessage(websocket.TextMessage, data)
	}
}
