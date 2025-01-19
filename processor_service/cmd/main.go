package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Furkan-Gulsen/reliable_messaging_system/processor_service/internal/application/handlers"
	"github.com/Furkan-Gulsen/reliable_messaging_system/processor_service/internal/application/service"
	"github.com/Furkan-Gulsen/reliable_messaging_system/processor_service/internal/domain"
	"github.com/Furkan-Gulsen/reliable_messaging_system/processor_service/internal/infrastructure/webhook"
	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/adapters"
	"github.com/Furkan-Gulsen/reliable_messaging_system/shared/config"

	"github.com/gin-gonic/gin"
	redisClient "github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	cfg := config.LoadConfig()

	mongoClient, err := mongo.Connect(nil, options.Client().ApplyURI(cfg.MongoDB.URI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(nil)

	db := mongoClient.Database(cfg.MongoDB.Database)
	messageRepo := adapters.NewMessageRepository(db)
	messageQueue, err := adapters.NewMessageQueue(cfg.RabbitMQ.URI)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	redisOpts := &redisClient.Options{
		Addr:     cfg.Redis.URI,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}
	redisConn := redisClient.NewClient(redisOpts)
	idempotencyService := adapters.NewIdempotencyService(redisConn)

	webhookClient := webhook.NewHTTPWebhookClient(cfg.Webhook.URL, cfg.Webhook.Timeout)

	processor := domain.NewMessageProcessor(cfg.MessageProcessor.MaxRetries, 4*time.Minute)

	processorService := service.NewProcessorService(
		processor,
		messageRepo,
		messageQueue,
		idempotencyService,
		webhookClient,
	)

	healthService := service.NewHealthService(messageRepo, messageQueue, idempotencyService)
	healthHandler := handlers.NewHealthHandler(healthService)

	router := gin.Default()
	router.GET("/status", healthHandler.GetStatus)

	go func() {
		if err := router.Run(":8081"); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	go processorService.Start()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down Message Processor Service...")
	processorService.Stop()
	messageQueue.Close()
	if err := redisConn.Close(); err != nil {
		log.Printf("Error closing Redis connection: %v", err)
	}
	if err := mongoClient.Disconnect(nil); err != nil {
		log.Printf("Error disconnecting from MongoDB: %v", err)
	}
} 