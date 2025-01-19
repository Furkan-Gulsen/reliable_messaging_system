package service

import (
	"context"
	"net/http"
	"time"

	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/ports/mongodb/interfaces"
	rabbitPort "github.com/Furkan-Gulsen/reliable_messaging_system/shared/ports/rabbitmq/interfaces"
)

type HealthStatus struct {
	MongoDB   bool `json:"mongodb"`
	RabbitMQ  bool `json:"rabbitmq"`
	Service   bool `json:"service"`
}

type HealthService struct {
	repository   interfaces.MessageRepository
	queue        rabbitPort.MessageQueue
	processorURL string
}

func NewHealthService(repository interfaces.MessageRepository, queue rabbitPort.MessageQueue) *HealthService {
	return &HealthService{
		repository:   repository,
		queue:        queue,
		processorURL: "http://processor_service:8081/status",
	}
}

func (s *HealthService) CheckHealth() HealthStatus {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var status HealthStatus

	_, err := s.queue.GetDLQMessageCount()
	status.RabbitMQ = err == nil

	req, _ := http.NewRequestWithContext(ctx, "GET", s.processorURL, nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		status.Service = false
	} else {
		defer resp.Body.Close()
		status.Service = resp.StatusCode == http.StatusOK
	}

	_, err = s.repository.ListMessages(ctx)
	status.MongoDB = err == nil

	return status
}
