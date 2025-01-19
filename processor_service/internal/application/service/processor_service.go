package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Furkan-Gulsen/reliable_messaging_system/processor_service/internal/application/ports"
	"github.com/Furkan-Gulsen/reliable_messaging_system/processor_service/internal/domain"
	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/models"
	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/ports/mongodb/interfaces"
	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/ports/rabbitmq/contracts"
	rabbitPort "github.com/Furkan-Gulsen/reliable_messaging_system/shared/ports/rabbitmq/interfaces"
	redisPort "github.com/Furkan-Gulsen/reliable_messaging_system/shared/ports/redis/interfaces"

	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProcessorService struct {
	processor          *domain.MessageProcessor
	repository         interfaces.MessageRepository
	queue             rabbitPort.MessageQueue
	idempotencyService redisPort.IdempotencyServicePort
	webhookClient      ports.WebhookClient
	done              chan bool
}

func NewProcessorService(
	processor *domain.MessageProcessor,
	repository interfaces.MessageRepository,
	queue rabbitPort.MessageQueue,
	idempotencyService redisPort.IdempotencyServicePort,
	webhookClient ports.WebhookClient,
) *ProcessorService {
	return &ProcessorService{
		processor:          processor,
		repository:         repository,
		queue:             queue,
		idempotencyService: idempotencyService,
		webhookClient:      webhookClient,
		done:              make(chan bool),
	}
}

func (s *ProcessorService) Start() {
	log.Println("Message Processor Service started")

	go s.monitorStaleMessages()

	messages, err := s.queue.ConsumeMessages(contracts.MainQueueName)
	if err != nil {
		log.Fatalf("Failed to start consuming messages: %v", err)
	}

	for {
		select {
		case msg := <-messages:
			if err := s.processMessage(msg); err != nil {
				log.Printf("Error processing message: %v", err)
			}
		case <-s.done:
			return
		}
	}
}

func (s *ProcessorService) Stop() {
	close(s.done)
	s.queue.Close()
}

func (s *ProcessorService) processMessage(delivery amqp.Delivery) error {
	var queueMsg contracts.QueueMessage

	if delivery.DeliveryTag > 0 {
		if err := delivery.Ack(false); err != nil {
			log.Printf("Failed to ack message: %v", err)
		}
	}

	if err := s.handleMessageProcessing(delivery, queueMsg); err != nil {
		return fmt.Errorf("failed to process message: %v", err)
	}

	return nil
}

func (s *ProcessorService) handleMessageProcessing(delivery amqp.Delivery, queueMsg contracts.QueueMessage) error {
	if err := json.Unmarshal(delivery.Body, &queueMsg); err != nil {
		return s.handleMalformedMessage(delivery, err)
	}

	processed, err := s.idempotencyService.IsProcessed(context.Background(), queueMsg.ID)
	if err != nil {
		delivery.Nack(false, true)
		return err
	}

	if processed {
		return s.handleDuplicateMessage(queueMsg)
	}

	msgID, err := primitive.ObjectIDFromHex(queueMsg.ID)
	if err != nil {
		return s.handleInvalidID(delivery, err)
	}

	msg, err := s.repository.GetByID(context.Background(), msgID)
	if err != nil {
		log.Printf("Failed to get message from MongoDB: %v", err)
		delivery.Nack(false, true)
		return err
	}

	result := s.processor.ShouldProcessMessage(msg)
	if !result.Success {
		if result.IsStale {
			return s.handleStaleMessage(delivery, msg)
		}
		return s.handleMaxRetriesReached(delivery, msg)
	}

	webhookResp, err := s.webhookClient.SendMessage(context.Background(), msg.Content, msg.To)
	if err != nil {
		return s.handleWebhookError(delivery, msg, err)
	}

	if webhookResp != nil && webhookResp.MessageID != "" {
		if err := s.idempotencyService.StoreWebhookMessageID(context.Background(), queueMsg.ID, webhookResp.MessageID, 24*time.Hour); err != nil {
			log.Printf("Failed to store webhook messageId in Redis: %v", err)
		}
	} else {
		log.Printf("No messageId received from webhook for message %s", queueMsg.ID)
	}

	return s.handleSuccessfulProcessing(queueMsg, msgID)
}

func (s *ProcessorService) handleMalformedMessage(delivery amqp.Delivery, err error) error {
	if err := s.queue.MoveToDeadLetter(&delivery); err != nil {
		log.Printf("Failed to move malformed message to DLQ: %v", err)
	}
	return err
}

func (s *ProcessorService) handleDuplicateMessage(queueMsg contracts.QueueMessage) error {
	msgID, err := primitive.ObjectIDFromHex(queueMsg.ID)
	if err != nil {
		return err
	}

	if err := s.repository.UpdateStatus(context.Background(), msgID, models.StatusDuplicate); err != nil {
		log.Printf("Failed to update message status to duplicate: %v", err)
	}

	log.Printf("Message %s marked as duplicate", queueMsg.ID)
	return nil
}

func (s *ProcessorService) handleInvalidID(delivery amqp.Delivery, err error) error {
	if err := s.queue.MoveToDeadLetter(&delivery); err != nil {
		log.Printf("Failed to move message with invalid ID to DLQ: %v", err)
	}
	return err
}

func (s *ProcessorService) handleMaxRetriesReached(delivery amqp.Delivery, msg *models.Message) error {
	if err := s.queue.MoveToDeadLetter(&delivery); err != nil {
		log.Printf("Failed to move message to DLQ: %v", err)
	}
	if err := s.repository.UpdateStatus(context.Background(), msg.ID, models.StatusFailed); err != nil {
		log.Printf("Failed to update message status to failed: %v", err)
	}
	return fmt.Errorf("message reached max retry count")
}

func (s *ProcessorService) handleStaleMessage(delivery amqp.Delivery, msg *models.Message) error {
	if err := s.queue.MoveToDeadLetter(&delivery); err != nil {
		log.Printf("Failed to move stale message to DLQ: %v", err)
	}
	if err := s.repository.UpdateStatus(context.Background(), msg.ID, models.StatusFailed); err != nil {
		log.Printf("Failed to update stale message status to failed: %v", err)
	}
	return fmt.Errorf("message is stale")
}

func (s *ProcessorService) handleWebhookError(delivery amqp.Delivery, msg *models.Message, err error) error {
	log.Printf("Failed to send message %s to webhook (attempt %d): %v", msg.ID.Hex(), msg.RetryCount+1, err)
	
	if err := s.repository.IncrementRetryCount(context.Background(), msg.ID); err != nil {
		log.Printf("Failed to increment retry count: %v", err)
		return err
	}

	if err := s.repository.UpdateStatus(context.Background(), msg.ID, models.StatusProcessing); err != nil {
		log.Printf("Failed to update message status: %v", err)
		return err
	}

	updatedMsg, getErr := s.repository.GetByID(context.Background(), msg.ID)
	if getErr != nil {
		log.Printf("Failed to get updated message: %v", getErr)
		return getErr
	}

	if updatedMsg.RetryCount >= s.processor.GetMaxRetries() {
		if err := s.queue.MoveToDeadLetter(&delivery); err != nil {
			log.Printf("Failed to move message to DLQ: %v", err)
		}
		if err := s.repository.UpdateStatus(context.Background(), msg.ID, models.StatusFailed); err != nil {
			log.Printf("Failed to update message status to failed: %v", err)
		}
		return fmt.Errorf("message reached max retry count: %v", err)
	}

	if err := s.queue.MoveToRetryQueue(&delivery); err != nil {
		log.Printf("Failed to move message to retry queue: %v", err)
		return err
	}

	log.Printf("Message %s moved to retry queue (attempt %d)", msg.ID.Hex(), updatedMsg.RetryCount)
	return err
}

func (s *ProcessorService) handleSuccessfulProcessing(queueMsg contracts.QueueMessage, msgID primitive.ObjectID) error {
	if err := s.idempotencyService.MarkAsProcessed(context.Background(), queueMsg.ID); err != nil {
		return err
	}

	if err := s.repository.UpdateStatus(context.Background(), msgID, models.StatusSent); err != nil {
		return err
	}

	log.Printf("Successfully processed message %s", queueMsg.ID)
	return nil
}

func (s *ProcessorService) monitorStaleMessages() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.checkStaleMessages(); err != nil {
				log.Printf("Error checking stale messages: %v", err)
			}
		case <-s.done:
			return
		}
	}
}

func (s *ProcessorService) checkStaleMessages() error {
	staleDuration := 4 * time.Minute
	
	messages, err := s.repository.FindStaleProcessingMessages(context.Background(), staleDuration)
	if err != nil {
		return fmt.Errorf("failed to find stale messages: %v", err)
	}

	for _, msg := range messages {
		if err := s.handleStaleMessageRecovery(&msg); err != nil {
			log.Printf("Failed to handle stale message %s: %v", msg.ID.Hex(), err)
		}
	}

	return nil
}

func (s *ProcessorService) handleStaleMessageRecovery(msg *models.Message) error {
	log.Printf("Found stale message %s in processing state", msg.ID.Hex())

	msgBody, err := json.Marshal(contracts.QueueMessage{
		ID:      msg.ID.Hex(),
		Content: msg.Content,
		Retry:   msg.RetryCount,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal message for DLQ: %v", err)
	}

	delivery := amqp.Delivery{Body: msgBody}
	if err := s.queue.MoveToDeadLetter(&delivery); err != nil {
		return fmt.Errorf("failed to move stale message to DLQ: %v", err)
	}

	if err := s.repository.UpdateStatus(context.Background(), msg.ID, models.StatusFailed); err != nil {
		return fmt.Errorf("failed to update stale message status to failed: %v", err)
	}

	log.Printf("Successfully moved stale message %s to DLQ and marked as failed", msg.ID.Hex())
	return nil
} 