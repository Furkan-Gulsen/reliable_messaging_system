package ports

import (
	"context"

	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MessageService interface {
	CreateMessage(ctx context.Context, content string, to string) (primitive.ObjectID, error)
	ListMessages(ctx context.Context) ([]models.Message, error)
	StartScheduler(ctx context.Context)
	StopScheduler(ctx context.Context)
} 