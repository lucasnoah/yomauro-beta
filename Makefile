.PHONY: dev dev-api db migrate-up test setup

# Development
dev-api:
	go run ./cmd/api/

dev: dev-api

# Database
db:
	docker compose up -d postgres

migrate-up:
	go run ./cmd/migrate/

# Testing
test:
	go test ./...
	cd web && npm test

# Setup
setup: db migrate-up
	@echo "Setup complete"
