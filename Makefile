# Shortcuts for everyday Inkwell commands. Run `make help` to list them.

GOLANGCI_LINT_VERSION := v2.15.0
API_DIR := apps/api

.DEFAULT_GOAL := help
.PHONY: help creds up down ps logs test test-integration lint

help: ## List available commands
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  make %-16s %s\n", $$1, $$2}'

creds: ## Write 1-hour AWS credentials for the API container (role inkwell-dev-api)
	@umask 077 && aws configure export-credentials --profile inkwell-api --format env-no-export > .aws-session.env
	@echo "Wrote .aws-session.env (valid for 1 hour). Now run: make up"

up: ## Build and start the whole stack in the background
	docker compose up -d --build -V

down: ## Stop the stack (data volumes are kept)
	docker compose down

ps: ## Show container status
	docker compose ps -a

logs: ## Follow the API's logs (Ctrl+C to stop following)
	docker compose logs -f api

logs-web: ## Follow the web dev server's logs (Ctrl+C to stop following)
	docker compose logs -f web

test: ## Run the Go unit tests
	cd $(API_DIR) && go test ./...

test-integration: ## Run integration tests (MongoDB + HTTP isolation) against the local stack (run `make up` first)
	set -a && . ./.env && set +a && cd $(API_DIR) && go test -tags integration -count=1 ./...

lint: ## Run golangci-lint in Docker (no local install needed)
	docker run --rm -v "$(CURDIR)/$(API_DIR):/app" -w /app golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) golangci-lint run
FOUNDATION_DIR := infra/terraform/envs/prod/foundation
IMAGE_TAG ?= $(shell git rev-parse --short=12 HEAD)
ECR_REPO = $(shell terraform -chdir=$(FOUNDATION_DIR) output -raw ecr_repository_url)

push-api: ## Build the API for arm64 and push it to ECR, tagged with the git SHA (clean tree only)
	@git diff --quiet && git diff --cached --quiet || { echo "Commit your changes first: the image tag must match a commit."; exit 1; }
	eval "$$(aws configure export-credentials --profile inkwell --format env)" && docker buildx build --platform linux/arm64 --provenance=false -t $(ECR_REPO):$(IMAGE_TAG) --push $(API_DIR)
	@echo "Pushed $(ECR_REPO):$(IMAGE_TAG)"
