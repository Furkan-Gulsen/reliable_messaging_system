package interfaces

import (
	"context"
	"time"

	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MessageRepository interface {
	FindUnsentMessages(ctx context.Context, limit int) ([]models.Message, error)
	UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.MessageStatus) error
	IncrementRetryCount(ctx context.Context, id primitive.ObjectID) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Message, error)
	CreateMessage(ctx context.Context, msg *models.Message) error
	ListMessages(ctx context.Context) ([]models.Message, error)
	FindStaleProcessingMessages(ctx context.Context, staleDuration time.Duration) ([]models.Message, error)
} 