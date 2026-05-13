.PHONY: build run test clean build-frontend docker-build local-test-up local-test-down local-test-logs local-test-ps local-test-config local-test-clean

# Go 参数
BINARY_NAME=spf-server
GO=go
GOFLAGS=-ldflags="-s -w"
DOCKER_COMPOSE ?= docker compose
LOCAL_TEST_COMPOSE ?= docker-compose.local-test.yml
LOCAL_TEST_PROJECT ?= spf-local-test

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

# 本地 Docker 测试环境
local-test-up:
	$(DOCKER_COMPOSE) -f $(LOCAL_TEST_COMPOSE) -p $(LOCAL_TEST_PROJECT) up -d

local-test-down:
	$(DOCKER_COMPOSE) -f $(LOCAL_TEST_COMPOSE) -p $(LOCAL_TEST_PROJECT) down --remove-orphans

local-test-logs:
	$(DOCKER_COMPOSE) -f $(LOCAL_TEST_COMPOSE) -p $(LOCAL_TEST_PROJECT) logs -f

local-test-ps:
	$(DOCKER_COMPOSE) -f $(LOCAL_TEST_COMPOSE) -p $(LOCAL_TEST_PROJECT) ps

local-test-config:
	$(DOCKER_COMPOSE) -f $(LOCAL_TEST_COMPOSE) -p $(LOCAL_TEST_PROJECT) config

local-test-clean:
	$(DOCKER_COMPOSE) -f $(LOCAL_TEST_COMPOSE) -p $(LOCAL_TEST_PROJECT) down -v --remove-orphans

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
	@echo "  local-test-up      - Start local Docker test stack"
	@echo "  local-test-down    - Stop local Docker test stack (keep volumes)"
	@echo "  local-test-logs    - Follow local Docker test stack logs"
	@echo "  local-test-ps      - Show local Docker test stack status"
	@echo "  local-test-config  - Render local Docker test stack compose config"
	@echo "  local-test-clean   - Stop local Docker test stack and remove volumes"
	@echo "  lint-frontend      - Run frontend lint and format check"
	@echo "  lint-backend       - Run backend lint"
	@echo "  test-sqlite-integration - Run SQLite integration tests"
	@echo "  test-mysql-integration  - Run MySQL integration tests"
	@echo "  ci                 - Run all CI checks"
