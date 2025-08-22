.PHONY: help build up down logs clean test docker-build docker-push

# Default target
help:
	@echo "Available commands:"
	@echo "  build       - Build the Docker image"
	@echo "  up          - Start all services in development mode"
	@echo "  down        - Stop all services"
	@echo "  logs        - Show logs from all services"
	@echo "  clean       - Remove all containers, volumes, and images"
	@echo "  test        - Run tests in Docker container"
	@echo "  prod-up     - Start services in production mode"
	@echo "  prod-down   - Stop production services"
	@echo "  docker-build - Build Docker image with tag"
	@echo "  docker-push  - Push Docker image to registry"

# Development commands
build:
	docker-compose build

up:
	docker-compose up -d

down:
	docker-compose down

logs:
	docker-compose logs -f

restart:
	docker-compose restart

# Production commands
prod-up:
	docker-compose -f docker-compose.prod.yml up -d

prod-down:
	docker-compose -f docker-compose.prod.yml down

prod-logs:
	docker-compose -f docker-compose.prod.yml logs -f

# Maintenance commands
clean:
	docker-compose down -v --rmi all --remove-orphans
	docker system prune -f

clean-volumes:
	docker-compose down -v

# Testing
test:
	docker-compose exec bankapi go test ./...

test-integration:
	docker-compose exec bankapi go test ./test/integration/...

# Database operations
db-migrate:
	docker-compose exec postgres psql -U bankuser -d bankapi -f /docker-entrypoint-initdb.d/001_create_users_table.up.sql
	docker-compose exec postgres psql -U bankuser -d bankapi -f /docker-entrypoint-initdb.d/002_create_accounts_table.up.sql
	docker-compose exec postgres psql -U bankuser -d bankapi -f /docker-entrypoint-initdb.d/003_create_transfers_table.up.sql

db-shell:
	docker-compose exec postgres psql -U bankuser -d bankapi

redis-shell:
	docker-compose exec redis redis-cli -a redispass123

# Docker registry operations (customize REGISTRY and IMAGE_NAME as needed)
REGISTRY ?= your-registry.com
IMAGE_NAME ?= bankapi
TAG ?= latest

docker-build:
	docker build -t $(REGISTRY)/$(IMAGE_NAME):$(TAG) .

docker-push: docker-build
	docker push $(REGISTRY)/$(IMAGE_NAME):$(TAG)

# Health checks
health:
	@echo "Checking service health..."
	@curl -f http://localhost:8080/api/v1/health || echo "API health check failed"

# Development helpers
dev-setup:
	@echo "Setting up development environment..."
	@cp .env.example .env
	@echo "Please edit .env file with your configuration"

dev-reset: down clean dev-setup up
	@echo "Development environment reset complete"