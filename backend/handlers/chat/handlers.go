package chat

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"matcherator/backend/handlers/auth"
	"matcherator/backend/handlers/status"

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
		log.Printf("WebSocket connection attempt received")
		log.Printf("Request URL: %s", r.URL.String())
		log.Printf("Request Headers: %v", r.Header)
		log.Printf("Request Method: %s", r.Method)
		log.Printf("Request Protocol: %s", r.Proto)
		log.Printf("Request Host: %s", r.Host)
		log.Printf("Request RemoteAddr: %s", r.RemoteAddr)

		token := strings.TrimPrefix(r.URL.Query().Get("token"), "Bearer ")
		if token == "" {
			log.Printf("No token provided in WebSocket connection")
			http.Error(w, "No token provided", http.StatusUnauthorized)
			return
		}
		log.Printf("Token received: %s", token)

		userID, err := auth.GetUserIDFromToken(&http.Request{
			Header: http.Header{"Authorization": []string{"Bearer " + token}},
		})
		if err != nil {
			log.Printf("Invalid token in WebSocket connection: %v", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		log.Printf("User authenticated: %d", userID)

		matchID, err := strconv.Atoi(mux.Vars(r)["matchId"])
		if err != nil {
			log.Printf("Invalid match ID in WebSocket connection: %v", err)
			http.Error(w, "Invalid match ID", http.StatusBadRequest)
			return
		}
		log.Printf("Match ID: %d", matchID)

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

		if err != nil {
			log.Printf("Database error checking connection: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		if count == 0 {
			log.Printf("No valid connection found for match ID %d and user ID %d", matchID, userID)
			http.Error(w, "Unauthorized or chat not available", http.StatusUnauthorized)
			return
		}
		log.Printf("Connection verified for match ID %d and user ID %d", matchID, userID)

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Error upgrading connection: %v", err)
			return
		}
		defer conn.Close()

		log.Printf("WebSocket connection established successfully")

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

type ChatPreview struct {
	ID               int            `json:"id"`
	InitiatorID      int            `json:"initiator_id"`
	TargetID         int            `json:"target_id"`
	InitiatorName    sql.NullString `json:"-"`
	TargetName       sql.NullString `json:"-"`
	InitiatorPicture sql.NullString `json:"-"`
	TargetPicture    sql.NullString `json:"-"`
	LastMessageTime  sql.NullTime   `json:"-"`
	LastMessage      string         `json:"last_message,omitempty"`
	LastMessageAt    *time.Time     `json:"last_message_at,omitempty"`
	OtherUserName    string         `json:"other_user_name"`
	OtherUserPicture string         `json:"other_user_picture"`
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
				SELECT 
					match_id,
					timestamp as last_message_time,
					content as last_message,
					ROW_NUMBER() OVER (PARTITION BY match_id ORDER BY timestamp DESC) as rn
				FROM chat_messages
			)
			SELECT DISTINCT 
				c.id,
				c.initiator_id,
				c.target_id,
				COALESCE(p1.organization_name, '') as initiator_name,
				COALESCE(p2.organization_name, '') as target_name,
				COALESCE(p1.profile_picture_url, '') as initiator_picture,
				COALESCE(p2.profile_picture_url, '') as target_picture,
				COALESCE(lm.last_message_time, CURRENT_TIMESTAMP) as last_message_time,
				COALESCE(lm.last_message, '') as last_message
			FROM connections c
			JOIN profiles p1 ON c.initiator_id = p1.user_id
			JOIN profiles p2 ON c.target_id = p2.user_id
			LEFT JOIN LastMessage lm ON c.id = lm.match_id AND lm.rn = 1
			WHERE (c.initiator_id = $1 OR c.target_id = $1)
			ORDER BY last_message_time DESC NULLS LAST
		`, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Get column types and names once before the loop
		types, _ := rows.ColumnTypes()
		columns, _ := rows.Columns()
		log.Printf("Query returns %d columns", len(columns))
		for i, t := range types {
			log.Printf("Column %d: %s has type %s", i+1, columns[i], t.DatabaseTypeName())
		}

		var chats []ChatPreview
		for rows.Next() {
			var chat ChatPreview
			// Debug values before scan
			var id, initiatorID, targetID int
			var initiatorName, targetName, initiatorPicture, targetPicture, lastMessage sql.NullString
			var lastMessageTime sql.NullTime

			err := rows.Scan(
				&id,
				&initiatorID,
				&targetID,
				&initiatorName,
				&targetName,
				&initiatorPicture,
				&targetPicture,
				&lastMessageTime,
				&lastMessage,
			)
			if err != nil {
				log.Printf("Debug row values: id=%d, initiatorID=%d, targetID=%d, initiatorName=%v, targetName=%v, initiatorPic=%v, targetPic=%v, lastMessageTime=%v, lastMessage=%v",
					id, initiatorID, targetID, initiatorName, targetName, initiatorPicture, targetPicture, lastMessageTime, lastMessage)
				log.Printf("Failed to scan chat row: %v", err)
				http.Error(w, "Failed to scan chat row: "+err.Error(), http.StatusInternalServerError)
				return
			}

			chat.ID = id
			chat.InitiatorID = initiatorID
			chat.TargetID = targetID
			chat.InitiatorName = initiatorName
			chat.TargetName = targetName
			chat.InitiatorPicture = initiatorPicture
			chat.TargetPicture = targetPicture
			chat.LastMessageTime = lastMessageTime
			chat.LastMessage = lastMessage.String

			// Set the other user's name and picture based on whether the current user is initiator or target
			if chat.InitiatorID == userID {
				if chat.TargetName.Valid {
					chat.OtherUserName = chat.TargetName.String
				}
				if chat.TargetPicture.Valid {
					chat.OtherUserPicture = chat.TargetPicture.String
				}
			} else {
				if chat.InitiatorName.Valid {
					chat.OtherUserName = chat.InitiatorName.String
				}
				if chat.InitiatorPicture.Valid {
					chat.OtherUserPicture = chat.InitiatorPicture.String
				}
			}

			if chat.LastMessageTime.Valid {
				chat.LastMessageAt = &chat.LastMessageTime.Time
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
