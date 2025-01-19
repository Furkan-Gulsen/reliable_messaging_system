package contracts

const (
	MainQueueName    = "messages"
	RetryQueueName   = "messages.retry"
	DLQQueueName     = "messages.dlq"
	MainExchange     = "messages.exchange"
	RetryExchange    = "messages.retry.exchange"
	MaxRetryAttempts = 5
) 