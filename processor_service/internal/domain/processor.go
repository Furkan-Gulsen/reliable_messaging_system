package domain

import (
	"time"

	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/models"
)

type ProcessingResult struct {
	Success      bool
	Error        error
	ShouldRetry  bool
	IsDuplicate  bool
	IsStale      bool
}

type MessageProcessor struct {
	maxRetries    int
	staleDuration time.Duration
}

func NewMessageProcessor(maxRetries int, staleDuration time.Duration) *MessageProcessor {
	return &MessageProcessor{
		maxRetries:    maxRetries,
		staleDuration: staleDuration,
	}
}

func (p *MessageProcessor) GetMaxRetries() int {
	return p.maxRetries
}

func (p *MessageProcessor) ShouldProcessMessage(msg *models.Message) ProcessingResult {
	if msg.RetryCount >= p.maxRetries {
		return ProcessingResult{
			Success:     false,
			Error:       nil,
			ShouldRetry: false,
		}
	}

	if time.Since(msg.UpdatedAt) > p.staleDuration {
		return ProcessingResult{
			Success:  false,
			Error:    nil,
			IsStale:  true,
		}
	}

	return ProcessingResult{
		Success:     true,
		ShouldRetry: true,
	}
}

func (p *MessageProcessor) IsMessageStale(lastUpdateTime time.Time) bool {
	return time.Since(lastUpdateTime) > p.staleDuration
} 