.PHONY: build run test clean build-frontend docker-build

# Go 参数
BINARY_NAME=spf-server
GO=go
GOFLAGS=-ldflags="-s -w"

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

# 帮助
help:
	@echo "Available targets:"
	@echo "  build           - Build frontend and backend"
	@echo "  build-frontend  - Build frontend only"
	@echo "  build-backend   - Build backend only (frontend must be built)"
	@echo "  run             - Run the server"
	@echo "  test            - Run all tests"
	@echo "  vet             - Run go vet"
	@echo "  clean           - Clean build artifacts"
	@echo "  docker-build    - Build Docker image"
