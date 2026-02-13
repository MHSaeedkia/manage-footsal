.PHONY: help build up down restart logs clean test

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build Docker images
	docker-compose build

up: ## Start all services
	docker-compose up -d
	@echo "Services started! Check logs with: make logs"

down: ## Stop all services
	docker-compose down

restart: ## Restart all services
	docker-compose restart

logs: ## Show logs from all services
	docker-compose logs -f

logs-bot: ## Show logs from bot service only
	docker-compose logs -f bot

logs-db: ## Show logs from PostgreSQL service only
	docker-compose logs -f postgres

clean: ## Stop and remove all containers, networks, and volumes
	docker-compose down -v
	@echo "All data has been removed!"

ps: ## Show status of services
	docker-compose ps

shell-bot: ## Open shell in bot container
	docker-compose exec bot sh

shell-db: ## Open PostgreSQL shell
	docker-compose exec postgres psql -U futsalbot -d futsalbot

test: ## Run tests
	go test -v ./...

fmt: ## Format Go code
	go fmt ./...

lint: ## Run linter
	golangci-lint run

setup: ## Setup .env file from example
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo ".env file created. Please edit it with your configuration."; \
	else \
		echo ".env file already exists."; \
	fi
