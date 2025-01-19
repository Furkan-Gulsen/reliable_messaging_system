package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/Furkan-Gulsen/reliable_messaging_system/sender_service/internal/application/ports"
	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/models"

	"github.com/gin-gonic/gin"
)

type MessageHandler struct {
	service ports.MessageService
}

type SendMessageRequest struct {
	To      string `json:"to" binding:"required" example:"+90111111111"`
	Content string `json:"content" binding:"required" example:"project1"`
}

type SendMessageResponse struct {
	Message   string `json:"message" example:"Accepted"`
	MessageId string `json:"messageId" example:"67f2f8a8-ea58-4ed0-a6f9-ff217df4d849"`
}

type ListMessagesResponse struct {
	Messages []models.Message `json:"messages"`
}

func NewMessageHandler(service ports.MessageService) *MessageHandler {
	return &MessageHandler{
		service: service,
	}
}

// SendMessage handles message creation requests
// @Summary Send a new message
// @Description Send a new message to be processed
// @Tags messages
// @Accept json
// @Produce json
// @Param message body SendMessageRequest true "Message to send"
// @Success 200 {object} SendMessageResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /messages [post]
func (h *MessageHandler) SendMessage(c *gin.Context) {
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	id, err := h.service.CreateMessage(ctx, req.Content, req.To)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, SendMessageResponse{Message: "Accepted", MessageId: id.Hex()})
}

// ListMessages handles message listing requests
// @Summary List all messages
// @Description Get all messages with their current status
// @Tags messages
// @Produce json
// @Success 200 {object} ListMessagesResponse
// @Failure 500 {object} map[string]string
// @Router /messages [get]
func (h *MessageHandler) ListMessages(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	messages, err := h.service.ListMessages(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ListMessagesResponse{Messages: messages})
}

// StartScheduler handles scheduler start requests
// @Summary Start scheduler
// @Description Start the message processing scheduler
// @Tags scheduler
// @Produce json
// @Success 200 {object} map[string]string
// @Router /scheduler/start [post]
func (h *MessageHandler) StartScheduler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	h.service.StartScheduler(ctx)
	c.JSON(http.StatusOK, gin.H{"status": "scheduler started"})
}

// StopScheduler handles scheduler stop requests
// @Summary Stop scheduler
// @Description Stop the message processing scheduler
// @Tags scheduler
// @Produce json
// @Success 200 {object} map[string]string
// @Router /scheduler/stop [post]
func (h *MessageHandler) StopScheduler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	h.service.StopScheduler(ctx)
	c.JSON(http.StatusOK, gin.H{"status": "scheduler stopped"})
} 