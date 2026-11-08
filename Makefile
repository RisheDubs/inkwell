# Shortcuts for everyday Inkwell commands. Run `make help` to list them.

GOLANGCI_LINT_VERSION := v2.15.0
API_DIR := apps/api

.DEFAULT_GOAL := help
.PHONY: help up down ps logs test lint

help: ## List available commands
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  make %-6s %s\n", $$1, $$2}'

up: ## Build and start the whole stack in the background
	docker compose up -d --build

down: ## Stop the stack (data volumes are kept)
	docker compose down

ps: ## Show container status
	docker compose ps -a

logs: ## Follow the API's logs (Ctrl+C to stop following)
	docker compose logs -f api

test: ## Run the Go unit tests
	cd $(API_DIR) && go test ./...

lint: ## Run golangci-lint in Docker (no local install needed)
	docker run --rm -v "$(CURDIR)/$(API_DIR):/app" -w /app golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) golangci-lint run
