package ports

import "context"

type WebhookResponse struct {
	Message   string `json:"message"`
	MessageID string `json:"messageId"`
}

type WebhookClient interface {
	SendMessage(ctx context.Context, content string, to string) (*WebhookResponse, error)
} 