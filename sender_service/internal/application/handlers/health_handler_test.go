package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Furkan-Gulsen/reliable_messaging_system/sender_service/internal/application/service"
)

type MockHealthService struct {
	mock.Mock
}

func (m *MockHealthService) CheckHealth() service.HealthStatus {
	args := m.Called()
	return args.Get(0).(service.HealthStatus)
}

func TestHealthHandler_GetStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupMock      func(*MockHealthService)
		expectedStatus int
		expectedBody   service.HealthStatus
	}{
		{
			name: "all services healthy",
			setupMock: func(m *MockHealthService) {
				m.On("CheckHealth").Return(service.HealthStatus{
					MongoDB:   true,
					RabbitMQ:  true,
					Service:   true,
				})
			},
			expectedStatus: http.StatusOK,
			expectedBody: service.HealthStatus{
				MongoDB:   true,
				RabbitMQ:  true,
				Service:   true,
			},
		},
		{
			name: "some services unhealthy",
			setupMock: func(m *MockHealthService) {
				m.On("CheckHealth").Return(service.HealthStatus{
					MongoDB:   true,
					RabbitMQ:  false,
					Service:   false,
				})
			},
			expectedStatus: http.StatusOK,
			expectedBody: service.HealthStatus{
				MongoDB:   true,
				RabbitMQ:  false,
				Service:   false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockHealthService)
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			handler := NewHealthHandler(mockService)
			router := gin.New()
			router.GET("/status", handler.GetStatus)

			req := httptest.NewRequest(http.MethodGet, "/status", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)

			var response service.HealthStatus
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedBody, response)
		})
	}
} 