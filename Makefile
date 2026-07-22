.PHONY: help run build test tidy migrate-up migrate-down migrate-create

# Load DB vars from .env so migrate targets get the connection string.
include .env
export

MIGRATE_DSN = postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)
MIGRATIONS_DIR = migrations

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-16s %s\n", $$1, $$2}'

run: ## Run the API
	go run ./cmd/api

build: ## Build the binary into ./bin
	go build -o bin/api ./cmd/api

test: ## Run all tests
	go test ./... -race -count=1

tidy: ## Sync go.mod / go.sum
	go mod tidy

migrate-up: ## Apply all up migrations via CLI (app also auto-migrates on startup)
	migrate -path $(MIGRATIONS_DIR) -database "$(MIGRATE_DSN)" up

migrate-down: ## Roll back the last migration
	migrate -path $(MIGRATIONS_DIR) -database "$(MIGRATE_DSN)" down 1

migrate-create: ## Create a new migration: make migrate-create name=add_orders
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)
