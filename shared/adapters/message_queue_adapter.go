package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/ports/rabbitmq/contracts"
	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/ports/rabbitmq/interfaces"

	"github.com/streadway/amqp"
)

type rabbitMQAdapter struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewMessageQueue(url string) (interfaces.MessageQueue, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %v", err)
	}

	mq := &rabbitMQAdapter{
		conn:    conn,
		channel: ch,
	}

	if err := mq.setupQueues(); err != nil {
		mq.Close()
		return nil, err
	}

	return mq, nil
}

func (mq *rabbitMQAdapter) setupQueues() error {
	err := mq.channel.ExchangeDeclare(
		contracts.MainExchange, 
		"direct",     
		true,         
		false,        
		false,        
		false,        
		nil,          
	)
	if err != nil {
		return fmt.Errorf("failed to declare main exchange: %v", err)
	}

	// Retry Exchange
	err = mq.channel.ExchangeDeclare(
		contracts.RetryExchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare retry exchange: %v", err)
	}

	// Main Queue
	args := amqp.Table{
		"x-dead-letter-exchange":    contracts.RetryExchange,
		"x-dead-letter-routing-key": contracts.RetryQueueName,
	}
	_, err = mq.channel.QueueDeclare(
		contracts.MainQueueName, 
		true,          
		false,         
		false,         
		false,         
		args,          
	)
	if err != nil {
		return fmt.Errorf("failed to declare main queue: %v", err)
	}

	// Retry Queue with TTL and dead-letter to main queue
	retryArgs := amqp.Table{
		"x-message-ttl":             10000, 
		"x-dead-letter-exchange":    contracts.MainExchange,
		"x-dead-letter-routing-key": contracts.MainQueueName,
	}
	_, err = mq.channel.QueueDeclare(
		contracts.RetryQueueName,
		true,
		false,
		false,
		false,
		retryArgs,
	)
	if err != nil {
		return fmt.Errorf("failed to declare retry queue: %v", err)
	}

	// DLQ - Dead Letter Queue
	_, err = mq.channel.QueueDeclare(
		contracts.DLQQueueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare DLQ: %v", err)
	}

	// Bind queues to exchanges
	err = mq.channel.QueueBind(
		contracts.MainQueueName, 
		contracts.MainQueueName, 
		contracts.MainExchange, 
		false, 
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind main queue: %v", err)
	}

	err = mq.channel.QueueBind(
		contracts.RetryQueueName, 
		contracts.RetryQueueName, 
		contracts.RetryExchange, 
		false, 
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind retry queue: %v", err)
	}

	return nil
}

func (mq *rabbitMQAdapter) PublishMessage(ctx context.Context, msg contracts.QueueMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	return mq.channel.Publish(
		contracts.MainExchange,  
		contracts.MainQueueName, 
		false,         
		false,         
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
			Timestamp:    time.Now(),
		},
	)
}

func (mq *rabbitMQAdapter) ConsumeMessages(queueName string) (<-chan amqp.Delivery, error) {
	return mq.channel.Consume(
		queueName, 
		"",        
		false,     
		false,     
		false,     
		false,     
		nil,       
	)
}

func (mq *rabbitMQAdapter) MoveToDeadLetter(msg *amqp.Delivery) error {
	headers := amqp.Table{}
	if msg.Headers != nil {
		headers = msg.Headers
	}

	return mq.channel.Publish(
		"",          
		contracts.DLQQueueName,
		false,       
		false,       
		amqp.Publishing{
			Headers:      headers,
			ContentType: msg.ContentType,
			Body:        msg.Body,
			Timestamp:   time.Now(),
		},
	)
}

func (mq *rabbitMQAdapter) MoveToRetryQueue(msg *amqp.Delivery) error {
	headers := amqp.Table{}
	if msg.Headers != nil {
		headers = msg.Headers
	}

	return mq.channel.Publish(
		contracts.RetryExchange,
		contracts.RetryQueueName,
		false,
		false,
		amqp.Publishing{
			Headers:      headers,
			ContentType:  msg.ContentType,
			Body:        msg.Body,
			Timestamp:   time.Now(),
		},
	)
}

func (mq *rabbitMQAdapter) GetDLQMessageCount() (int, error) {
	queue, err := mq.channel.QueueInspect(contracts.DLQQueueName)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect DLQ: %v", err)
	}
	return queue.Messages, nil
}

func (mq *rabbitMQAdapter) Close() {
	if mq.channel != nil {
		mq.channel.Close()
	}
	if mq.conn != nil {
		mq.conn.Close()
	}
} 