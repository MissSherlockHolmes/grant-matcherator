package connection

import "time"

// Connection represents a connection between two users
type Connection struct {
	ID               int       `json:"id"`
	InitiatorID      int       `json:"initiator_id"` // The user who created the connection
	TargetID         int       `json:"target_id"`    // The user being followed/connected to
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	OtherUserName    string    `json:"other_user_name"`
	OtherUserPicture string    `json:"other_user_picture"`
	ConnectionType   string    `json:"connection_type"` // "following" or "follower"
}

// ConnectionRequest represents the request body for creating a connection
type ConnectionRequest struct {
	TargetID int `json:"target_id"`
}
