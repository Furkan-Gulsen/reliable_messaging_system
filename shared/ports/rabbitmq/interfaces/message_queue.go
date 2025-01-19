package interfaces

import (
	"context"

	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/ports/rabbitmq/contracts"

	"github.com/streadway/amqp"
)

type MessageQueue interface {
	PublishMessage(ctx context.Context, msg contracts.QueueMessage) error
	ConsumeMessages(queueName string) (<-chan amqp.Delivery, error)
	MoveToDeadLetter(msg *amqp.Delivery) error
	MoveToRetryQueue(msg *amqp.Delivery) error
	GetDLQMessageCount() (int, error)
	Close()
} 