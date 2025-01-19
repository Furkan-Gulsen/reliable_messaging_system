package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/Furkan-Gulsen/reliable_messaging_system/docs"
	"github.com/Furkan-Gulsen/reliable_messaging_system/sender_service/internal/application/handlers"
	"github.com/Furkan-Gulsen/reliable_messaging_system/sender_service/internal/application/service"
	"github.com/Furkan-Gulsen/reliable_messaging_system/sender_service/internal/domain"
	"github.com/Furkan-Gulsen/reliable_messaging_system/sender_service/internal/infrastructure/middleware"
	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/adapters"
	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/config"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// @title Message Sender Service API
// @version 1.0
// @description This is the Message Sender Service API
// @host localhost:8080
// @BasePath /api/v1
// @schemes http
func main() {
	cfg := config.LoadConfig()
	ctx := context.Background()

	mongoClient, err := mongodbConnect(ctx, cfg.MongoDB.URI)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongodbDisconnect(ctx, mongoClient)

	db := mongoClient.Database(cfg.MongoDB.Database)
	messageRepo := adapters.NewMessageRepository(db)
	messageQueue, err := adapters.NewMessageQueue(cfg.RabbitMQ.URI)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	sender := domain.NewMessageSender(2, 2*time.Minute)
	senderService := service.NewSenderService(sender, messageRepo, messageQueue)
	messageHandler := handlers.NewMessageHandler(senderService)

	healthService := service.NewHealthService(messageRepo, messageQueue)
	healthHandler := handlers.NewHealthHandler(healthService)

	router := gin.Default()

	apiGroup := router.Group("/api/v1")
	apiGroup.Use(middleware.RateLimit(50, 100)) 


	// API routes
	apiGroup.POST("/messages", messageHandler.SendMessage)
	apiGroup.GET("/messages", messageHandler.ListMessages)
	apiGroup.POST("/scheduler/start", messageHandler.StartScheduler)
	apiGroup.POST("/scheduler/stop", messageHandler.StopScheduler)
	apiGroup.GET("/status", healthHandler.GetStatus)

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	go func() {
		if err := router.Run(":8080"); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down Message Sender Service...")
	senderService.StopScheduler(ctx)
	messageQueue.Close()
	if err := mongoClient.Disconnect(nil); err != nil {
		log.Printf("Error disconnecting from MongoDB: %v", err)
	}
} 

func mongodbConnect(ctx context.Context, uri string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return mongo.Connect(ctx, options.Client().ApplyURI(uri))
}

func mongodbDisconnect(ctx context.Context, client *mongo.Client) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return client.Disconnect(ctx)
}

