package interfaces

import (
	"context"
	"time"
)

type IdempotencyServicePort interface {
	IsProcessed(ctx context.Context, messageID string) (bool, error)
	MarkAsProcessed(ctx context.Context, messageID string) error
	StoreWebhookMessageID(ctx context.Context, messageID string, webhookMessageID string, expiration time.Duration) error
} 