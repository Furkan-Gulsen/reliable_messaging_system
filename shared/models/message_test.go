package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestMessage(t *testing.T) {
	t.Run("should create message with correct status", func(t *testing.T) {
		// Arrange
		now := time.Now()
		id := primitive.NewObjectID()
		
		// Act
		msg := Message{
			ID:         id,
			Content:    "test message",
			Status:     StatusUnsent,
			RetryCount: 0,
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		// Assert
		assert.Equal(t, id, msg.ID)
		assert.Equal(t, "test message", msg.Content)
		assert.Equal(t, StatusUnsent, msg.Status)
		assert.Equal(t, 0, msg.RetryCount)
		assert.Equal(t, now, msg.CreatedAt)
		assert.Equal(t, now, msg.UpdatedAt)
	})

	t.Run("should validate message status constants", func(t *testing.T) {
		// Assert
		assert.Equal(t, MessageStatus("unsent"), StatusUnsent)
		assert.Equal(t, MessageStatus("processing"), StatusProcessing)
		assert.Equal(t, MessageStatus("sent"), StatusSent)
		assert.Equal(t, MessageStatus("failed"), StatusFailed)
		assert.Equal(t, MessageStatus("duplicate"), StatusDuplicate)
	})
} 