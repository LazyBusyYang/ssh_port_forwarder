.PHONY: build run test clean build-frontend docker-build compose-up compose-down compose-logs compose-ps compose-config

# Go 参数
BINARY_NAME=spf-server
GO=go
GOFLAGS=-ldflags="-s -w"
DOCKER_COMPOSE ?= docker compose

# 默认目标
all: build

# 构建前端
build-frontend:
	cd web && npm install && npm run build

# 构建后端（包含嵌入的前端）
build: build-frontend
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) ./cmd/server/

# 仅构建后端（假设前端已构建）
build-backend:
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) ./cmd/server/

# 运行
run:
	$(GO) run ./cmd/server/ -config config/config.yaml

# 测试
test:
	$(GO) test ./... -v -count=1

# Go vet
vet:
	$(GO) vet ./...

# 清理
clean:
	rm -f $(BINARY_NAME)
	rm -rf web/dist

# Docker 构建
docker-build:
	docker build -t ssh-port-forwarder:latest .

# 根目录 docker-compose.yml（需已复制 .env.example 为 .env 并填写密钥）
compose-up:
	$(DOCKER_COMPOSE) up -d

compose-down:
	$(DOCKER_COMPOSE) down --remove-orphans

compose-logs:
	$(DOCKER_COMPOSE) logs -f

compose-ps:
	$(DOCKER_COMPOSE) ps

compose-config:
	$(DOCKER_COMPOSE) config

# CI 相关目标
lint-frontend:
	cd web && npm run lint
	cd web && npm run format:check

lint-backend:
	gofmt -d .
	go vet ./...
	golangci-lint run

test-sqlite-integration:
	@bash scripts/test-sqlite.sh

test-mysql-integration:
	@bash scripts/test-mysql.sh

ci: lint-frontend lint-backend test test-sqlite-integration test-mysql-integration
	@echo "All CI checks passed!"

# 帮助
help:
	@echo "Available targets:"
	@echo "  build              - Build frontend and backend"
	@echo "  build-frontend     - Build frontend only"
	@echo "  build-backend      - Build backend only (frontend must be built)"
	@echo "  run                - Run the server"
	@echo "  test               - Run all tests"
	@echo "  vet                - Run go vet"
	@echo "  clean              - Clean build artifacts"
	@echo "  docker-build       - Build Docker image"
	@echo "  compose-up         - Start docker-compose.yml stack"
	@echo "  compose-down       - Stop compose stack (keep volumes)"
	@echo "  compose-logs       - Follow compose logs"
	@echo "  compose-ps         - Show compose service status"
	@echo "  compose-config     - Validate/render compose config"
	@echo "  lint-frontend      - Run frontend lint and format check"
	@echo "  lint-backend       - Run backend lint"
	@echo "  test-sqlite-integration - Run SQLite integration tests"
	@echo "  test-mysql-integration  - Run MySQL integration tests"
	@echo "  ci                 - Run all CI checks"
