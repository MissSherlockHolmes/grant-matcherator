package chat

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"matchme/backend/handlers/auth"
	"matchme/backend/handlers/status"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type ChatMessage struct {
	ID        int       `json:"id"`
	MatchID   int       `json:"match_id"`
	SenderID  int       `json:"sender_id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Read      bool      `json:"read"`
}

type TypingMessage struct {
	MatchID int  `json:"match_id"`
	UserID  int  `json:"user_id"`
	Typing  bool `json:"typing"`
}

type ChatPreferences struct {
	OptIn bool `json:"opt_in"`
}

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
	connections = make(map[int]map[*websocket.Conn]bool) // map[matchID]map[conn]bool
	connLock    sync.Mutex
)

// UpdateChatPreferencesHandler allows users to opt in/out of chat
func UpdateChatPreferencesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, err := auth.GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var prefs ChatPreferences
		if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Check if user is active
		userStatus, err := status.GetUserStatus(db, userID)
		if err != nil {
			http.Error(w, "Error checking user status", http.StatusInternalServerError)
			return
		}
		if userStatus.Status != "active" {
			http.Error(w, "Only active users can update chat preferences", http.StatusForbidden)
			return
		}

		_, err = db.Exec(`
			UPDATE profiles 
			SET chat_opt_in = $1, updated_at = CURRENT_TIMESTAMP 
			WHERE user_id = $2
		`, prefs.OptIn, userID)

		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// GetChatPreferencesHandler retrieves the user's chat preferences
func GetChatPreferencesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, err := auth.GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var prefs ChatPreferences
		err = db.QueryRow(`
			SELECT chat_opt_in 
			FROM profiles 
			WHERE user_id = $1
		`, userID).Scan(&prefs.OptIn)

		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(prefs)
	}
}

func HandleWebSocket(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.URL.Query().Get("token"), "Bearer ")
		if token == "" {
			http.Error(w, "No token provided", http.StatusUnauthorized)
			return
		}

		userID, err := auth.GetUserIDFromToken(&http.Request{
			Header: http.Header{"Authorization": []string{"Bearer " + token}},
		})
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		matchID, err := strconv.Atoi(mux.Vars(r)["matchId"])
		if err != nil {
			http.Error(w, "Invalid match ID", http.StatusBadRequest)
			return
		}

		// Verify user is part of this connection and both users are active and opted in
		var count int
		err = db.QueryRow(`
			SELECT COUNT(*) 
			FROM connections c
			JOIN users u1 ON c.initiator_id = u1.id
			JOIN users u2 ON c.target_id = u2.id
			JOIN profiles p1 ON u1.id = p1.user_id
			JOIN profiles p2 ON u2.id = p2.user_id
			WHERE c.id = $1 
			AND (c.initiator_id = $2 OR c.target_id = $2)
			AND p1.chat_opt_in = true 
			AND p2.chat_opt_in = true
			AND u1.role = 'provider' 
			AND u2.role = 'recipient'
			AND (
				(u1.id = c.initiator_id AND u1.status = 'active') OR
				(u2.id = c.initiator_id AND u2.status = 'active')
			)
			AND (
				(u1.id = c.target_id AND u1.status = 'active') OR
				(u2.id = c.target_id AND u2.status = 'active')
			)
		`, matchID, userID).Scan(&count)

		if err != nil || count == 0 {
			http.Error(w, "Unauthorized or chat not available", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Store connection
		connLock.Lock()
		if connections[matchID] == nil {
			connections[matchID] = make(map[*websocket.Conn]bool)
		}
		connections[matchID][conn] = true
		connLock.Unlock()

		// Listen for messages
		for {
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				break
			}

			if strings.Contains(string(p), `"typing"`) {
				var typingMessage TypingMessage
				if err := json.Unmarshal(p, &typingMessage); err != nil {
					continue
				}
				typingMessage.UserID = userID
				broadcastTyping(matchID, messageType, typingMessage)
				continue
			}

			var message ChatMessage
			if err := json.Unmarshal(p, &message); err != nil {
				continue
			}

			message.MatchID = matchID
			message.SenderID = userID
			message.Timestamp = time.Now()

			_, err = db.Exec(`
				INSERT INTO chat_messages (id, match_id, sender_id, content, timestamp) 
				VALUES ($1, $2, $3, $4, $5)
			`, message.ID, message.MatchID, message.SenderID, message.Content, message.Timestamp)
			if err != nil {
				continue
			}

			// Broadcast message
			broadcastMessage(matchID, messageType, message)
		}

		// Cleanup on disconnect
		connLock.Lock()
		delete(connections[matchID], conn)
		if len(connections[matchID]) == 0 {
			delete(connections, matchID)
		}
		connLock.Unlock()
	}
}

func broadcastMessage(matchID, messageType int, message ChatMessage) {
	connLock.Lock()
	defer connLock.Unlock()

	msgData, err := json.Marshal(message)
	if err != nil {
		return
	}

	for conn := range connections[matchID] {
		if err := conn.WriteMessage(messageType, msgData); err != nil {
			conn.Close()
			delete(connections[matchID], conn)
		}
	}
}

func broadcastTyping(matchID, messageType int, typingMessage TypingMessage) {
	connLock.Lock()
	defer connLock.Unlock()

	msgData, err := json.Marshal(typingMessage)
	if err != nil {
		return
	}

	for conn := range connections[matchID] {
		if err := conn.WriteMessage(messageType, msgData); err != nil {
			conn.Close()
			delete(connections[matchID], conn)
		}
	}
}

func GetChatsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := auth.GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check if user is active and opted in
		var chatOptIn bool
		err = db.QueryRow(`
			SELECT p.chat_opt_in 
			FROM profiles p
			JOIN users u ON p.user_id = u.id
			WHERE p.user_id = $1 AND u.status = 'active'
		`, userID).Scan(&chatOptIn)

		if err == sql.ErrNoRows {
			http.Error(w, "User not found or inactive", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		if !chatOptIn {
			http.Error(w, "Chat is not enabled for this user", http.StatusForbidden)
			return
		}

		rows, err := db.Query(`
			WITH LastMessage AS (
				SELECT match_id, MAX(timestamp) as last_message_time
				FROM chat_messages
				GROUP BY match_id
			)
			SELECT DISTINCT c.id, c.initiator_id, c.target_id,
				   p1.organization_name as initiator_name,
				   p2.organization_name as target_name,
				   p1.profile_picture_url as initiator_picture,
				   p2.profile_picture_url as target_picture,
				   lm.last_message_time
			FROM connections c
			JOIN profiles p1 ON c.initiator_id = p1.user_id
			JOIN profiles p2 ON c.target_id = p2.user_id
			LEFT JOIN LastMessage lm ON c.id = lm.match_id
			WHERE (c.initiator_id = $1 OR c.target_id = $1)
			ORDER BY lm.last_message_time DESC NULLS LAST
		`, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type ChatPreview struct {
			MatchID          int       `json:"match_id"`
			OtherUserID      int       `json:"other_user_id"`
			OtherUserName    string    `json:"other_user_name"`
			OtherUserPicture string    `json:"other_user_picture"`
			LastMessage      string    `json:"last_message"`
			UnreadCount      int       `json:"unread_count"`
			LastMessageTime  time.Time `json:"last_message_time"`
		}

		var chats []ChatPreview
		for rows.Next() {
			var chat ChatPreview
			var initiatorID, targetID int
			var initiatorName, targetName string
			var initiatorPicture, targetPicture string
			var lastMessageTime sql.NullTime

			err := rows.Scan(
				&chat.MatchID, &initiatorID, &targetID,
				&initiatorName, &targetName,
				&initiatorPicture, &targetPicture,
				&lastMessageTime,
			)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Determine which user is the "other" user
			if userID == initiatorID {
				chat.OtherUserID = targetID
				chat.OtherUserName = targetName
				chat.OtherUserPicture = targetPicture
			} else {
				chat.OtherUserID = initiatorID
				chat.OtherUserName = initiatorName
				chat.OtherUserPicture = initiatorPicture
			}

			if lastMessageTime.Valid {
				chat.LastMessageTime = lastMessageTime.Time
			}

			// Get last message and unread count
			err = db.QueryRow(`
				SELECT content, COUNT(*) FILTER (WHERE NOT read AND sender_id != $1)
				FROM chat_messages
				WHERE match_id = $2
				ORDER BY timestamp DESC
				LIMIT 1
			`, userID, chat.MatchID).Scan(&chat.LastMessage, &chat.UnreadCount)

			if err != nil && err != sql.ErrNoRows {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			chats = append(chats, chat)
		}

		json.NewEncoder(w).Encode(chats)
	}
}

func GetChatMessagesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := auth.GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		matchID, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			http.Error(w, "Invalid match ID", http.StatusBadRequest)
			return
		}

		// Verify user is part of this connection and both users are active and opted in
		var count int
		err = db.QueryRow(`
			SELECT COUNT(*) 
			FROM connections c
			JOIN users u1 ON c.initiator_id = u1.id
			JOIN users u2 ON c.target_id = u2.id
			JOIN profiles p1 ON u1.id = p1.user_id
			JOIN profiles p2 ON u2.id = p2.user_id
			WHERE c.id = $1 
			AND (c.initiator_id = $2 OR c.target_id = $2)
			AND p1.chat_opt_in = true 
			AND p2.chat_opt_in = true
			AND u1.role = 'provider' 
			AND u2.role = 'recipient'
			AND (
				(u1.id = c.initiator_id AND u1.status = 'active') OR
				(u2.id = c.initiator_id AND u2.status = 'active')
			)
			AND (
				(u1.id = c.target_id AND u1.status = 'active') OR
				(u2.id = c.target_id AND u2.status = 'active')
			)
		`, matchID, userID).Scan(&count)

		if err != nil || count == 0 {
			http.Error(w, "Unauthorized or chat not available", http.StatusUnauthorized)
			return
		}

		rows, err := db.Query(`
			SELECT id, sender_id, content, timestamp, read
			FROM chat_messages
			WHERE match_id = $1
			ORDER BY timestamp ASC
		`, matchID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var messages []ChatMessage
		for rows.Next() {
			var msg ChatMessage
			err := rows.Scan(&msg.ID, &msg.SenderID, &msg.Content, &msg.Timestamp, &msg.Read)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			msg.MatchID = matchID
			messages = append(messages, msg)
		}

		json.NewEncoder(w).Encode(messages)
	}
}

func MarkMessagesAsReadHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := auth.GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		matchID, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			http.Error(w, "Invalid match ID", http.StatusBadRequest)
			return
		}

		// Verify user is part of this connection and both users are active and opted in
		var count int
		err = db.QueryRow(`
			SELECT COUNT(*) 
			FROM connections c
			JOIN users u1 ON c.initiator_id = u1.id
			JOIN users u2 ON c.target_id = u2.id
			JOIN profiles p1 ON u1.id = p1.user_id
			JOIN profiles p2 ON u2.id = p2.user_id
			WHERE c.id = $1 
			AND (c.initiator_id = $2 OR c.target_id = $2)
			AND p1.chat_opt_in = true 
			AND p2.chat_opt_in = true
			AND u1.role = 'provider' 
			AND u2.role = 'recipient'
			AND (
				(u1.id = c.initiator_id AND u1.status = 'active') OR
				(u2.id = c.initiator_id AND u2.status = 'active')
			)
			AND (
				(u1.id = c.target_id AND u1.status = 'active') OR
				(u2.id = c.target_id AND u2.status = 'active')
			)
		`, matchID, userID).Scan(&count)

		if err != nil || count == 0 {
			http.Error(w, "Unauthorized or chat not available", http.StatusUnauthorized)
			return
		}

		_, err = db.Exec(`
			UPDATE chat_messages
			SET read = true
			WHERE match_id = $1 AND sender_id != $2 AND read = false
		`, matchID, userID)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
