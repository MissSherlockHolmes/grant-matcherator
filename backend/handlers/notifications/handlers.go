package notifications

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"matcherator/backend/handlers/auth"

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

		// Get unread notifications
		rows, err := db.Query(`
			SELECT id, type, content, created_at, read_at
			FROM notifications
			WHERE user_id = $1
			ORDER BY created_at DESC
		`, userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
			return
		}
		defer rows.Close()

		type Notification struct {
			ID        int        `json:"id"`
			Type      string     `json:"type"`
			Content   string     `json:"content"`
			CreatedAt time.Time  `json:"created_at"`
			ReadAt    *time.Time `json:"read_at"`
		}

		var notifications []Notification

		for rows.Next() {
			var n Notification
			err := rows.Scan(&n.ID, &n.Type, &n.Content, &n.CreatedAt, &n.ReadAt)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "Error scanning notifications"})
				return
			}
			notifications = append(notifications, n)
		}

		json.NewEncoder(w).Encode(notifications)
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

		// Update read_at timestamp for all unread notifications
		_, err = db.Exec(`
			UPDATE notifications
			SET read_at = CURRENT_TIMESTAMP
			WHERE user_id = $1 AND read_at IS NULL
		`, userID)
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
