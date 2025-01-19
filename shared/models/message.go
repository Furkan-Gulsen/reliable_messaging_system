package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MessageStatus string

const (
	StatusUnsent     MessageStatus = "unsent"
	StatusProcessing MessageStatus = "processing"
	StatusSent       MessageStatus = "sent"
	StatusFailed     MessageStatus = "failed"
	StatusDuplicate  MessageStatus = "duplicate"
)

type Message struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	To        string            `bson:"to" json:"to"`
	Content   string            `bson:"content" json:"content"`
	Status    MessageStatus     `bson:"status" json:"status"`
	RetryCount int              `bson:"retry_count" json:"retry_count"`
	CreatedAt time.Time         `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time         `bson:"updated_at" json:"updated_at"`
} 