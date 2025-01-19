package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Furkan-Gulsen/reliable_messaging_system/processor_service/internal/application/ports"

	"github.com/stretchr/testify/assert"
)

func TestHTTPWebhookClient_SendMessage(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter)
		content        string
		to            string
		expectedResp   *ports.WebhookResponse
		expectError    bool
	}{
		{
			name: "successful message send",
			serverResponse: func(w http.ResponseWriter) {
				resp := ports.WebhookResponse{
					Message:   "Message sent successfully",
					MessageID: "test-message-id",
				}
				json.NewEncoder(w).Encode(resp)
			},
			content: "test content",
			to:      "test@example.com",
			expectedResp: &ports.WebhookResponse{
				Message:   "Message sent successfully",
				MessageID: "test-message-id",
			},
			expectError: false,
		},
		{
			name: "server error",
			serverResponse: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			content:      "test content",
			to:          "+905321234569",
			expectedResp: nil,
			expectError:  true,
		},
		{
			name: "invalid response format",
			serverResponse: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("invalid json"))
			},
			content: "test content",
			to:      "+905321132269",
			expectedResp: &ports.WebhookResponse{
				Message:   "",
				MessageID: "",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				var requestBody map[string]string
				err := json.NewDecoder(r.Body).Decode(&requestBody)
				assert.NoError(t, err)
				assert.Equal(t, tt.content, requestBody["content"])
				assert.Equal(t, tt.to, requestBody["to"])

				tt.serverResponse(w)
			}))
			defer server.Close()

			client := NewHTTPWebhookClient(server.URL, 5*time.Second)
			resp, err := client.SendMessage(context.Background(), tt.content, tt.to)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				if tt.expectedResp != nil {
					assert.Equal(t, tt.expectedResp.Message, resp.Message)
					assert.Equal(t, tt.expectedResp.MessageID, resp.MessageID)
				}
			}
		})
	}
}

func TestHTTPWebhookClient_SendMessage_CircuitBreaker(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewHTTPWebhookClient(server.URL, 5*time.Second)

	for i := 0; i < 5; i++ {
		_, err := client.SendMessage(context.Background(), "test", "+905321234569")
		assert.Error(t, err)
	}

	_, err := client.SendMessage(context.Background(), "test", "+905321234569")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker")
}

