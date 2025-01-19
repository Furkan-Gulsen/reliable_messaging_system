.PHONY: build run clean test docker-build docker-run docker-stop help seed

BINARY_SENDER=sender_service
BINARY_PROCESSOR=processor_service
GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/bin

DOCKER_COMPOSE=docker-compose

help:
	@echo "Make commands:"
	@echo "build         - Build both services"
	@echo "run          - Run both services locally"
	@echo "clean        - Clean build files"
	@echo "test         - Run tests"
	@echo "docker-build - Build Docker images"
	@echo "docker-run   - Run services with Docker Compose"
	@echo "docker-stop  - Stop Docker Compose services"
	@echo "seed         - Seed MongoDB with test data"

build:
	@echo "Building services..."
	go build -o $(GOBIN)/$(BINARY_SENDER) ./sender_service
	go build -o $(GOBIN)/$(BINARY_PROCESSOR) ./processor_service

run: build
	@echo "Running services..."
	$(GOBIN)/$(BINARY_SENDER) & $(GOBIN)/$(BINARY_PROCESSOR)

clean:
	@echo "Cleaning build files..."
	rm -rf $(GOBIN)
	go clean
	docker-compose down -v

test:
	@echo "Running tests..."
	go test -v ./...

docker-build:
	@echo "Building Docker images..."
	$(DOCKER_COMPOSE) build

docker-run:
	@echo "Starting services with Docker Compose..."
	$(DOCKER_COMPOSE) up -d

docker-stop:
	@echo "Stopping services..."
	$(DOCKER_COMPOSE) down


logs:
	@echo "Showing service logs..."
	$(DOCKER_COMPOSE) logs -f

seed:
	@echo "Seeding MongoDB with test data..."
	chmod +x scripts/seed_messages.sh
	./scripts/seed_messages.sh

default: help 