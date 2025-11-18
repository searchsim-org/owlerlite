.PHONY: help setup proto tidy build up down clean

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

setup: ## Complete setup: create env files, generate proto, tidy dependencies
	@echo "Setting up OwlerLite..."
	@echo "==> Creating .env.example for lightrag service..."
	@mkdir -p services/lightrag
	@if [ ! -f services/lightrag/.env.example ]; then \
		echo "# LightRAG Configuration" > services/lightrag/.env.example; \
		echo "# Configure your LLM provider (e.g., OpenAI, Anthropic, or local models)" >> services/lightrag/.env.example; \
		echo "LLM_BINDING_HOST=https://api.openai.com/v1" >> services/lightrag/.env.example; \
		echo "LLM_BINDING_API_KEY=your-api-key-here" >> services/lightrag/.env.example; \
		echo "LLM_MODEL=gpt-4o-mini" >> services/lightrag/.env.example; \
		echo "" >> services/lightrag/.env.example; \
		echo "# Configure your embedding model provider" >> services/lightrag/.env.example; \
		echo "EMBEDDING_BINDING_HOST=https://api.openai.com/v1" >> services/lightrag/.env.example; \
		echo "EMBEDDING_BINDING_API_KEY=your-api-key-here" >> services/lightrag/.env.example; \
		echo "EMBEDDING_MODEL=text-embedding-3-small" >> services/lightrag/.env.example; \
		echo "EMBEDDING_DIM=1536" >> services/lightrag/.env.example; \
		echo "Created services/lightrag/.env.example"; \
	else \
		echo "services/lightrag/.env.example already exists"; \
	fi
	@$(MAKE) proto
	@$(MAKE) tidy
	@echo "==> Setup complete! You can now run 'make build' to build Docker images."

proto: ## Generate protobuf code
	@echo "==> Generating protobuf code..."
	@if ! command -v protoc-gen-go >/dev/null 2>&1; then \
		echo "Installing protoc-gen-go..."; \
		go install google.golang.org/protobuf/cmd/protoc-gen-go@latest; \
	fi
	@if ! command -v protoc-gen-go-grpc >/dev/null 2>&1; then \
		echo "Installing protoc-gen-go-grpc..."; \
		go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest; \
	fi
	@export PATH="$$PATH:$$(go env GOPATH)/bin" && \
		cd services/frontier/proto && \
		protoc --go_out=. --go_opt=paths=source_relative \
		       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
		       urlfrontier.proto
	@if [ ! -f services/frontier/proto/go.mod ]; then \
		echo "module owlite/frontier/proto" > services/frontier/proto/go.mod; \
		echo "" >> services/frontier/proto/go.mod; \
		echo "go 1.22" >> services/frontier/proto/go.mod; \
		echo "" >> services/frontier/proto/go.mod; \
		echo "require (" >> services/frontier/proto/go.mod; \
		echo "	google.golang.org/grpc v1.66.0" >> services/frontier/proto/go.mod; \
		echo "	google.golang.org/protobuf v1.34.2" >> services/frontier/proto/go.mod; \
		echo ")" >> services/frontier/proto/go.mod; \
		cd services/frontier/proto && go mod tidy; \
	fi
	@echo "Protobuf code generated successfully"

tidy: ## Run go mod tidy for all Go services
	@echo "==> Running go mod tidy for all services..."
	@cd services/orchestrator && go mod tidy && echo "✓ orchestrator"
	@cd services/frontier && go mod tidy && echo "✓ frontier"
	@cd services/crawler && go mod tidy && echo "✓ crawler"
	@echo "Dependencies tidied successfully"

build: ## Build all Docker images
	@echo "==> Building Docker images..."
	@docker compose build

up: ## Start all services
	@echo "==> Starting services..."
	@docker compose up -d

down: ## Stop all services
	@echo "==> Stopping services..."
	@docker compose down

logs: ## Show logs from all services
	@docker compose logs -f

clean: ## Clean up generated files and Docker volumes
	@echo "==> Cleaning up..."
	@docker compose down -v
	@find services -name "go.sum" -delete
	@rm -f services/frontier/proto/*.pb.go
	@rm -f services/frontier/proto/go.mod services/frontier/proto/go.sum
	@echo "Cleanup complete"

rebuild: clean setup build ## Clean, setup, and rebuild everything

test-orchestrator: ## Test orchestrator service
	@cd services/orchestrator && go test -v ./...

test-frontier: ## Test frontier service
	@cd services/frontier && go test -v ./...

test-crawler: ## Test crawler service
	@cd services/crawler && go test -v ./...

test: test-orchestrator test-frontier test-crawler ## Run all tests

