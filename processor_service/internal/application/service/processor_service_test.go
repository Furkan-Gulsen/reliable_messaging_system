package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Furkan-Gulsen/reliable_messaging_system/processor_service/internal/application/ports"
	"github.com/Furkan-Gulsen/reliable_messaging_system/processor_service/internal/domain"
	"github.com/Furkan-Gulsen/reliable_messaging_system/processor_service/internal/mocks"
	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/models"
	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/ports/rabbitmq/contracts"

	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestProcessorService_ProcessMessage(t *testing.T) {
	mockRepo := new(mocks.MockMessageRepository)
	mockQueue := new(mocks.MockMessageQueue)
	mockIdempotency := new(mocks.MockIdempotencyService)
	mockWebhook := new(mocks.MockWebhookClient)
	processor := domain.NewMessageProcessor(3, 4*time.Minute)

	service := NewProcessorService(processor, mockRepo, mockQueue, mockIdempotency, mockWebhook)

	tests := []struct {
		name           string
		setupMocks     func(msgID string)
		expectedError  bool
		messageContent string
		messageTo      string
	}{
		{
			name: "successful message processing",
			setupMocks: func(msgID string) {
				id, _ := primitive.ObjectIDFromHex(msgID)
				msg := &models.Message{
					ID:         id,
					Content:    "test content",
					To:         "test@example.com",
					RetryCount: 0,
					UpdatedAt:  time.Now(),
				}

				mockIdempotency.On("IsProcessed", mock.Anything, msgID).Return(false, nil)
				mockRepo.On("GetByID", mock.Anything, id).Return(msg, nil)
				mockWebhook.On("SendMessage", mock.Anything, msg.Content, msg.To).Return(&ports.WebhookResponse{MessageID: "webhook-123"}, nil)
				mockIdempotency.On("StoreWebhookMessageID", mock.Anything, msgID, "webhook-123", 24*time.Hour).Return(nil)
				mockIdempotency.On("MarkAsProcessed", mock.Anything, msgID).Return(nil)
				mockRepo.On("UpdateStatus", mock.Anything, id, models.StatusSent).Return(nil)
			},
			expectedError:  false,
			messageContent: "test content",
			messageTo:      "test@example.com",
		},
		{
			name: "duplicate message",
			setupMocks: func(msgID string) {
				mockIdempotency.On("IsProcessed", mock.Anything, msgID).Return(true, nil)
				id, _ := primitive.ObjectIDFromHex(msgID)
				mockRepo.On("UpdateStatus", mock.Anything, id, models.StatusDuplicate).Return(nil)
			},
			expectedError:  false,
			messageContent: "test content",
			messageTo:      "test@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgID := primitive.NewObjectID().Hex()
			queueMsg := contracts.QueueMessage{
				ID:      msgID,
				Content: tt.messageContent,
				To:      tt.messageTo,
			}
			msgBody, _ := json.Marshal(queueMsg)
			delivery := amqp.Delivery{Body: msgBody}

			tt.setupMocks(msgID)

			err := service.processMessage(delivery)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
			mockQueue.AssertExpectations(t)
			mockIdempotency.AssertExpectations(t)
			mockWebhook.AssertExpectations(t)
		})
	}
}

func TestProcessorService_HandleStaleMessages(t *testing.T) {
	mockRepo := new(mocks.MockMessageRepository)
	mockQueue := new(mocks.MockMessageQueue)
	mockIdempotency := new(mocks.MockIdempotencyService)
	mockWebhook := new(mocks.MockWebhookClient)
	processor := domain.NewMessageProcessor(3, 4*time.Minute)

	service := NewProcessorService(processor, mockRepo, mockQueue, mockIdempotency, mockWebhook)

	staleDuration := 4 * time.Minute
	staleMessages := []models.Message{
		{
			ID:         primitive.NewObjectID(),
			Content:    "stale content",
			To:         "stale@example.com",
			RetryCount: 1,
			UpdatedAt:  time.Now().Add(-5 * time.Minute),
		},
	}

	mockRepo.On("FindStaleProcessingMessages", mock.Anything, staleDuration).Return(staleMessages, nil)
	mockQueue.On("MoveToDeadLetter", mock.Anything).Return(nil)
	mockRepo.On("UpdateStatus", mock.Anything, staleMessages[0].ID, models.StatusFailed).Return(nil)

	err := service.checkStaleMessages()
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
	mockQueue.AssertExpectations(t)
}

func TestProcessorService_HandleWebhookError(t *testing.T) {
	mockRepo := new(mocks.MockMessageRepository)
	mockQueue := new(mocks.MockMessageQueue)
	mockIdempotency := new(mocks.MockIdempotencyService)
	mockWebhook := new(mocks.MockWebhookClient)
	processor := domain.NewMessageProcessor(3, 4*time.Minute)

	service := NewProcessorService(processor, mockRepo, mockQueue, mockIdempotency, mockWebhook)

	msgID := primitive.NewObjectID()
	msg := &models.Message{
		ID:         msgID,
		Content:    "test content",
		To:         "test@example.com",
		RetryCount: 0,
		UpdatedAt:  time.Now(),
	}

	delivery := amqp.Delivery{}
	testErr := assert.AnError

	mockRepo.On("IncrementRetryCount", mock.Anything, msgID).Return(nil)
	mockRepo.On("UpdateStatus", mock.Anything, msgID, models.StatusProcessing).Return(nil)
	mockRepo.On("GetByID", mock.Anything, msgID).Return(msg, nil)
	mockQueue.On("MoveToRetryQueue", &delivery).Return(nil)

	err := service.handleWebhookError(delivery, msg, testErr)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), testErr.Error())

	mockRepo.AssertExpectations(t)
	mockQueue.AssertExpectations(t)
} 