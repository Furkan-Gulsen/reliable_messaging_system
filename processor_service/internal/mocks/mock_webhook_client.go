package mocks

import (
	"context"

	"github.com/Furkan-Gulsen/reliable_messaging_system/processor_service/internal/application/ports"

	"github.com/stretchr/testify/mock"
)

type MockWebhookClient struct {
	mock.Mock
}

func (m *MockWebhookClient) SendMessage(ctx context.Context, content string, to string) (*ports.WebhookResponse, error) {
	args := m.Called(ctx, content, to)
	if resp, ok := args.Get(0).(*ports.WebhookResponse); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
} 