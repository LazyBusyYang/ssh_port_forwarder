# SSH Port Forwarder (SPF)

SSH Port Forwarder 是一个基于 SSH 本地端口转发的负载均衡服务，用于将目标网络内的 TCP 服务通过 SSH 隧道安全地暴露到公网或可控网络环境中。

## 功能特性

- **SSH 端口转发**：通过 SSH 隧道将内网服务转发到本地端口
- **负载均衡**：支持 RoundRobin、LeastRules、Weighted 三种策略，在多个 SSH Host 间自动故障切换
- **健康检查**：定期检测 SSH Host 连通性和端口转发可用性
- **Web 管理界面**：基于 Vue3 的现代化管理界面，支持实时状态监控
- **Prometheus 指标**：内置 `/metrics` 端点，支持 Prometheus 监控
- **审计日志**：记录所有配置变更和重要系统事件
- **多数据库支持**：支持 SQLite（单机/开发）和 MySQL（生产/多实例）
- **JWT 认证**：支持 Token 刷新和 Secret 轮转
- **AES-256-GCM 加密**：SSH 凭证安全存储

## 架构概览

```
┌─────────────────────────────────────────────────────────────────┐
│                         SPF Service                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌─────────────────┐  │
│  │  Web UI  │  │  REST API│  │Prometheus│  │ SSH Connection  │  │
│  │ (Vue/Go) │  │  (Gin)   │  │ /metrics │  │    Manager      │  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────────┬────────┘  │
│       └──────────────┴─────────────┴─────────────────┘          │
│                          │                                       │
│              ┌───────────┴───────────┐                          │
│              │   Core Engine (Go)    │                          │
│              │  ┌─────────────────┐  │                          │
│              │  │  Scheduler      │  │                          │
│              │  │  Health Checker │  │                          │
│              │  │  LB Pool        │  │                          │
│              │  └─────────────────┘  │                          │
│              └───────────┬───────────┘                          │
│                          │                                       │
│              ┌───────────┴───────────┐                          │
│              │  SQLite / MySQL       │                          │
│              └───────────────────────┘                          │
└─────────────────────────────────────────────────────────────────┘
                              │
          ┌───────────────────┼───────────────────┐
          │                   │                   │
   ┌──────┴───────┐    ┌──────┴───────┐    ┌──────┴───────┐
   │  SSH Host A  │    │  SSH Host B  │    │  SSH Host C  │
   └──────┬───────┘    └──────┬───────┘    └──────┬───────┘
          │                   │                   │
          └───────────────────┼───────────────────┘
                              │
                    ┌─────────┴─────────┐
                    │   Target Services │
                    │  MySQL/Redis/HTTP │
                    └───────────────────┘
```

## 技术栈

- **后端**: Go 1.22+, Gin, GORM, Viper
- **前端**: Vue3, Vite, TypeScript
- **数据库**: SQLite3 / MySQL 8.0+
- **SSH**: golang.org/x/crypto/ssh
- **监控**: Prometheus Client

## 环境配置

### 系统要求

- Go 1.22 或更高版本
- Node.js 18+ 和 npm（构建前端）
- SQLite3 或 MySQL 8.0+
- Linux/macOS/Windows

### 依赖安装

1. **克隆项目**

```bash
git clone <repository-url>
cd ssh-port-forwarder
```

2. **安装 Go 依赖**

```bash
go mod download
```

3. **安装前端依赖**

```bash
cd web
npm install
cd ..
```

## 配置说明

### 配置文件

复制示例配置文件并修改：

```bash
cp config/config.yaml.example config/config.yaml
```

### 配置项说明

```yaml
server:
  host: 0.0.0.0          # 服务监听地址
  port: 8080             # 服务监听端口
  env: production        # 运行环境: development/production

database:
  type: sqlite           # 数据库类型: sqlite / mysql
  sqlite:
    path: ./data/spf.db  # SQLite 数据库文件路径
  mysql:
    dsn: root:password@tcp(host:port)/db?charset=utf8mb4

jwt:
  secret_current: your_jwt_secret_key     # 当前 JWT 密钥
  secret_previous: ""                     # 上一个 JWT 密钥（用于轮转）
  token_expire: 86400                     # Token 有效期（秒）
  refresh_expire: 604800                  # Refresh Token 有效期（秒）

encryption:
  key: your_32_byte_base64_encoded_key    # AES-256-GCM 加密密钥
  key_previous: ""                        # 上一个加密密钥（用于轮转）

port_range:
  min: 30000             # 转发端口范围最小值
  max: 33000             # 转发端口范围最大值
```

### 环境变量

以下配置项也可以通过环境变量设置（优先级高于配置文件）：

| 环境变量 | 说明 | 示例 |
|---------|------|------|
| `SPF_SERVER_HOST` | 服务监听地址 | `0.0.0.0` |
| `SPF_SERVER_PORT` | 服务监听端口 | `8080` |
| `SPF_DB_TYPE` | 数据库类型 | `mysql` |
| `SPF_DB_DSN` | MySQL DSN | `user:pass@tcp(host:3306)/db` |
| `SPF_JWT_SECRET_CURRENT` | JWT 当前密钥 | `your-secret-key` |
| `SPF_JWT_SECRET_PREVIOUS` | JWT 上一个密钥 | `old-secret-key` |
| `SPF_ENCRYPTION_KEY` | AES 加密密钥（Base64） | `base64-encoded-32-byte-key` |
| `SPF_ENCRYPTION_KEY_PREVIOUS` | AES 上一个密钥 | `old-base64-key` |
| `SPF_PORT_RANGE_MIN` | 端口范围最小值 | `30000` |
| `SPF_PORT_RANGE_MAX` | 端口范围最大值 | `33000` |

### 生成密钥

**生成 32 字节 Base64 编码的加密密钥：**

```bash
# Linux/macOS
openssl rand -base64 32

# 或使用 Go
go run -e 'package main; import ("crypto/rand"; "encoding/base64"; "fmt"); func main() { b := make([]byte, 32); rand.Read(b); fmt.Println(base64.StdEncoding.EncodeToString(b)) }'
```

## 启动项目

### 开发模式

1. **启动前端开发服务器**

```bash
cd web
npm run dev
```

前端将运行在 http://localhost:5173

2. **启动后端服务**

```bash
go run ./cmd/server/ -config config/config.yaml
```

或使用 Makefile：

```bash
make run
```

后端 API 将运行在 http://localhost:8080

### 生产模式

1. **构建完整项目**

```bash
make build
```

这将构建前端并嵌入到后端二进制中，生成 `spf-server` 可执行文件。

2. **运行服务**

```bash
./spf-server -config config/config.yaml
```

### 命令行参数

```bash
./spf-server -h

Usage of ./spf-server:
  -config string
        配置文件路径 (default "config/config.yaml")
  -version
        显示版本信息
```

### 初始化管理员账号

首次启动后，系统会自动创建一个默认管理员账号：

- 用户名: `admin`
- 密码: `admin`

**生产环境请务必修改默认密码！**

## 容器化部署

### 构建 Docker 镜像

```bash
make docker-build
```

或手动构建：

```bash
docker build -t ssh-port-forwarder:latest .
```

### 运行容器

**SQLite 模式（单机测试）：**

```bash
docker run -d \
  --name spf-server \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -e SPF_JWT_SECRET_CURRENT="your-jwt-secret" \
  -e SPF_ENCRYPTION_KEY="your-base64-key" \
  ssh-port-forwarder:latest
```

**MySQL 模式（生产环境）：**

```bash
docker run -d \
  --name spf-server \
  -p 8080:8080 \
  -e SPF_DB_TYPE=mysql \
  -e SPF_DB_DSN="user:password@tcp(mysql-host:3306)/spf_db?charset=utf8mb4" \
  -e SPF_JWT_SECRET_CURRENT="your-jwt-secret" \
  -e SPF_ENCRYPTION_KEY="your-base64-key" \
  -e SPF_PORT_RANGE_MIN=30000 \
  -e SPF_PORT_RANGE_MAX=33000 \
  ssh-port-forwarder:latest
```

### Docker Compose 部署

创建 `docker-compose.yml`：

```yaml
version: '3.8'

services:
  spf-server:
    image: ssh-port-forwarder:latest
    container_name: spf-server
    ports:
      - "8080:8080"
      # 转发端口范围（根据实际需求调整）
      - "30000-33000:30000-33000"
    environment:
      - SPF_DB_TYPE=mysql
      - SPF_DB_DSN=root:password@tcp(mysql:3306)/spf?charset=utf8mb4
      - SPF_JWT_SECRET_CURRENT=${JWT_SECRET}
      - SPF_ENCRYPTION_KEY=${ENCRYPTION_KEY}
      - SPF_PORT_RANGE_MIN=30000
      - SPF_PORT_RANGE_MAX=33000
    volumes:
      - ./data:/app/data
    depends_on:
      - mysql
    restart: unless-stopped

  mysql:
    image: mysql:8.0
    container_name: spf-mysql
    environment:
      - MYSQL_ROOT_PASSWORD=password
      - MYSQL_DATABASE=spf
    volumes:
      - mysql_data:/var/lib/mysql
    restart: unless-stopped

volumes:
  mysql_data:
```

启动服务：

```bash
# 设置环境变量
export JWT_SECRET="your-jwt-secret"
export ENCRYPTION_KEY="your-base64-key"

# 启动
docker-compose up -d
```

### Kubernetes 部署

创建 `deployment.yaml`：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: spf-server
  labels:
    app: spf-server
spec:
  replicas: 1
  selector:
    matchLabels:
      app: spf-server
  template:
    metadata:
      labels:
        app: spf-server
    spec:
      containers:
      - name: spf-server
        image: ssh-port-forwarder:latest
        ports:
        - containerPort: 8080
        env:
        - name: SPF_DB_TYPE
          value: "mysql"
        - name: SPF_DB_DSN
          valueFrom:
            secretKeyRef:
              name: spf-secrets
              key: db-dsn
        - name: SPF_JWT_SECRET_CURRENT
          valueFrom:
            secretKeyRef:
              name: spf-secrets
              key: jwt-secret
        - name: SPF_ENCRYPTION_KEY
          valueFrom:
            secretKeyRef:
              name: spf-secrets
              key: encryption-key
        - name: SPF_PORT_RANGE_MIN
          value: "30000"
        - name: SPF_PORT_RANGE_MAX
          value: "33000"
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: spf-server
spec:
  selector:
    app: spf-server
  ports:
  - port: 8080
    targetPort: 8080
  type: ClusterIP
```

创建 Secret：

```bash
kubectl create secret generic spf-secrets \
  --from-literal=db-dsn="user:pass@tcp(mysql:3306)/spf?charset=utf8mb4" \
  --from-literal=jwt-secret="your-jwt-secret" \
  --from-literal=encryption-key="your-base64-key"
```

部署：

```bash
kubectl apply -f deployment.yaml
```

## API 文档

### REST API 基础信息

- **Base URL**: `/api/v1`
- **认证方式**: Bearer Token
- **Content-Type**: `application/json`

### 主要接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/v1/auth/login` | POST | 用户登录 |
| `/api/v1/auth/refresh` | POST | 刷新 Token |
| `/api/v1/hosts` | GET/POST | SSH Host 列表/创建 |
| `/api/v1/hosts/:id` | GET/PUT/DELETE | SSH Host 详情/更新/删除 |
| `/api/v1/groups` | GET/POST | 转发组列表/创建 |
| `/api/v1/groups/:id` | GET/PUT/DELETE | 转发组详情/更新/删除 |
| `/api/v1/rules` | GET/POST | 转发规则列表/创建 |
| `/api/v1/rules/:id` | GET/PUT/DELETE | 转发规则详情/更新/删除 |
| `/api/v1/status/overview` | GET | 系统状态概览 |
| `/api/v1/audit-logs` | GET | 审计日志 |
| `/metrics` | GET | Prometheus 指标 |

### WebSocket 接口

- **URL**: `ws://host/api/v1/ws/status`
- **功能**: 实时推送 Host 和 Rule 状态变更

## 监控指标

服务暴露以下 Prometheus 指标：

```
# SSH Host 健康状态
ssh_host_health{host="10.0.0.101"} 1

# 活跃 SSH 连接数
ssh_connection_active 5

# 端口转发在线状态
forward_active{local_port="12001", target="10.0.0.101:3306"} 1

# 健康检查延迟
ssh_healthcheck_latency_seconds{host="10.0.0.101"} 0.023

# LB 故障切换次数
lb_failover_total{from_host="10.0.0.101", to_host="10.0.0.102"} 3
```

## 安全注意事项

1. **修改默认密码**: 首次登录后务必修改默认管理员密码
2. **使用 HTTPS**: 生产环境请通过反向代理（Nginx/Traefik）启用 HTTPS
3. **密钥管理**: JWT Secret 和加密密钥应通过 K8S Secret 或 Vault 管理
4. **端口安全**: 合理配置防火墙规则，限制转发端口的访问来源
5. **定期轮转**: 定期轮换 JWT Secret 和加密密钥

## 开发指南

### 项目结构

```
ssh-port-forwarder/
├── cmd/server/           # 程序入口
├── internal/
│   ├── config/          # 配置管理
│   ├── model/           # 数据模型
│   ├── repository/      # 数据访问层
│   ├── service/         # 业务逻辑层
│   │   ├── ssh_manager/ # SSH 连接管理
│   │   ├── health/      # 健康检查
│   │   ├── lb/          # 负载均衡
│   │   └── scheduler/   # 调度器
│   ├── handler/         # HTTP 处理器
│   ├── middleware/      # Gin 中间件
│   └── pkg/             # 工具包
├── web/                 # Vue3 前端
├── config/              # 配置文件
├── Makefile
└── Dockerfile
```

### 运行测试

```bash
make test
```

### 代码检查

```bash
make vet
```

## 许可证

[MIT License](LICENSE)

## 贡献指南

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request
