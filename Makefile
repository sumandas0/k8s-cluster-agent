# Variables
BINARY_NAME=k8s-cluster-agent
DOCKER_IMAGE=k8s-cluster-agent
VERSION?=latest
GO=go
GOFLAGS=-v

# Build the binary
.PHONY: build
build:
	$(GO) build $(GOFLAGS) -o bin/$(BINARY_NAME) ./cmd/agent

# Run tests
.PHONY: test
test:
	$(GO) test -v -race -coverprofile=coverage.out ./...

# Run integration tests
.PHONY: test-integration
test-integration:
	$(GO) test -v -tags=integration ./test/integration/...

# Run linter
.PHONY: lint
lint:
	golangci-lint run

# Build Docker image
.PHONY: docker-build
docker-build:
	docker build -f build/docker/Dockerfile -t $(DOCKER_IMAGE):$(VERSION) .

# Push Docker image
.PHONY: docker-push
docker-push:
	docker push $(DOCKER_IMAGE):$(VERSION)

# Deploy to development
.PHONY: deploy-dev
deploy-dev:
	kubectl apply -k deployments/overlays/development

# Deploy to production
.PHONY: deploy-prod
deploy-prod:
	kubectl apply -k deployments/overlays/production

# Generate mocks
.PHONY: generate-mocks
generate-mocks:
	mockgen -source=internal/core/interfaces.go -destination=internal/core/mocks/mock_services.go

# Generate OpenAPI spec
.PHONY: generate-openapi
generate-openapi:
	@echo "Generating OpenAPI documentation..."
	@swag init -g cmd/agent/main.go -o docs --parseInternal --parseDependency

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf bin/
	rm -f coverage.out

# Run the application locally
.PHONY: run
run: build
	./bin/$(BINARY_NAME)

# Install dependencies
.PHONY: deps
deps:
	$(GO) mod download
	$(GO) install github.com/golang/mock/mockgen@latest
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install github.com/swaggo/swag/cmd/swag@latest

# Tidy dependencies
.PHONY: tidy
tidy:
	$(GO) mod tidy

# View coverage report
.PHONY: coverage
coverage: test
	$(GO) tool cover -html=coverage.out

# Run all checks before committing
.PHONY: check
check: tidy lint test

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build            - Build the binary"
	@echo "  test             - Run unit tests"
	@echo "  test-integration - Run integration tests"
	@echo "  lint             - Run linter"
	@echo "  docker-build     - Build Docker image"
	@echo "  docker-push      - Push Docker image"
	@echo "  deploy-dev       - Deploy to development"
	@echo "  deploy-prod      - Deploy to production"
	@echo "  generate-mocks   - Generate mocks"
	@echo "  generate-openapi - Generate OpenAPI spec"
	@echo "  clean            - Clean build artifacts"
	@echo "  run              - Run the application locally"
	@echo "  deps             - Install dependencies"
	@echo "  tidy             - Tidy dependencies"
	@echo "  coverage         - View coverage report"
	@echo "  check            - Run all checks"
	@echo "  help             - Show this help"

.DEFAULT_GOAL := help 