package domain

import (
	"time"

	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/models"
)

type MessageSender struct {
	batchSize     int
	checkInterval time.Duration
}

func NewMessageSender(batchSize int, checkInterval time.Duration) *MessageSender {
	return &MessageSender{
		batchSize:     batchSize,
		checkInterval: checkInterval,
	}
}

func (s *MessageSender) PrepareMessage(content string, to string) *models.Message {
	return &models.Message{
		Content:    content,
		To:         to,
		Status:     models.StatusUnsent,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		RetryCount: 0,
	}
}

func (s *MessageSender) GetBatchSize() int {
	return s.batchSize
}

func (s *MessageSender) GetCheckInterval() time.Duration {
	return s.checkInterval
} 