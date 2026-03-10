.PHONY: dev dev-api db migrate-up test setup

# Development
dev-api:
	go run ./cmd/api/

dev: dev-api

# Database
db:
	docker compose up -d postgres

migrate-up:
	@if [ -d cmd/migrate ]; then go run ./cmd/migrate/; else echo "No migrations yet"; fi

# Testing
test:
	go test -p 1 ./...
	cd web && npm test

# Setup
setup: db migrate-up
	@echo "Setup complete"
