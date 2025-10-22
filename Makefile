# Joint Analysis - Expense Tracker Makefile

.PHONY: help build up down restart rebuild-backend restart-backend logs logs-backend clean generate-db

# Default target
help:
	@echo "Available commands:"
	@echo "  build            - Build all Docker containers"
	@echo "  up               - Start all services with docker-compose"
	@echo "  down             - Stop all services"
	@echo "  restart          - Restart all services"
	@echo "  rebuild-backend  - Rebuild only the backend container"
	@echo "  rebuild-frontend - Rebuild only the frontend container"
	@echo "  restart-backend  - Restart only the backend service"
	@echo "  restart-frontend - Restart only the frontend service"
	@echo "  logs             - Show logs for all services"
	@echo "  logs-backend     - Show logs for backend service only"
	@echo "  generate-db      - Generate database code using sqlc"
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

# Rebuild only the frontend container
rebuild-frontend:
	docker-compose build frontend

# Restart only the backend service
restart-backend:
	docker-compose up -d backend

# Restart only the frontend service
restart-frontend:
	docker-compose up -d frontend

# Rebuild and restart backend (useful after code changes)
reload-backend: rebuild-backend restart-backend
	@echo "Backend reloaded successfully!"

# Rebuild and restart frontend (useful after dependency changes)
reload-frontend: rebuild-frontend restart-frontend
	@echo "Frontend reloaded successfully!"

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

# Generate database code using sqlc
generate-db:
	pushd backend/db && sqlc generate && popd
	@echo "Database code generated successfully!"