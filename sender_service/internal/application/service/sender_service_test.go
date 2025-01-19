package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/Furkan-Gulsen/reliable_messaging_system/sender_service/internal/domain"
	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/models"
	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/ports/mongodb/interfaces"
	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/ports/rabbitmq/contracts"
	rabbitInterfaces "github.com/Furkan-Gulsen/reliable_messaging_system/shared/ports/rabbitmq/interfaces"
)

type MockMessageRepository struct {
	mock.Mock
	interfaces.MessageRepository
}

func (m *MockMessageRepository) CreateMessage(ctx context.Context, message *models.Message) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *MockMessageRepository) ListMessages(ctx context.Context) ([]models.Message, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Message), args.Error(1)
}

func (m *MockMessageRepository) FindUnsentMessages(ctx context.Context, limit int) ([]models.Message, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]models.Message), args.Error(1)
}

func (m *MockMessageRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.MessageStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockMessageRepository) FindStaleProcessingMessages(ctx context.Context, duration time.Duration) ([]models.Message, error) {
	args := m.Called(ctx, duration)
	return args.Get(0).([]models.Message), args.Error(1)
}

func (m *MockMessageRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Message, error) {
	args := m.Called(ctx, id)
	if msg, ok := args.Get(0).(*models.Message); ok {
		return msg, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMessageRepository) IncrementRetryCount(ctx context.Context, id primitive.ObjectID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type MockMessageQueue struct {
	mock.Mock
	rabbitInterfaces.MessageQueue
}

func (m *MockMessageQueue) PublishMessage(ctx context.Context, message contracts.QueueMessage) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *MockMessageQueue) GetDLQMessageCount() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

func (m *MockMessageQueue) Close() {
	m.Called()
}

func TestSenderService_CreateMessage(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockQueue := new(MockMessageQueue)
	sender := domain.NewMessageSender(5, 10*time.Second)
	service := NewSenderService(sender, mockRepo, mockQueue)

	ctx := context.Background()
	content := "test content"
	to := "+905321234567"

	mockRepo.On("CreateMessage", ctx, mock.MatchedBy(func(msg *models.Message) bool {
		return msg.Content == content &&
			msg.To == to &&
			msg.Status == models.StatusUnsent &&
			msg.RetryCount == 0 &&
			!msg.CreatedAt.IsZero() &&
			!msg.UpdatedAt.IsZero()
	})).Run(func(args mock.Arguments) {
		msg := args.Get(1).(*models.Message)
		msg.ID = primitive.NewObjectID()
	}).Return(nil)

	id, err := service.CreateMessage(ctx, content, to)

	assert.NoError(t, err)
	assert.NotEqual(t, primitive.NilObjectID, id)
	mockRepo.AssertExpectations(t)
}

func TestSenderService_ListMessages(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockQueue := new(MockMessageQueue)
	sender := domain.NewMessageSender(5, 10*time.Second)
	service := NewSenderService(sender, mockRepo, mockQueue)

	ctx := context.Background()
	expectedMessages := []models.Message{
		{
			ID:         primitive.NewObjectID(),
			Content:    "test1",
			To:         "+905321234567",
			Status:     models.StatusUnsent,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			RetryCount: 0,
		},
		{
			ID:         primitive.NewObjectID(),
			Content:    "test2",
			To:         "+905321234568",
			Status:     models.StatusProcessing,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			RetryCount: 1,
		},
	}

	mockRepo.On("ListMessages", ctx).Return(expectedMessages, nil)

	messages, err := service.ListMessages(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expectedMessages, messages)
	mockRepo.AssertExpectations(t)
}

func TestSenderService_Scheduler(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockQueue := new(MockMessageQueue)
	sender := domain.NewMessageSender(5, 10*time.Second)
	service := NewSenderService(sender, mockRepo, mockQueue)

	ctx := context.Background()

	service.StartScheduler(ctx)
	assert.True(t, service.scheduler.isRunning)

	service.StopScheduler(ctx)
	assert.False(t, service.scheduler.isRunning)
}

func TestMessageScheduler_ProcessUnsentMessages(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockQueue := new(MockMessageQueue)
	sender := domain.NewMessageSender(5, 10*time.Second)
	service := NewSenderService(sender, mockRepo, mockQueue)

	ctx := context.Background()
	unsentMessages := []models.Message{
		{
			ID:         primitive.NewObjectID(),
			Content:    "test1",
			To:         "+905321234569",
			Status:     models.StatusUnsent,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			RetryCount: 0,
		},
	}

	mockRepo.On("FindUnsentMessages", ctx, 5).Return(unsentMessages, nil)
	mockQueue.On("PublishMessage", ctx, mock.MatchedBy(func(msg contracts.QueueMessage) bool {
		return msg.ID == unsentMessages[0].ID.Hex() &&
			msg.Content == unsentMessages[0].Content &&
			msg.To == unsentMessages[0].To &&
			msg.Retry == unsentMessages[0].RetryCount
	})).Return(nil)
	mockRepo.On("UpdateStatus", ctx, unsentMessages[0].ID, models.StatusProcessing).Return(nil)

	err := service.scheduler.processUnsentMessages()

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockQueue.AssertExpectations(t)
} 