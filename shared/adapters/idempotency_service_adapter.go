package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/ports/redis/interfaces"

	"github.com/go-redis/redis/v8"
	"github.com/sony/gobreaker"
)

type redisIdempotencyService struct {
	client *redis.Client
	cb     *gobreaker.CircuitBreaker
}

func NewIdempotencyService(client *redis.Client) interfaces.IdempotencyServicePort {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "redis-idempotency",
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

	return &redisIdempotencyService{
		client: client,
		cb:     cb,
	}
}

func (s *redisIdempotencyService) IsProcessed(ctx context.Context, messageID string) (bool, error) {
	result, err := s.cb.Execute(func() (interface{}, error) {
		key := fmt.Sprintf("inbox:%s", messageID)
		val, err := s.client.Get(ctx, key).Result()
		if err == redis.Nil {
			return false, nil
		}
		if err != nil {
			return false, err
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(val), &data); err != nil {
			return val == "processed", nil
		}

		status, ok := data["status"].(string)
		return ok && status == "processed", nil
	})

	if err != nil {
		return false, fmt.Errorf("circuit breaker error: %v", err)
	}

	return result.(bool), nil
}

func (s *redisIdempotencyService) MarkAsProcessed(ctx context.Context, messageID string) error {
	_, err := s.cb.Execute(func() (interface{}, error) {
		key := fmt.Sprintf("inbox:%s", messageID)
		data := map[string]interface{}{
			"status":    "processed",
			"timestamp": time.Now().Unix(),
		}
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal data: %v", err)
		}
		return nil, s.client.Set(ctx, key, string(jsonData), 24*time.Hour).Err()
	})

	if err != nil {
		return fmt.Errorf("circuit breaker error: %v", err)
	}

	return nil
}

func (s *redisIdempotencyService) StoreWebhookMessageID(ctx context.Context, messageID string, webhookMessageID string, expiration time.Duration) error {
	_, err := s.cb.Execute(func() (interface{}, error) {
		key := fmt.Sprintf("webhook:msg:%s", messageID)
		if err := s.client.Set(ctx, key, webhookMessageID, expiration).Err(); err != nil {
			return nil, fmt.Errorf("failed to store webhook messageId: %v", err)
		}
		return nil, nil
	})
	return err
} 