package handlers

import (
	"net/http"

	"github.com/Furkan-Gulsen/reliable_messaging_system/processor_service/internal/application/service"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	service *service.HealthService
}

func NewHealthHandler(service *service.HealthService) *HealthHandler {
	return &HealthHandler{
		service: service,
	}
}

// GetStatus handles health check requests
// @Summary Get service health status
// @Description Get the health status of the service and its dependencies (MongoDB, RabbitMQ, Redis)
// @Tags health
// @Produce json
// @Success 200 {object} service.HealthStatus
// @Router /status [get]
func (h *HealthHandler) GetStatus(c *gin.Context) {
	status := h.service.CheckHealth()
	c.JSON(http.StatusOK, status)
} 