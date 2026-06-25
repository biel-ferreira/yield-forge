# YieldForge — developer task runner.
# Requires GNU Make + a POSIX shell. On Windows, use Git Bash or WSL, or run the
# underlying `go` / `docker` commands directly.

.DEFAULT_GOAL := help
BINARY := yield-forge

.PHONY: help run ingest build test lint docker-up

help: ## Show available targets
	@echo "YieldForge - make targets:"
	@echo "  run        Run the API locally (go run ./cmd/api)"
	@echo "  ingest     Run one market-data ingestion pass (go run ./cmd/ingest)"
	@echo "  build      Build the API binary into bin/"
	@echo "  test       Run all tests"
	@echo "  lint       Run go vet (+ golangci-lint if installed)"
	@echo "  docker-up  Build and start via docker compose"

run: ## Run the API locally
	go run ./cmd/api

ingest: ## Run one market-data ingestion pass (SPEC-006)
	go run ./cmd/ingest

build: ## Build the API binary
	go build -o bin/$(BINARY) ./cmd/api

test: ## Run all tests
	go test ./...

lint: ## Static analysis (go vet, then golangci-lint if available)
	go vet ./...
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed; ran go vet only"

docker-up: ## Build & run via docker compose
	docker compose -f deploy/docker-compose.yml up --build
