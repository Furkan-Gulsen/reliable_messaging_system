package mocks

import (
	"context"
	"time"

	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/models"

	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MockMessageRepository struct {
	mock.Mock
}

func (m *MockMessageRepository) CreateMessage(ctx context.Context, message *models.Message) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *MockMessageRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Message, error) {
	args := m.Called(ctx, id)
	if msg, ok := args.Get(0).(*models.Message); ok {
		return msg, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMessageRepository) ListMessages(ctx context.Context) ([]models.Message, error) {
	args := m.Called(ctx)
	if msgs, ok := args.Get(0).([]models.Message); ok {
		return msgs, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMessageRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.MessageStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockMessageRepository) IncrementRetryCount(ctx context.Context, id primitive.ObjectID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMessageRepository) FindStaleProcessingMessages(ctx context.Context, staleDuration time.Duration) ([]models.Message, error) {
	args := m.Called(ctx, staleDuration)
	if msgs, ok := args.Get(0).([]models.Message); ok {
		return msgs, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMessageRepository) FindUnsentMessages(ctx context.Context, limit int) ([]models.Message, error) {
	args := m.Called(ctx, limit)
	if msgs, ok := args.Get(0).([]models.Message); ok {
		return msgs, args.Error(1)
	}
	return nil, args.Error(1)
} 