# SSH Port Forwarder 本地测试环境使用指南

## 概述

本文档说明如何在本地启动 SSH Port Forwarder（SPF）完整测试环境，用于功能验收和开发调试。

**安全提示**：本文档涉及的密码、密钥、SSH 私钥均为**本地测试 fixture**，请勿用于生产环境或个人基础设施。

## 环境架构

```
┌─────────────────────────────────────────────────────┐
│              spf-local-dev (Docker Network)          │
│                                                      │
│  ┌──────────────┐   ┌──────────────┐              │
│  │    mysql     │   │   backend     │   :8080      │
│  │   :3306      │   │  (Go + Gin)  │──────────────┤─→ 宿主机 :8080
│  └──────────────┘   └──────────────┘              │
│                              │                       │
│                        ┌─────┴─────┐                │
│                        │  frontend │  Vite dev      │
│                        │  (Vue3)   │──────────────┤─→ 宿主机 :5173
│                        └───────────┘               │
│                                                      │
│  ┌────────────────┐  ┌────────────────┐           │
│  │  ssh-password  │  │    ssh-key     │           │
│  │   :2222        │  │    :2222       │           │
│  │ (密码认证)      │  │   (公钥认证)    │           │
│  └────────────────┘  └────────────────┘           │
└─────────────────────────────────────────────────────┘
                          │
              ┌───────────┴───────────┐
              │  宿主机端口映射         │
              │  MySQL  :3307         │
              │  Backend:8080         │
              │  Frontend:5173        │
              │  SSH密码 :2222        │
              │  SSH公钥 :2223        │
              └───────────────────────┘
```

## 快速启动

### 方式一：在宿主机运行后端（推荐开发调试用）

需要宿主机已安装 Go 1.22+、Node.js 18+ 和 MySQL 客户端。

#### 步骤 1：启动 MySQL 和 SSH fixture

```bash
docker compose -f spf-local-dev.yml --project-name spf-local-dev up -d mysql ssh-password ssh-key
```

#### 步骤 2：确认 MySQL 就绪

```bash
docker compose -f spf-local-dev.yml --project-name spf-local-dev ps
```

确认 `mysql` 状态为 `healthy` 后进行下一步。

#### 步骤 3：在宿主机启动后端

以下命令均在仓库根目录执行：

```bash
export SERVER_HOST=0.0.0.0
export SERVER_PORT=8080
export SERVER_ENV=development
export DATABASE_TYPE=mysql
export SPF_DB_DSN="root:spf_local_test_root@tcp(127.0.0.1:3307)/spf_local_test?charset=utf8mb4&parseTime=true&loc=Local"
export JWT_SECRET_CURRENT=local-test-jwt-secret-not-for-production
export SPF_ENCRYPTION_KEY=VO/LNju26T7/QmFljBsajoAgcXhNCk4IpNegJr+nKvs=
export SPF_DEFAULT_ADMIN_USER=admin
export SPF_DEFAULT_ADMIN_PASS=admin123

go run ./cmd/server/
```

#### 步骤 4：在新终端窗口启动前端

```bash
cd web
npm install   # 首次运行后保留 node_modules
npm run dev
```

### 方式二：全容器运行（源码挂载，热重载）

全部服务运行在 Docker 中，后端和前端代码通过 volume 挂载支持热重载。

```bash
docker compose -f spf-local-dev.yml --project-name spf-local-dev up -d
```

> 注意：方式二会占用宿主机端口 8080、5173、3307、2222、2223，请确保这些端口未被占用。

## 服务地址

| 组件 | 宿主机访问地址 | 容器内地址 |
|------|--------------|-----------|
| 前端（Vite dev server） | <http://localhost:5173> | `frontend:5173` |
| 后端 API | <http://localhost:8080> | `backend:8080` |
| MySQL | `127.0.0.1:3307` | `mysql:3306` |
| SSH 密码认证 fixture | `127.0.0.1:2222` | `ssh-password:2222` |
| SSH 公钥认证 fixture | `127.0.0.1:2223` | `ssh-key:2222` |

## 登录信息

- **应用地址**：<http://localhost:8080>（Backend 内嵌已构建前端，可直接使用全部功能）

- **Vite 开发服务器**：<http://localhost:5173>（前端源码热重载开发模式，需等待 npm install 完成后可用）
- **管理员用户名**：`admin`
- **管理员密码**：`admin123`

### SSH Fixture 使用方法

> **已知问题**：`lscr.io/linuxserver/openssh-server` 最新版本（10.2_p1-r0-ls225）中，`PASSWORD_ACCESS=true` 环境变量在某些情况下未能正确覆盖 sshd_config 中的 `PasswordAuthentication no`。如遇 SSH 密码连接失败，请改用公钥认证，或考虑替换为 `openssh-server:9.6_p1-r1-ls54` 等较早版本镜像。

### 密码认证 SSH

在 SPF 应用中创建 SSH Host 时填写：

- **Host**：`127.0.0.1`（SPF 运行在宿主机）或 `ssh-password`（SPF 运行在 Docker 同网络）
- **Port**：`2222`
- **用户名**：`testuser`
- **认证方式**：密码
- **密码**：`testpass123`

宿主机连通性验证：

```bash
ssh -p 2222 -o StrictHostKeyChecking=no testuser@127.0.0.1 true
# 提示密码时输入：testpass123
```

### 公钥认证 SSH

#### 解码私钥文件

```bash
openssl base64 -d \
  -in scripts/fixtures/spf-local-test-ed25519-openssh-inline.pem.b64 \
  -out /tmp/spf-local-test-key.pem
chmod 600 /tmp/spf-local-test-key.pem
```

#### 在 SPF 应用中创建 SSH Host

- **Host**：`127.0.0.1`（SPF 运行在宿主机）或 `ssh-key`（SPF 运行在 Docker 同网络）
- **Port**：`2222`（Docker 网络内）或 `2223`（从宿主机连接）
- **用户名**：`keyuser`
- **认证方式**：公钥
- **私钥内容**：将 `/tmp/spf-local-test-key.pem` 文件的全部内容粘贴到应用私钥输入框

宿主机连通性验证：

```bash
ssh -i /tmp/spf-local-test-key.pem -p 2223 \
  -o StrictHostKeyChecking=no keyuser@127.0.0.1 true
```

对应的公钥行（已在 SSH 服务端配置）：

```
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHdOtIbc4G8PIRvJ/4hdsyc+gVftBS+01nNw71Q66z5K spf-local-test-key-not-for-production
```

## 常用命令

### 查看服务状态

```bash
docker compose -f spf-local-dev.yml --project-name spf-local-dev ps
```

### 查看日志

```bash
# 全部服务日志
docker compose -f spf-local-dev.yml --project-name spf-local-dev logs -f

# 单个服务日志
docker compose -f spf-local-dev.yml --project-name spf-local-dev logs -f backend
docker compose -f spf-local-dev.yml --project-name spf-local-dev logs -f mysql
```

### 停止环境

```bash
# 停止（保留数据卷）
docker compose -f spf-local-dev.yml --project-name spf-local-dev down

# 停止并删除数据卷（清空数据）
docker compose -f spf-local-dev.yml --project-name spf-local-dev down -v
```

### 重启服务

```bash
docker compose -f spf-local-dev.yml --project-name spf-local-dev restart backend
```

### 重新构建（代码变更后）

```bash
docker compose -f spf-local-dev.yml --project-name spf-local-dev up -d --force-recreate backend frontend
```

## 环境变量说明

| 环境变量 | 值 | 说明 |
|---------|-----|------|
| `SERVER_HOST` | `0.0.0.0` | 后端监听地址 |
| `SERVER_PORT` | `8080` | 后端监听端口 |
| `SERVER_ENV` | `development` | 运行环境 |
| `DATABASE_TYPE` | `mysql` | 数据库类型 |
| `SPF_DB_DSN` | `root:spf_local_test_root@tcp(...)` | MySQL 连接字符串 |
| `JWT_SECRET_CURRENT` | `local-test-jwt-secret-not-for-production` | JWT 签名密钥 |
| `SPF_ENCRYPTION_KEY` | `VO/LNju26T7/QmFljBsajoAgcXhNCk4IpNegJr+nKvs=` | AES-256-GCM 加密密钥 |
| `SPF_DEFAULT_ADMIN_USER` | `admin` | 默认管理员用户名 |
| `SPF_DEFAULT_ADMIN_PASS` | `admin123` | 默认管理员密码 |

## 故障排除

### 端口占用

修改 `spf-local-dev.yml` 中宿主机侧端口映射，避免与其他服务冲突。

### Docker 内 SPF 无法连接 SSH fixture

在容器网络内，Host 填写服务名 `ssh-password` 或 `ssh-key`，端口填 `2222`。只有在 SPF 运行在宿主机时才填 `127.0.0.1`。

### 私钥权限错误

使用私钥文件前必须执行：

```bash
chmod 600 /tmp/spf-local-test-key.pem
```

### 前端 API 请求失败

确认 `backend` 和 `frontend` 均已启动。前端使用 `network_mode: service:backend`，Vite 将 `/api` 请求代理到容器内的 `localhost:8080`。

### MySQL 连接失败

确认 MySQL 容器状态为 `healthy`：

```bash
docker compose -f spf-local-dev.yml --project-name spf-local-dev ps mysql
```

### 镜像拉取失败

检查网络连接。可用镜像包括：`mysql:8.0`、`golang:1.25-alpine`、`node:22-alpine`、`lscr.io/linuxserver/openssh-server:latest`。

### 清空所有数据

```bash
docker compose -f spf-local-dev.yml --project-name spf-local-dev down -v
```

## 与正式部署的关系

- **正式 Docker Compose**：根目录 `docker-compose.yml`，提供 MySQL + 发布镜像，适合生产级部署。
- **本地测试环境**：`spf-local-dev.yml`，提供源码级前后端 + MySQL + SSH fixture，适合开发调试和功能验收。

其他部署方式（Kubernetes、局域网访问、CI/Release 说明）见 [DEPLOYMENT.md](./DEPLOYMENT.md)。
