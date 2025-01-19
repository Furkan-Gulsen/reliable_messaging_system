package webhook

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Furkan-Gulsen/reliable_messaging_system/processor_service/internal/application/ports"
	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/ratelimit"

	"github.com/sony/gobreaker"
)

type httpWebhookClient struct {
	client      *http.Client
	baseURL     string
	rateLimiter *ratelimit.RateLimiter
	cb          *gobreaker.CircuitBreaker
}

func NewHTTPWebhookClient(webhookURL string, timeout time.Duration) ports.WebhookClient {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "webhook-client",
		MaxRequests: 3,
		Interval:    10 * time.Second,  
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			fmt.Printf("Circuit breaker %s state changed from %s to %s\n", name, from, to)
		},
	})

	return &httpWebhookClient{
		baseURL: webhookURL,
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
		rateLimiter: ratelimit.NewRateLimiter(50, 100), 
		cb:          cb,
	}
}

func (c *httpWebhookClient) SendMessage(ctx context.Context, content string, to string) (*ports.WebhookResponse, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %v", err)
	}

	result, err := c.cb.Execute(func() (interface{}, error) {
		jsonContent := map[string]string{
			"to":      to,
			"content": content,
		}
		
		jsonBytes, err := json.Marshal(jsonContent)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal content: %v", err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(jsonBytes))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %v", err)
		}
		
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("webhook request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("webhook returned error status: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %v", err)
		}

		var webhookResp ports.WebhookResponse
		if err := json.Unmarshal(body, &webhookResp); err != nil {
			return &ports.WebhookResponse{
				Message:   "",
				MessageID: "",
			}, nil
		}

		return &webhookResp, nil
	})

	if err != nil {
		return nil, fmt.Errorf("circuit breaker error: %v", err)
	}

	webhookResp, ok := result.(*ports.WebhookResponse)
	if !ok {
		return &ports.WebhookResponse{}, nil
	}

	return webhookResp, nil
} 