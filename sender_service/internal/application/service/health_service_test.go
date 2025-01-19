package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/models"
	mongoInterfaces "github.com/Furkan-Gulsen/reliable_messaging_system/shared/ports/mongodb/interfaces"
	rabbitInterfaces "github.com/Furkan-Gulsen/reliable_messaging_system/shared/ports/rabbitmq/interfaces"
)

type mockHealthRepository struct {
	mock.Mock
	mongoInterfaces.MessageRepository
}

func (m *mockHealthRepository) ListMessages(ctx context.Context) ([]models.Message, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Message), args.Error(1)
}

type mockHealthQueue struct {
	mock.Mock
	rabbitInterfaces.MessageQueue
}

func (m *mockHealthQueue) GetDLQMessageCount() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

func TestHealthService_CheckHealth(t *testing.T) {
	tests := []struct {
		name                string
		setupMocks         func(*mockHealthRepository, *mockHealthQueue)
		setupProcessorMock func(*httptest.Server)
		expectedStatus     HealthStatus
	}{
		{
			name: "all services healthy",
			setupMocks: func(repo *mockHealthRepository, queue *mockHealthQueue) {
				repo.On("ListMessages", mock.Anything).Return([]models.Message{}, nil)
				queue.On("GetDLQMessageCount").Return(0, nil)
			},
			setupProcessorMock: func(server *httptest.Server) {
				server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})
			},
			expectedStatus: HealthStatus{
				MongoDB:   true,
				RabbitMQ:  true,
				Service:   true,
			},
		},
		{
			name: "mongodb unhealthy",
			setupMocks: func(repo *mockHealthRepository, queue *mockHealthQueue) {
				repo.On("ListMessages", mock.Anything).Return([]models.Message{}, assert.AnError)
				queue.On("GetDLQMessageCount").Return(0, nil)
			},
			setupProcessorMock: func(server *httptest.Server) {
				server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})
			},
			expectedStatus: HealthStatus{
				MongoDB:   false,
				RabbitMQ:  true,
				Service:   true,
			},
		},
		{
			name: "rabbitmq unhealthy",
			setupMocks: func(repo *mockHealthRepository, queue *mockHealthQueue) {
				repo.On("ListMessages", mock.Anything).Return([]models.Message{}, nil)
				queue.On("GetDLQMessageCount").Return(0, assert.AnError)
			},
			setupProcessorMock: func(server *httptest.Server) {
				server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})
			},
			expectedStatus: HealthStatus{
				MongoDB:   true,
				RabbitMQ:  false,
				Service:   true,
			},
		},
		{
			name: "processor service unhealthy",
			setupMocks: func(repo *mockHealthRepository, queue *mockHealthQueue) {
				repo.On("ListMessages", mock.Anything).Return([]models.Message{}, nil)
				queue.On("GetDLQMessageCount").Return(0, nil)
			},
			setupProcessorMock: func(server *httptest.Server) {
				server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				})
			},
			expectedStatus: HealthStatus{
				MongoDB:   true,
				RabbitMQ:  true,
				Service:   false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockHealthRepository{}
			mockQueue := &mockHealthQueue{}
			if tt.setupMocks != nil {
				tt.setupMocks(mockRepo, mockQueue)
			}

			processorServer := httptest.NewServer(nil)
			defer processorServer.Close()
			if tt.setupProcessorMock != nil {
				tt.setupProcessorMock(processorServer)
			}

			service := &HealthService{
				repository: mockRepo,
				queue:      mockQueue,
				processorURL: fmt.Sprintf("%s/status", processorServer.URL),
			}
			status := service.CheckHealth()

			assert.Equal(t, tt.expectedStatus, status)
			mockRepo.AssertExpectations(t)
			mockQueue.AssertExpectations(t)
		})
	}
} 