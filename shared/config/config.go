package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	MongoDB struct {
		URI      string
		Database string
	}
	Redis struct {
		URI      string
		Password string
		DB       int
	}
	RabbitMQ struct {
		URI string
	}
	Webhook struct {
		URL     string
		Timeout time.Duration
	}

	MessageProcessor struct {
		BatchSize     int
		PollInterval  time.Duration
		MaxRetries    int
		RetryInterval time.Duration
		DLQAlertThreshold int
	}
}

func LoadConfig() *Config {
	cfg := &Config{}

	cfg.MongoDB.URI = getEnv("MONGODB_URI", "mongodb://localhost:27018")
	cfg.MongoDB.Database = getEnv("MONGODB_DATABASE", "message_system")

	cfg.Redis.URI = getEnv("REDIS_URI", "localhost:6380")
	cfg.Redis.Password = getEnv("REDIS_PASSWORD", "")
	cfg.Redis.DB = getEnvAsInt("REDIS_DB", 0)

	cfg.RabbitMQ.URI = getEnv("RABBITMQ_URI", "amqp://guest:guest@localhost:5672/")

	cfg.Webhook.URL = getEnv("WEBHOOK_URL", "http://localhost:8080/webhook")
	cfg.Webhook.Timeout = time.Duration(getEnvAsInt("WEBHOOK_TIMEOUT_SECONDS", 30)) * time.Second

	cfg.MessageProcessor.BatchSize = getEnvAsInt("MESSAGE_BATCH_SIZE", 2)
	cfg.MessageProcessor.PollInterval = time.Duration(getEnvAsInt("POLL_INTERVAL_SECONDS", 120)) * time.Second
	cfg.MessageProcessor.MaxRetries = getEnvAsInt("MAX_RETRIES", 5)
	cfg.MessageProcessor.RetryInterval = time.Duration(getEnvAsInt("RETRY_INTERVAL_SECONDS", 10)) * time.Second
	cfg.MessageProcessor.DLQAlertThreshold = getEnvAsInt("DLQ_ALERT_THRESHOLD", 10)

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
} 