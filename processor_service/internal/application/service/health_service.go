package service

import (
	"context"
	"time"

	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/ports/mongodb/interfaces"
	rabbitPort "github.com/Furkan-Gulsen/reliable_messaging_system/shared/ports/rabbitmq/interfaces"
	redisPort "github.com/Furkan-Gulsen/reliable_messaging_system/shared/ports/redis/interfaces"
)

type HealthStatus struct {
	MongoDB   bool `json:"mongodb"`
	RabbitMQ  bool `json:"rabbitmq"`
	Redis     bool `json:"redis"`
	Service   bool `json:"service"`
}

type HealthService struct {
	repository         interfaces.MessageRepository
	queue             rabbitPort.MessageQueue
	idempotencyService redisPort.IdempotencyServicePort
}

func NewHealthService(
	repository interfaces.MessageRepository,
	queue rabbitPort.MessageQueue,
	idempotencyService redisPort.IdempotencyServicePort,
) *HealthService {
	return &HealthService{
		repository:         repository,
		queue:             queue,
		idempotencyService: idempotencyService,
	}
}

func (s *HealthService) CheckHealth() HealthStatus {
	status := HealthStatus{
		Service: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.repository.ListMessages(ctx)
	status.MongoDB = err == nil

	_, err = s.queue.GetDLQMessageCount()
	status.RabbitMQ = err == nil

	_, err = s.idempotencyService.IsProcessed(ctx, "health-check")
	status.Redis = err == nil

	return status
} 