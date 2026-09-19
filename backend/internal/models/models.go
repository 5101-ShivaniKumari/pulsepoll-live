package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents an authenticated poll creator
type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"password_hash" json:"-"`
	Name         string             `bson:"name" json:"name"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}

// Option represents a choice in a poll
type Option struct {
	ID        string `bson:"id" json:"id"`
	Text      string `bson:"text" json:"text"`
	VoteCount int64  `bson:"vote_count" json:"vote_count"`
	Order     int    `bson:"order" json:"order"`
}

// Poll represents a live poll
type Poll struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title       string             `bson:"title" json:"title"`
	Description string             `bson:"description,omitempty" json:"description,omitempty"`
	Options     []Option           `bson:"options" json:"options"`
	CreatorID   primitive.ObjectID `bson:"creator_id" json:"creator_id"`
	IsClosed    bool               `bson:"is_closed" json:"is_closed"`
	ExpiryAt    *time.Time         `bson:"expiry_at,omitempty" json:"expiry_at,omitempty"`
	TotalVotes  int64              `bson:"total_votes" json:"total_votes"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// Vote represents a durable vote audit record in MongoDB
type Vote struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PollID     primitive.ObjectID `bson:"poll_id" json:"poll_id"`
	OptionID   string             `bson:"option_id" json:"option_id"`
	VoterToken string             `bson:"voter_token" json:"voter_token"`
	IPHash     string             `bson:"ip_hash" json:"ip_hash"`
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
}

// LivePollUpdate is published to Redis and broadcast over WebSockets
type LivePollUpdate struct {
	Type                string             `json:"type"` // "vote_cast", "poll_closed", "poll_state", "viewer_update"
	PollID              string             `json:"poll_id"`
	TotalVotes          int64              `json:"total_votes"`
	OptionVotes         map[string]int64   `json:"option_votes"`
	Percentages         map[string]float64 `json:"percentages"`
	IsClosed            bool               `json:"is_closed"`
	ViewerCount         int64              `json:"viewer_count"`
	RecentVotedOptionID string             `json:"recent_voted_option_id,omitempty"`
	Timestamp           int64              `json:"timestamp"`
}

// DTOs for requests and responses

type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6,max=100"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string   `json:"token"`
	User  UserInfo `json:"user"`
}

type UserInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type CreatePollRequest struct {
	Title       string     `json:"title" binding:"required,min=3,max=300"`
	Description string     `json:"description" binding:"max=1000"`
	Options     []string   `json:"options" binding:"required,min=2,max=10"`
	ExpiryAt    *time.Time `json:"expiry_at,omitempty"`
}

type VoteRequest struct {
	OptionID   string `json:"option_id" binding:"required"`
	VoterToken string `json:"voter_token" binding:"required,min=10,max=128"`
}

type PollDetailResponse struct {
	Poll
	HasVoted   bool               `json:"has_voted"`
	VotedFor   string             `json:"voted_for,omitempty"`
	Percentages map[string]float64 `json:"percentages"`
}
