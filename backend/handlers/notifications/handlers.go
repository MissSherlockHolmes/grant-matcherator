package notifications

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sync"

	"matchme/backend/handlers/auth"

	"github.com/gorilla/websocket"
)

type NotificationResponse struct {
	UnreadMessages int `json:"unreadMessages"`
	NewConnections int `json:"newConnections"`
}

var notificationConnections = make(map[int]*websocket.Conn)
var notifLock sync.Mutex

func GetNotificationsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, err := auth.GetUserIDFromToken(r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}

		// Get the last check time
		var lastCheck sql.NullTime
		err = db.QueryRow(
			`SELECT last_notification_check FROM notifications WHERE user_id = $1`,
			userID).Scan(&lastCheck)
		if err != nil && err != sql.ErrNoRows {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
			return
		}

		// Get unread messages count
		var unreadMessages int
		err = db.QueryRow(
			`SELECT COUNT(*) FROM chat_messages cm
			JOIN connections c ON cm.match_id = c.id
			WHERE (c.initiator_id = $1 OR c.target_id = $1)
			AND cm.sender_id != $1
			AND cm.read = false
			AND ($2::timestamp IS NULL OR cm.timestamp > $2)`,
			userID, lastCheck).Scan(&unreadMessages)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
			return
		}

		// Get new connections count
		var newConnections int
		err = db.QueryRow(
			`SELECT COUNT(*) FROM connections
			WHERE (initiator_id = $1 OR target_id = $1)
			AND ($2::timestamp IS NULL OR created_at > $2)`,
			userID, lastCheck).Scan(&newConnections)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
			return
		}

		response := NotificationResponse{
			UnreadMessages: unreadMessages,
			NewConnections: newConnections,
		}

		json.NewEncoder(w).Encode(response)
	}
}

func MarkNotificationsAsReadHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, err := auth.GetUserIDFromToken(r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}

		// Update last notification check time
		_, err = db.Exec(
			`INSERT INTO notifications (user_id, last_notification_check)
			VALUES ($1, CURRENT_TIMESTAMP)
			ON CONFLICT (user_id) 
			DO UPDATE SET last_notification_check = CURRENT_TIMESTAMP`,
			userID)
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
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "No token provided", http.StatusUnauthorized)
			return
		}

		mockReq := &http.Request{
			Header: http.Header{
				"Authorization": []string{"Bearer " + token},
			},
		}

		userID, err := auth.GetUserIDFromToken(mockReq)
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
			return
		}

		defer func() {
			notifLock.Lock()
			delete(notificationConnections, userID)
			notifLock.Unlock()
			conn.Close()
		}()

		notifLock.Lock()
		notificationConnections[userID] = conn
		notifLock.Unlock()

		data, _ := json.Marshal(map[string]string{"type": "connected"})
		err = conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			return
		}

		for {
			messageType, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					// Log error if needed
				}
				break
			}

			if messageType == websocket.PingMessage {
				if err := conn.WriteMessage(websocket.PongMessage, nil); err != nil {
					break
				}
			}
		}
	}
}

// SendNotification broadcasts a notification to a specific user
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
