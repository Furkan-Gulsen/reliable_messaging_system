package domain

import (
	"testing"
	"time"

	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/models"

	"github.com/stretchr/testify/assert"
)

func TestNewMessageSender(t *testing.T) {
	batchSize := 5
	checkInterval := 10 * time.Second

	sender := NewMessageSender(batchSize, checkInterval)

	assert.NotNil(t, sender)
	assert.Equal(t, batchSize, sender.GetBatchSize())
	assert.Equal(t, checkInterval, sender.GetCheckInterval())
}

func TestMessageSender_PrepareMessage(t *testing.T) {
	sender := NewMessageSender(5, 10*time.Second)
	content := "test content"
	to := "+905321234569"

	msg := sender.PrepareMessage(content, to)

	assert.NotNil(t, msg)
	assert.Equal(t, content, msg.Content)
	assert.Equal(t, to, msg.To)
	assert.Equal(t, models.StatusUnsent, msg.Status)
	assert.Equal(t, 0, msg.RetryCount)
	assert.False(t, msg.CreatedAt.IsZero())
	assert.False(t, msg.UpdatedAt.IsZero())
} 