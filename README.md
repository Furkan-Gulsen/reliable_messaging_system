# Message Processing System

![system architecture](assets/structure.png)

A robust, distributed message processing system built with Go, implementing a microservices architecture. The system consists of two main services: a Sender Service and a Processor Service, working together to handle message delivery with reliability, scalability, and fault tolerance.

## Features

- **Distributed Architecture**
  - Sender Service for message ingestion and management
  - Processor Service for reliable message delivery
  - Asynchronous communication via RabbitMQ
  - Data persistence with MongoDB
  - Redis for idempotency and rate limiting

- **Reliability Features**
  - Outbox pattern implemented in MongoDB for reliable message publishing
  - Inbox pattern implemented in Redis to ensure idempotent processing
  - Circuit breaker pattern for external service calls
  - Rate limiting for API endpoints
  - Message retry mechanism with configurable attempts
  - Dead Letter Queue (DLQ) for failed messages
  - Stale message detection and recovery
  - Idempotent message processing

- **Monitoring & Health Checks**
  - Health check endpoints for all services
  - Message status tracking
  - DLQ monitoring
  - Detailed logging

## System Architecture

### Sender Service
- Handles incoming message requests
- Manages message lifecycle
- Implements message scheduling
- Provides message status tracking
- Exposes REST API endpoints

### Processor Service
- Processes messages from the queue
- Implements webhook delivery with retry logic
- Handles message idempotency
- Manages circuit breaking for external calls
- Provides rate limiting for outbound requests

## Patterns Used

### Outbox Pattern (MongoDB)
The Sender Service uses the **Outbox pattern** to ensure reliable message publishing. Messages are first stored in MongoDB with their state (`UNSENT`, `PROCESSING`, `SENT`, or `FAILED`). The outbox acts as a durable store to guarantee that messages are not lost in case of system failures. Key steps include:

1. Messages are stored in MongoDB when created.
2. The Sender Service periodically fetches unsent messages and publishes them to RabbitMQ.
3. MongoDB is updated to mark the message state as `PROCESSING` to prevent duplicate processing.

This pattern ensures that messages are published exactly once and guarantees consistency between the database and the message queue.

### Inbox Pattern (Redis)
The Processor Service uses the **Inbox pattern** to ensure idempotency during message processing. Redis acts as a fast, in-memory store to track processed messages and prevent duplicates. Key steps include:

1. When a message is consumed from RabbitMQ, its unique ID is checked in Redis.
2. If the ID exists in Redis, the message is skipped to avoid reprocessing.
3. If the ID does not exist, the message is processed, and its ID is stored in Redis with a time-to-live (TTL).

This pattern helps maintain the integrity of message processing even in scenarios involving retries or network failures.

## Prerequisites

- Go 1.19 or later
- Docker and Docker Compose
- MongoDB
- RabbitMQ
- Redis

## Getting Started

1. Clone the repository:
   ```bash
   git clone https://github.com/Furkan-Gulsen/reliable_messaging_system
   cd reliable_messaging_system
   ```

2. Install dependencies:
   ```bash
   make dev-deps
   ```

3. Start the infrastructure services:
   ```bash
   make docker-run
   ```

4. Build and run the services:
   ```bash
   make run
   ```

## API Endpoints

### Sender Service

#### Message Management
- `POST /messages`
  - Create a new message
  - Request body: `{"to": "+90111111111", "content": "message content"}`

- `GET /messages`
  - List all messages with their current status

#### Scheduler Management
- `POST /scheduler/start`
  - Start the message processing scheduler
- `POST /scheduler/stop`
  - Stop the message processing scheduler

#### Health Check
- `GET /status`
  - Get service health status including MongoDB and RabbitMQ connectivity

### Processor Service

- `GET /status`
  - Get service health status including webhook availability

## Configuration

The system can be configured through environment variables:

```env
# MongoDB
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=messages

# RabbitMQ
RABBITMQ_URI=amqp://guest:guest@localhost:5672/

# Redis
REDIS_URI=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# Webhook
WEBHOOK_URL=http://external-service/webhook
WEBHOOK_TIMEOUT=5s

# Message Processing
MAX_RETRIES=3
STALE_DURATION=4m
```

## Development

### Running Tests
```bash
make test
```

### Building Services
```bash
# Build both services
make build

# Clean build artifacts
make clean
```

### Docker Operations
```bash
# Build Docker images
make docker-build

# Start services with Docker
make docker-run

# Stop services
make docker-stop

# View logs
make logs
```

### Database Operations
```bash
# Seed test data
make seed
```

## Message Processing Flow

1. **Message Creation**
   - Client sends message via REST API
   - Message stored in MongoDB with 'UNSENT' status
   - Message scheduled for processing

2. **Message Processing**
   - Scheduler picks up unsent messages
   - Messages published to RabbitMQ queue
   - Processor service consumes messages
   - Webhook delivery attempted with circuit breaker and rate limiting
   - Message status updated based on delivery result

3. **Error Handling**
   - Failed messages retried with exponential backoff
   - Messages exceeding retry limit moved to DLQ
   - Stale messages detected and recovered
   - Circuit breaker prevents cascade failures
   - Rate limiting ensures system stability

## Monitoring and Maintenance

- Use health check endpoints to monitor service status
- Monitor RabbitMQ queues for message buildup
- Check DLQ for failed messages
- Monitor Redis for rate limiting metrics
- Use logging for debugging and audit trails

## Best Practices

- Use environment variables for configuration
- Monitor service health endpoints
- Check logs for error patterns
- Regularly monitor DLQ
- Maintain proper rate limits
- Keep circuit breaker thresholds tuned
- Monitor message processing latency

