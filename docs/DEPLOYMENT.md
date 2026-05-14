# 部署说明

本文说明 **SSH Port Forwarder（SPF）** 支持的启动方式、默认镜像与网络访问方式。生产环境请务必替换默认口令与密钥，并配合 HTTPS（Ingress / 反向代理）与最小权限网络策略。

## 支持的启动形态

| 方式 | 适用场景 | 说明 |
|------|----------|------|
| 本地源码 + SQLite | 日常开发、单机试用 | 无需 Docker；见下文「本地源码 + SQLite」 |
| Docker Compose | 团队演示、一体化本地栈 | 仓库根目录 [`docker-compose.yml`](../docker-compose.yml) |
| Kubernetes | 集群部署 | 仓库 [`deploy/kubernetes/`](../deploy/kubernetes/) 最小示例 |

**前后端如何访问**：发布镜像由单一进程 `spf-server` 在 **同一端口**（默认 `8080`）提供 **嵌入的静态前端** 与 **REST / WebSocket API**（前端使用相对路径 `/api/v1`）。不存在「官方 Compose 里再拆一个前端容器」的路径；局域网访问使用 `http://<宿主机 IP>:8080` 即可。

## 公开镜像

默认使用 Docker Hub 命名空间 **`dockersenseyang/ssh_port_forwarder`**。文档与 Compose 中出现的 **`:latest`** 仅作本地/演示快捷引用；**CI 自动发布只推送与根目录 `VERSION` 一致的 semver tag**（如 `:0.2.0`），**不**更新 `latest`（`latest` 单独维护）。发版与手动 **`dev` / `sha-*`** 镜像流程见 [CI_RELEASE.md](./CI_RELEASE.md)。

本地构建时使用 `make docker-build` 或 `docker build -t ssh-port-forwarder:latest .`，与远程 semver tag 可并存；Compose 与 K8s 示例中请将镜像 tag 换成你们实际使用的版本号。

`Dockerfile` **默认不注入** HTTP(S) 通用代理；`GOPROXY` 默认为官方 `https://proxy.golang.org,direct`（不使用国内镜像等第三方 Go 代理）。若拉模块或 `apk`/`npm` 超时，可自行加 `--build-arg HTTP_PROXY=...`。

**经本机 HTTP 代理构建（示例：代理在宿主机 `127.0.0.1:7890`）**：构建容器内 `127.0.0.1` 指向容器自身，应使用 **`host.docker.internal`**（Docker Desktop macOS/Windows 常见）。Linux 可改为宿主机网桥地址（如 `172.17.0.1`）或查阅所用 Docker 发行版文档。

```bash
docker build \
  --build-arg HTTP_PROXY=http://host.docker.internal:7890 \
  --build-arg HTTPS_PROXY=http://host.docker.internal:7890 \
  --build-arg ALL_PROXY=http://host.docker.internal:7890 \
  -t ssh-port-forwarder:latest .
```

## 本地源码 + SQLite

1. 复制配置：`cp config/config.yaml.example config/config.yaml`，按需修改（默认 `database.type: sqlite`）。
2. 安装依赖：`go mod download`；前端 `cd web && npm install`。
3. 启动后端：`go run ./cmd/server/ -config config/config.yaml` 或 `make run`（监听 `config.yaml` 中 `server.host` / `server.port`，示例为 `0.0.0.0:8080`）。
4. 启动前端开发服务：`cd web && npm run dev`，默认 <http://localhost:5173>，Vite 将 `/api` 代理到 `http://localhost:8080`（见 `web/vite.config.ts`）。

**仅本机浏览器**：上述即可。

**同一局域网其他设备访问开发前端**：需让 Vite 监听 `0.0.0.0`（例如 `npm run dev -- --host 0.0.0.0`），且浏览器所在机器必须能访问到你电脑上的后端地址。此时 `vite.config.ts` 里代理目标仍为 `localhost:8080` 仅对 **运行 Vite 的那台机器** 有效；其他设备应直接访问后端 API（例如将前端构建为生产静态资源并由后端托管，或使用可配置 API 基址——当前仓库默认开发流以本机为主）。

## Docker Compose（MySQL + 应用）

1. `cp .env.example .env`，编辑 `.env`：`MYSQL_ROOT_PASSWORD` 与 `SPF_DB_DSN` 中的用户名、密码、库名保持一致；设置强随机 `JWT_SECRET_CURRENT` 与 `SPF_ENCRYPTION_KEY`（可用 `openssl rand -base64 32`）。
2. 启动：`docker compose up -d`（在项目根目录，默认读取 `docker-compose.yml` 与 `.env`）。
3. 浏览器访问：**本机** <http://localhost:8080>；**局域网** <http://\<本机局域网 IP\>:8080>（需防火墙放行 `8080`）。

Compose 同时将 **SSH 转发监听端口段** `30000-33000` 映射到宿主机，便于从外网或宿主机连接转发端口；该范围暴露面大，生产请按实际需求收窄并配置防火墙。

MySQL 对宿主机暴露 `3306` 便于本地客户端调试；若不需要，可从 `docker-compose.yml` 中删除 `mysql` 的 `ports` 段。

使用 MySQL 作生产库时的版本升级注意点见根目录 [README.md](../README.md) 中「版本升级（MySQL 生产）」。

`.env` / 容器内环境变量与 Viper 的对应关系见根目录 README「环境变量」表（如 **`DATABASE_TYPE`**、**`JWT_SECRET_CURRENT`**、**`SPF_DB_DSN`** 等）。

## Kubernetes

仓库内提供最小可应用示例（无 Ingress TLS 实现，生产请自行补充）：

- [`deploy/kubernetes/deployment.yaml`](../deploy/kubernetes/deployment.yaml)
- [`deploy/kubernetes/service.yaml`](../deploy/kubernetes/service.yaml)
- [`deploy/kubernetes/ingress.yaml.example`](../deploy/kubernetes/ingress.yaml.example)（可选，需集群已安装 Ingress Controller）

创建 Secret（键名需与 Deployment 中 `secretKeyRef` 一致）：

```bash
kubectl create secret generic spf-secrets \
  --from-literal=db-dsn='user:pass@tcp(mysql.default.svc.cluster.local:3306)/spf?charset=utf8mb4&parseTime=true&loc=Local' \
  --from-literal=jwt-secret='your-jwt-secret' \
  --from-literal=encryption-key='your-base64-key'
```

部署：

```bash
kubectl apply -f deploy/kubernetes/deployment.yaml -f deploy/kubernetes/service.yaml
```

按需应用 `ingress.yaml.example`（复制为正式清单并修改 host/TLS 后再 `kubectl apply`）。

## 相关文档

- **CI 发版与 Docker Hub**：见 [CI_RELEASE.md](./CI_RELEASE.md)。
- 在正式 Compose 之上叠加 **SSH fixture、源码级前后端** 等本地测试编排：见 [LOCAL_DOCKER_TEST_ENV.md](./LOCAL_DOCKER_TEST_ENV.md)。
- 架构与密钥管理建议：见 [ARCHITECTURE.md](./ARCHITECTURE.md)（K8s Secret / Vault 等）。
