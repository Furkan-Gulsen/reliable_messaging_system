package mocks

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

type MockIdempotencyService struct {
	mock.Mock
}

func (m *MockIdempotencyService) IsProcessed(ctx context.Context, messageID string) (bool, error) {
	args := m.Called(ctx, messageID)
	return args.Bool(0), args.Error(1)
}

func (m *MockIdempotencyService) MarkAsProcessed(ctx context.Context, messageID string) error {
	args := m.Called(ctx, messageID)
	return args.Error(0)
}

func (m *MockIdempotencyService) StoreWebhookMessageID(ctx context.Context, messageID string, webhookMessageID string, ttl time.Duration) error {
	args := m.Called(ctx, messageID, webhookMessageID, ttl)
	return args.Error(0)
} 