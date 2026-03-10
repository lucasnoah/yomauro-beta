.PHONY: dev dev-api db migrate migrate-create sqlc build web-dev web-build test setup

# Development
dev-api:
	go run ./cmd/api/

dev: dev-api

# Build
build:
	go build -o bin/api ./cmd/api/

# Database
db:
	docker compose up -d postgres

migrate:
	@set -a; \
	 [ -f config/.env ] && . ./config/.env; \
	 [ -f .env ] && . ./.env; \
	 set +a; \
	 migrate -database "$$DATABASE_URL" -path migrations/ up

migrate-create:
	@read -p "Migration name: " name; \
	last=$$(ls migrations/*.up.sql 2>/dev/null | sed 's|.*/||' | cut -d_ -f1 | sort -n | tail -1); \
	last_dec=$$(echo "$${last:-0}" | awk '{printf "%d\n", $$1}'); \
	seq=$$(printf "%06d" $$(( last_dec + 1 ))); \
	touch "migrations/$${seq}_$${name}.up.sql"; \
	echo "Created migrations/$${seq}_$${name}.up.sql"

# Code generation
sqlc:
	sqlc generate

# Frontend
web-dev:
	cd web && npm run dev

web-build:
	cd web && npm run build

# Testing
test:
	go test -p 1 ./... -timeout 60s
	cd web && npm test

# Setup
setup: db migrate
	@echo "Setup complete"
