# Joint Analysis - Expense Tracker Makefile

.PHONY: help build up down restart rebuild-backend restart-backend logs logs-backend clean

# Default target
help:
	@echo "Available commands:"
	@echo "  build            - Build all Docker containers"
	@echo "  up               - Start all services with docker-compose"
	@echo "  down             - Stop all services"
	@echo "  restart          - Restart all services"
	@echo "  rebuild-backend  - Rebuild only the backend container"
	@echo "  restart-backend  - Restart only the backend service"
	@echo "  logs             - Show logs for all services"
	@echo "  logs-backend     - Show logs for backend service only"
	@echo "  clean            - Remove all containers and volumes"

# Build all containers
build:
	docker-compose build

# Start all services
up:
	docker-compose up -d

# Start all services with build
up-build:
	docker-compose up --build -d

# Stop all services
down:
	docker-compose down

# Restart all services
restart:
	docker-compose restart

# Rebuild only the backend container
rebuild-backend:
	docker-compose build backend

# Restart only the backend service
restart-backend:
	docker-compose up -d backend

# Rebuild and restart backend (useful after code changes)
reload-backend: rebuild-backend restart-backend
	@echo "Backend reloaded successfully!"

# Show logs for all services
logs:
	docker-compose logs -f

# Show logs for backend service only
logs-backend:
	docker-compose logs -f backend

# Clean up - remove containers, networks, and volumes
clean:
	docker-compose down -v --remove-orphans
	docker system prune -f

# Development helpers
dev-setup: build up
	@echo "Development environment is ready!"
	@echo "Frontend: http://localhost:3001"
	@echo "Backend: http://localhost:8081"

# Quick backend development cycle
dev-backend: rebuild-backend restart-backend logs-backend