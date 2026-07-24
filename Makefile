.PHONY: help run dev build test test-cover test-race tidy fmt vet lint migrate-up migrate-down migrate-create tools

# Load DB vars from .env so migrate targets get the connection string.
# Leading "-" so a fresh clone without .env can still run make help/test.
-include .env
export

MIGRATE_DSN = postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)
MIGRATIONS_DIR = migrations

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-16s %s\n", $$1, $$2}'

run: ## Run the API
	go run ./cmd/api

dev: ## Run with hot reload (requires: make tools)
	air

build: ## Build the binary into ./bin
	go build -o bin/api ./cmd/api

test: ## Run all tests
	go test ./... -count=1

test-race: ## Run all tests with the race detector
	go test ./... -race -count=1

test-cover: ## Run tests and open an HTML coverage report
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out | tail -1
	go tool cover -html=coverage.out -o coverage.html
	@echo "report written to coverage.html"

fmt: ## Format all Go code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

lint: fmt vet ## Format, vet, and verify the build
	go build ./...

tidy: ## Sync go.mod / go.sum
	go mod tidy

tools: ## Install dev tools (air)
	go install github.com/air-verse/air@latest

migrate-up: ## Apply all up migrations via CLI (app also auto-migrates on startup)
	migrate -path $(MIGRATIONS_DIR) -database "$(MIGRATE_DSN)" up

migrate-down: ## Roll back the last migration
	migrate -path $(MIGRATIONS_DIR) -database "$(MIGRATE_DSN)" down 1

migrate-create: ## Create a new migration: make migrate-create name=add_orders
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)
