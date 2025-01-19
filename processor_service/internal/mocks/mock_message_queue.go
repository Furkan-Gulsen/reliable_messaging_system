package mocks

import (
	"context"

	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/ports/rabbitmq/contracts"

	"github.com/streadway/amqp"
	"github.com/stretchr/testify/mock"
)

type MockMessageQueue struct {
	mock.Mock
}

func (m *MockMessageQueue) ConsumeMessages(queueName string) (<-chan amqp.Delivery, error) {
	args := m.Called(queueName)
	if ch, ok := args.Get(0).(<-chan amqp.Delivery); ok {
		return ch, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMessageQueue) MoveToDeadLetter(delivery *amqp.Delivery) error {
	args := m.Called(delivery)
	return args.Error(0)
}

func (m *MockMessageQueue) MoveToRetryQueue(delivery *amqp.Delivery) error {
	args := m.Called(delivery)
	return args.Error(0)
}

func (m *MockMessageQueue) GetDLQMessageCount() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

func (m *MockMessageQueue) Close() {
	m.Called()
}

func (m *MockMessageQueue) PublishMessage(ctx context.Context, message contracts.QueueMessage) error {
	args := m.Called(ctx, message)
	return args.Error(0)
} 