package service

import (
	"context"
	"fmt"
	"log"
	"time"

	localDomain "github.com/Furkan-Gulsen/reliable_messaging_system/sender_service/internal/domain"
	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/models"
	mongoPort "github.com/Furkan-Gulsen/reliable_messaging_system/shared/ports/mongodb/interfaces"
	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/ports/rabbitmq/contracts"
	rabbitPort "github.com/Furkan-Gulsen/reliable_messaging_system/shared/ports/rabbitmq/interfaces"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SenderService struct {
	sender     *localDomain.MessageSender
	repository mongoPort.MessageRepository
	queue      rabbitPort.MessageQueue
	scheduler  *MessageScheduler
}

func NewSenderService(
	sender *localDomain.MessageSender,
	repository mongoPort.MessageRepository,
	queue rabbitPort.MessageQueue,
) *SenderService {
	service := &SenderService{
		sender:     sender,
		repository: repository,
		queue:      queue,
	}
	service.scheduler = NewMessageScheduler(service)
	return service
}

func (s *SenderService) CreateMessage(ctx context.Context, content string, to string) (primitive.ObjectID, error) {
	msg := s.sender.PrepareMessage(content, to)
	
	if err := s.repository.CreateMessage(ctx, msg); err != nil {
		return primitive.NilObjectID, fmt.Errorf("failed to create message: %v", err)
	}

	return msg.ID, nil
}

func (s *SenderService) ListMessages(ctx context.Context) ([]models.Message, error) {
	return s.repository.ListMessages(ctx)
}

func (s *SenderService) StartScheduler(ctx context.Context) {
	s.scheduler.Start()
}

func (s *SenderService) StopScheduler(ctx context.Context) {
	s.scheduler.Stop()
}

type MessageScheduler struct {
	service    *SenderService
	done       chan bool
	isRunning  bool
}

func NewMessageScheduler(service *SenderService) *MessageScheduler {
	return &MessageScheduler{
		service:   service,
		done:      make(chan bool),
		isRunning: false,
	}
}

func (s *MessageScheduler) Start() {
	if s.isRunning {
		return
	}

	s.isRunning = true
	go s.run()
}

func (s *MessageScheduler) Stop() {
	if !s.isRunning {
		return
	}

	s.isRunning = false
	s.done <- true
}

func (s *MessageScheduler) run() {
	ticker := time.NewTicker(s.service.sender.GetCheckInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.processUnsentMessages(); err != nil {
				log.Printf("Error processing unsent messages: %v", err)
			}
		case <-s.done:
			return
		}
	}
}

func (s *MessageScheduler) processUnsentMessages() error {
	ctx := context.Background()
	
	log.Println("Checking for unsent messages...")
	messages, err := s.service.repository.FindUnsentMessages(ctx, s.service.sender.GetBatchSize())
	if err != nil {
		return fmt.Errorf("failed to find unsent messages: %v", err)
	}

	log.Printf("Found %d unsent messages", len(messages))

	for _, msg := range messages {
		if err := s.handleMessage(ctx, &msg); err != nil {
			log.Printf("Failed to handle message %s: %v", msg.ID.Hex(), err)
			continue
		}
	}

	return nil
}

func (s *MessageScheduler) handleMessage(ctx context.Context, msg *models.Message) error {
	queueMsg := contracts.QueueMessage{
		ID:      msg.ID.Hex(),
		Content: msg.Content,
		To:      msg.To,
		Retry:   msg.RetryCount,
	}

	log.Printf("Attempting to publish message %s to queue", msg.ID.Hex())
	if err := s.service.queue.PublishMessage(ctx, queueMsg); err != nil {
		return fmt.Errorf("failed to publish message: %v", err)
	}
	log.Printf("Successfully published message %s to queue", msg.ID.Hex())

	log.Printf("Updating status to processing for message %s", msg.ID.Hex())
	if err := s.service.repository.UpdateStatus(ctx, msg.ID, models.StatusProcessing); err != nil {
		return fmt.Errorf("failed to update message status: %v", err)
	}
	log.Printf("Successfully updated status to processing for message %s", msg.ID.Hex())

	return nil
} 