package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/models"
	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/ports/mongodb/interfaces"

	"github.com/sony/gobreaker"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoMessageRepository struct {
	collection *mongo.Collection
	cb         *gobreaker.CircuitBreaker
}

func NewMessageRepository(db *mongo.Database) interfaces.MessageRepository {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "mongodb-repository",
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

	return &mongoMessageRepository{
		collection: db.Collection("messages"),
		cb:         cb,
	}
}

func (r *mongoMessageRepository) Save(ctx context.Context, message *models.Message) error {
	_, err := r.cb.Execute(func() (interface{}, error) {
		if message.ID.IsZero() {
			message.ID = primitive.NewObjectID()
		}
		message.CreatedAt = time.Now()
		
		_, err := r.collection.InsertOne(ctx, message)
		return nil, err
	})

	if err != nil {
		return fmt.Errorf("circuit breaker error: %v", err)
	}

	return nil
}

func (r *mongoMessageRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Message, error) {
	result, err := r.cb.Execute(func() (interface{}, error) {
		var message models.Message
		err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&message)
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		return &message, nil
	})

	if err != nil {
		return nil, fmt.Errorf("circuit breaker error: %v", err)
	}

	if result == nil {
		return nil, nil
	}

	return result.(*models.Message), nil
}

func (r *mongoMessageRepository) FindUnsentMessages(ctx context.Context, limit int) ([]models.Message, error) {
	result, err := r.cb.Execute(func() (interface{}, error) {
		filter := bson.M{"status": models.StatusUnsent}
		findOptions := options.Find().SetLimit(int64(limit))

		cursor, err := r.collection.Find(ctx, filter, findOptions)
		if err != nil {
			return nil, err
		}
		defer cursor.Close(ctx)

		var messages []models.Message
		if err = cursor.All(ctx, &messages); err != nil {
			return nil, err
		}
		return messages, nil
	})

	if err != nil {
		return nil, fmt.Errorf("circuit breaker error: %v", err)
	}

	return result.([]models.Message), nil
}

func (r *mongoMessageRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.MessageStatus) error {
	_, err := r.cb.Execute(func() (interface{}, error) {
		update := bson.M{
			"$set": bson.M{
				"status": status,
				"updated_at": time.Now(),
			},
		}
		_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
		return nil, err
	})

	if err != nil {
		return fmt.Errorf("circuit breaker error: %v", err)
	}

	return nil
}

func (r *mongoMessageRepository) IncrementRetryCount(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.cb.Execute(func() (interface{}, error) {
		update := bson.M{
			"$inc": bson.M{"retry_count": 1},
			"$set": bson.M{"updated_at": time.Now()},
		}
		_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
		return nil, err
	})

	if err != nil {
		return fmt.Errorf("circuit breaker error: %v", err)
	}

	return nil
}

func (r *mongoMessageRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Message, error) {
	var message models.Message
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&message)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &message, nil
}

func (r *mongoMessageRepository) CreateMessage(ctx context.Context, msg *models.Message) error {
	_, err := r.collection.InsertOne(ctx, msg)
	return err
}

func (r *mongoMessageRepository) ListMessages(ctx context.Context) ([]models.Message, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	if err = cursor.All(ctx, &messages); err != nil {
		return nil, err
	}

	return messages, nil
}

func (r *mongoMessageRepository) FindStaleProcessingMessages(ctx context.Context, staleDuration time.Duration) ([]models.Message, error) {
	staleTime := time.Now().Add(-staleDuration)
	
	filter := bson.M{
		"status": models.StatusProcessing,
		"updated_at": bson.M{"$lt": staleTime},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	if err = cursor.All(ctx, &messages); err != nil {
		return nil, err
	}

	return messages, nil
} 