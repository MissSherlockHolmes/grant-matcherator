package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type ChatMessage struct {
	ID        string    `json:"id"`
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

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
	connections = make(map[int]map[*websocket.Conn]bool) // map[matchID]map[conn]bool
	connLock    sync.Mutex
)

func HandleWebSocket(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.URL.Query().Get("token"), "Bearer ")
		if token == "" {
			http.Error(w, "No token provided", http.StatusUnauthorized)
			return
		}

		userID, err := GetUserIDFromToken(&http.Request{
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

		// Verify user is part of this match
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM matches WHERE id = $1 AND (user_id_1 = $2 OR user_id_2 = $2)", matchID, userID).Scan(&count)
		if err != nil || count == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			//log.printf("WebSocket upgrade error: %v", err)
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
				//log.printf("WebSocket error: %v", err)
				break
			}

			if strings.Contains(string(p), `"typing"`) {
				var typingMessage TypingMessage
				if err := json.Unmarshal(p, &typingMessage); err != nil {
					//log.printf("Error parsing message: %v", err)
					continue
				}
				typingMessage.UserID = userID
				broadcastTyping(matchID, messageType, typingMessage)
				continue
			}

			var message ChatMessage
			if err := json.Unmarshal(p, &message); err != nil {
				//log.printf("Error parsing message: %v", err)
				continue
			}

			message.MatchID = matchID
			message.SenderID = userID
			message.Timestamp = time.Now()

			_, err = db.Exec("INSERT INTO chat_messages (id, match_id, sender_id, content, timestamp) VALUES ($1, $2, $3, $4, $5)", message.ID, message.MatchID, message.SenderID, message.Content, message.Timestamp)
			if err != nil {
				//log.printf("Database error: %v", err)
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
		//log.printf("Error encoding message: %v", err)
		return
	}

	for conn := range connections[matchID] {
		if err := conn.WriteMessage(messageType, msgData); err != nil {
			//log.printf("Error sending message: %v", err)
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
		//log.printf("Error encoding message: %v", err)
		return
	}

	for conn := range connections[matchID] {
		if err := conn.WriteMessage(messageType, msgData); err != nil {
			//log.printf("Error sending message: %v", err)
			conn.Close()
			delete(connections[matchID], conn)
		}
	}
}

func GetChatsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		rows, err := db.Query(`
			WITH LastMessage AS (
				SELECT match_id, MAX(timestamp) as last_message_time
				FROM chat_messages
				GROUP BY match_id
			)
			SELECT DISTINCT m.id, m.user_id_1, m.user_id_2, m.status,
				   p1.name as user1_name, p2.name as user2_name,
				   p1.profile_picture_url as user1_picture,
				   p2.profile_picture_url as user2_picture,
				   lm.last_message_time
			FROM matches m
			JOIN profiles p1 ON m.user_id_1 = p1.user_id
			JOIN profiles p2 ON m.user_id_2 = p2.user_id
			LEFT JOIN LastMessage lm ON m.id = lm.match_id
			WHERE (m.user_id_1 = $1 OR m.user_id_2 = $1)
				AND m.status = 'connected'
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
			var user1ID, user2ID int
			var user1Name, user2Name string
			var user1Picture, user2Picture string
			var matchStatus string
			var lastMessageTime sql.NullTime

			err := rows.Scan(
				&chat.MatchID, &user1ID, &user2ID, &matchStatus,
				&user1Name, &user2Name, &user1Picture, &user2Picture,
				&lastMessageTime,
			)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Determine which user is the "other" user
			if userID == user1ID {
				chat.OtherUserID = user2ID
				chat.OtherUserName = user2Name
				chat.OtherUserPicture = user2Picture
			} else {
				chat.OtherUserID = user1ID
				chat.OtherUserName = user1Name
				chat.OtherUserPicture = user1Picture
			}

			if lastMessageTime.Valid {
				chat.LastMessageTime = lastMessageTime.Time
			}

			// Get last message and unread count
			err = db.QueryRow(`
				SELECT content
				FROM chat_messages
				WHERE match_id = $1
				ORDER BY timestamp DESC
				LIMIT 1
			`, chat.MatchID).Scan(&chat.LastMessage)
			if err != nil && err != sql.ErrNoRows {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			err = db.QueryRow(`
				SELECT COUNT(*)
				FROM chat_messages
				WHERE match_id = $1 AND sender_id != $2 AND read = false
			`, chat.MatchID, userID).Scan(&chat.UnreadCount)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			chats = append(chats, chat)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chats)
	}
}

func GetChatMessagesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		matchID := vars["id"]

		// Parse pagination parameters
		page := 1
		pageSize := 50
		if pageStr := r.URL.Query().Get("page"); pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}
		offset := (page - 1) * pageSize

		// Verify user is part of this match
		var count int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM matches
			WHERE id = $1 AND (user_id_1 = $2 OR user_id_2 = $2)
		`, matchID, userID).Scan(&count)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if count == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		rows, err := db.Query(`
			SELECT id, sender_id, content, timestamp, read
			FROM chat_messages
			WHERE match_id = $1
			ORDER BY timestamp ASC
			LIMIT $2 OFFSET $3
		`, matchID, pageSize, offset)
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
			messages = append(messages, msg)
		}

		// Get total count for pagination
		var totalMessages int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM chat_messages WHERE match_id = $1
		`, matchID).Scan(&totalMessages)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := struct {
			Messages []ChatMessage `json:"messages"`
			Total    int           `json:"total"`
			Page     int           `json:"page"`
			Pages    int           `json:"pages"`
		}{
			Messages: messages,
			Total:    totalMessages,
			Page:     page,
			Pages:    (totalMessages + pageSize - 1) / pageSize,
		}

		// Mark messages as read
		_, err = db.Exec(`
			UPDATE chat_messages
			SET read = true
			WHERE match_id = $1 AND sender_id != $2 AND read = false
		`, matchID, userID)
		if err != nil {
			//log.printf("Error marking messages as read: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func MarkMessagesAsReadHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := GetUserIDFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		_, err = db.Exec(
			`UPDATE user_status
            SET last_message_check = CURRENT_TIMESTAMP
            WHERE user_id = $1`, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
