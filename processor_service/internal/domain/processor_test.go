package domain

import (
	"testing"
	"time"

	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/models"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestMessageProcessor_ShouldProcessMessage(t *testing.T) {
	maxRetries := 3
	staleDuration := 4 * time.Minute
	processor := NewMessageProcessor(maxRetries, staleDuration)

	tests := []struct {
		name     string
		message  *models.Message
		expected ProcessingResult
	}{
		{
			name: "should process message with no retries",
			message: &models.Message{
				ID:         primitive.NewObjectID(),
				RetryCount: 0,
				UpdatedAt:  time.Now(),
			},
			expected: ProcessingResult{
				Success:     true,
				ShouldRetry: true,
			},
		},
		{
			name: "should not process message with max retries reached",
			message: &models.Message{
				ID:         primitive.NewObjectID(),
				RetryCount: maxRetries,
				UpdatedAt:  time.Now(),
			},
			expected: ProcessingResult{
				Success:     false,
				ShouldRetry: false,
			},
		},
		{
			name: "should not process stale message",
			message: &models.Message{
				ID:         primitive.NewObjectID(),
				RetryCount: 0,
				UpdatedAt:  time.Now().Add(-5 * time.Minute),
			},
			expected: ProcessingResult{
				Success: false,
				IsStale: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.ShouldProcessMessage(tt.message)
			assert.Equal(t, tt.expected.Success, result.Success)
			assert.Equal(t, tt.expected.ShouldRetry, result.ShouldRetry)
			assert.Equal(t, tt.expected.IsStale, result.IsStale)
		})
	}
}

func TestMessageProcessor_IsMessageStale(t *testing.T) {
	processor := NewMessageProcessor(3, 4*time.Minute)

	tests := []struct {
		name           string
		lastUpdateTime time.Time
		expected       bool
	}{
		{
			name:           "message is not stale",
			lastUpdateTime: time.Now(),
			expected:       false,
		},
		{
			name:           "message is stale",
			lastUpdateTime: time.Now().Add(-5 * time.Minute),
			expected:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.IsMessageStale(tt.lastUpdateTime)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessageProcessor_GetMaxRetries(t *testing.T) {
	maxRetries := 3
	processor := NewMessageProcessor(maxRetries, 4*time.Minute)

	assert.Equal(t, maxRetries, processor.GetMaxRetries())
} 