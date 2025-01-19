package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/models"
)

type MockSenderService struct {
	mock.Mock
}

func (m *MockSenderService) CreateMessage(ctx context.Context, content string, to string) (primitive.ObjectID, error) {
	args := m.Called(ctx, content, to)
	return args.Get(0).(primitive.ObjectID), args.Error(1)
}

func (m *MockSenderService) ListMessages(ctx context.Context) ([]models.Message, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Message), args.Error(1)
}

func (m *MockSenderService) StartScheduler(ctx context.Context) {
	m.Called(ctx)
}

func (m *MockSenderService) StopScheduler(ctx context.Context) {
	m.Called(ctx)
}

func TestMessageHandler_SendMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		request        SendMessageRequest
		setupMock      func(*MockSenderService)
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name: "successful message creation",
			request: SendMessageRequest{
				Content: "test content",
				To:      "+905321234567",
			},
			setupMock: func(m *MockSenderService) {
				id := primitive.NewObjectID()
				m.On("CreateMessage", mock.Anything, "test content", "+905321234567").Return(id, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: SendMessageResponse{
				Message: "Accepted",
			},
		},
		{
			name: "validation error - content too long",
			request: SendMessageRequest{
				Content: string(make([]byte, 251)),
				To:      "+905321234567",
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "validation error - invalid phone number format",
			request: SendMessageRequest{
				Content: "test content",
				To:      "05321234111",
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "validation error - empty required fields",
			request: SendMessageRequest{
				Content: "",
				To:      "",
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSenderService)
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			handler := NewMessageHandler(mockService)
			router := gin.New()
			router.POST("/messages", handler.SendMessage)

			reqBody, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)

			if tt.expectedBody != nil {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "Accepted", response["message"])
				assert.NotEmpty(t, response["messageId"])
			}
		})
	}
}

func TestMessageHandler_ListMessages(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupMock      func(*MockSenderService)
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name: "successful messages listing",
			setupMock: func(m *MockSenderService) {
				messages := []models.Message{
					{
						ID:         primitive.NewObjectID(),
						Content:    "test1",
						To:         "+905321234567",
						Status:     models.StatusUnsent,
						CreatedAt:  time.Now(),
						UpdatedAt:  time.Now(),
						RetryCount: 0,
					},
				}
				m.On("ListMessages", mock.Anything).Return(messages, nil)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSenderService)
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			handler := NewMessageHandler(mockService)
			router := gin.New()
			router.GET("/messages", handler.ListMessages)

			req := httptest.NewRequest(http.MethodGet, "/messages", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)

			if tt.expectedBody != nil {
				var response ListMessagesResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.Messages)
			}
		})
	}
}

func TestMessageHandler_Scheduler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("start scheduler", func(t *testing.T) {
		mockService := new(MockSenderService)
		mockService.On("StartScheduler", mock.Anything).Return()

		handler := NewMessageHandler(mockService)
		router := gin.New()
		router.POST("/scheduler/start", handler.StartScheduler)

		req := httptest.NewRequest(http.MethodPost, "/scheduler/start", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("stop scheduler", func(t *testing.T) {
		mockService := new(MockSenderService)
		mockService.On("StopScheduler", mock.Anything).Return()

		handler := NewMessageHandler(mockService)
		router := gin.New()
		router.POST("/scheduler/stop", handler.StopScheduler)

		req := httptest.NewRequest(http.MethodPost, "/scheduler/stop", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})
} 