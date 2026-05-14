# 本地 Docker 测试环境（基于正式 Compose 扩展）

仓库仅维护根目录正式编排 [`docker-compose.yml`](../docker-compose.yml)（MySQL + 发布镜像）。原「一体化 local-test」栈不再作为独立文件维护；以下说明如何在正式 Compose 之上或并行自建编排，完成 UI/API 与 SSH fixture 验收。

**安全提示**：下文密码、JWT、加密密钥、SSH 私钥均为**本地测试 fixture**，勿用于生产或个人基础设施。

## 方式一：在正式栈上附加 SSH fixture（端口不冲突）

在已通过 `docker compose up -d` 启动官方 `mysql` + `spf` 的前提下，将下列保存为仓库根目录旁的任意文件（例如 `spf-ssh-fixtures.yml`），**不要**在其中再次定义 `mysql` 或 `spf`：

```yaml
services:
  ssh-password:
    image: lscr.io/linuxserver/openssh-server:latest
    container_name: spf-local-ssh-password
    environment:
      PUID: 1000
      PGID: 1000
      TZ: Asia/Shanghai
      PASSWORD_ACCESS: "true"
      USER_NAME: testuser
      USER_PASSWORD: testpass123
    ports:
      - "2222:2222"
    volumes:
      - ssh-password-config:/config

  ssh-key:
    image: lscr.io/linuxserver/openssh-server:latest
    container_name: spf-local-ssh-key
    environment:
      PUID: 1000
      PGID: 1000
      TZ: Asia/Shanghai
      USER_NAME: keyuser
      PUBLIC_KEY: ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHdOtIbc4G8PIRvJ/4hdsyc+gVftBS+01nNw71Q66z5K spf-local-test-key-not-for-production
    ports:
      - "2223:2222"
    volumes:
      - ssh-key-config:/config

volumes:
  ssh-password-config:
  ssh-key-config:
```

启动：

```bash
docker compose -f docker-compose.yml -f spf-ssh-fixtures.yml up -d
```

在 **Docker 内运行的 SPF 应用** 中创建 SSH Host 时，主机名填 **`ssh-password` / `ssh-key`**，端口 **`2222`**（容器间通信均为 2222；宿主机映射为 2222 / 2223）。

若 SPF 在**宿主机**以 `go run` 运行而 fixture 在 Docker 中，则 Host 填 **`127.0.0.1`**，端口 **`2222` / `2223`**（对应该映射）。

## 方式二：源码级 backend + frontend + MySQL + SSH（原 local-test 等价）

与正式 `docker-compose.yml` **同时**占用时易冲突（尤其 `8080`、`3306`）。请 **`docker compose down`** 停掉正式栈，或改用下文**全部不同**的宿主机端口。

将下列完整内容保存为**本地自用**文件（例如 `docker-compose.override.yml` 且临时移走/重命名官方 `docker-compose.yml`，或固定使用第二文件名 `spf-local-dev.yml`），在项目根目录执行：

```bash
docker compose -f spf-local-dev.yml --project-name spf-local-dev up -d
```

**`spf-local-dev.yml` 示例**（与历史 local-test 行为一致；MySQL 使用 **`3307:3306`** 以免与将来同时运行的其它 MySQL 冲突）：

```yaml
name: spf-local-dev

services:
  mysql:
    image: mysql:8.0
    container_name: spf-local-dev-mysql
    environment:
      MYSQL_ROOT_PASSWORD: spf_local_test_root
      MYSQL_DATABASE: spf_local_test
    ports:
      - "3307:3306"
    volumes:
      - spf-local-dev-mysql-data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-uroot", "-pspf_local_test_root"]
      interval: 5s
      timeout: 5s
      retries: 20

  backend:
    image: golang:1.25-alpine
    container_name: spf-local-dev-backend
    working_dir: /workspace
    command: sh -c "go run ./cmd/server/ -config /tmp/spf-local-test-no-file.yaml"
    environment:
      GOPROXY: https://goproxy.cn,direct
      GOPRIVATE: ""
      GONOSUMDB: ""
      SERVER_HOST: 0.0.0.0
      SERVER_PORT: 8080
      SERVER_ENV: development
      DATABASE_TYPE: mysql
      SPF_DB_DSN: root:spf_local_test_root@tcp(mysql:3306)/spf_local_test?charset=utf8mb4&parseTime=true&loc=Local
      JWT_SECRET_CURRENT: local-test-jwt-secret-not-for-production
      # 以下为文档示例随机密钥（勿用于生产）；亦可自行 openssl rand -base64 32
      SPF_ENCRYPTION_KEY: VO/LNju26T7/QmFljBsajoAgcXhNCk4IpNegJr+nKvs=
      SPF_DEFAULT_ADMIN_USER: admin
      SPF_DEFAULT_ADMIN_PASS: admin123
    ports:
      - "8080:8080"
      - "5173:5173"
    volumes:
      - .:/workspace
      - go-mod-cache:/go/pkg/mod
      - go-build-cache:/root/.cache/go-build
    depends_on:
      mysql:
        condition: service_healthy

  frontend:
    image: node:22-alpine
    container_name: spf-local-dev-frontend
    working_dir: /workspace/web
    command: sh -c "npm ci && npm run dev -- --host 0.0.0.0"
    volumes:
      - ./web:/workspace/web
      - web-node-modules:/workspace/web/node_modules
    network_mode: "service:backend"
    depends_on:
      backend:
        condition: service_started

  ssh-password:
    image: lscr.io/linuxserver/openssh-server:latest
    container_name: spf-local-dev-ssh-password
    environment:
      PUID: 1000
      PGID: 1000
      TZ: Asia/Shanghai
      PASSWORD_ACCESS: "true"
      USER_NAME: testuser
      USER_PASSWORD: testpass123
    ports:
      - "2222:2222"
    volumes:
      - ssh-password-config:/config

  ssh-key:
    image: lscr.io/linuxserver/openssh-server:latest
    container_name: spf-local-dev-ssh-key
    environment:
      PUID: 1000
      PGID: 1000
      TZ: Asia/Shanghai
      USER_NAME: keyuser
      PUBLIC_KEY: ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHdOtIbc4G8PIRvJ/4hdsyc+gVftBS+01nNw71Q66z5K spf-local-test-key-not-for-production
    ports:
      - "2223:2222"
    volumes:
      - ssh-key-config:/config

volumes:
  spf-local-dev-mysql-data:
  go-mod-cache:
  go-build-cache:
  web-node-modules:
  ssh-password-config:
  ssh-key-config:
```

常用命令：

```bash
docker compose -f spf-local-dev.yml --project-name spf-local-dev ps
docker compose -f spf-local-dev.yml --project-name spf-local-dev logs -f
docker compose -f spf-local-dev.yml --project-name spf-local-dev down
docker compose -f spf-local-dev.yml --project-name spf-local-dev down -v
```

## 默认端口与 URL

| 组件 | 宿主机 URL / 端口 | Compose 服务内 |
|------|-------------------|----------------|
| 前端（方式二 Vite） | <http://localhost:5173> | `frontend` 与 `backend` 共享网络命名空间 |
| 后端 API（方式二） | <http://localhost:8080> | `backend:8080` |
| MySQL（方式二） | `127.0.0.1:3307` | `mysql:3306` |
| 密码 SSH fixture | `127.0.0.1:2222` | `ssh-password:2222` |
| 公钥 SSH fixture | `127.0.0.1:2223`（映射到容器 2222） | `ssh-key:2222` |

## 应用登录

默认管理员（由 `SPF_DEFAULT_ADMIN_USER` / `SPF_DEFAULT_ADMIN_PASS` 或后端默认逻辑创建）：

- 用户名：`admin`
- 密码：`admin123`

## SSH 密码 fixture

在应用内创建 SSH Host：

- Host：`ssh-password`（SPF 在 compose 同网络内）或 `127.0.0.1`（SPF 在宿主机）
- Port：`2222`
- 用户名：`testuser`
- 认证：密码 `testpass123`

宿主机连通性检查：

```bash
ssh -p 2222 -o StrictHostKeyChecking=no testuser@127.0.0.1 true
# 密码: testpass123
```

## SSH 公钥 fixture

在应用内创建 SSH Host：

- Host：`ssh-key` 或 `127.0.0.1`
- Port：容器网络内 `2222`；宿主机连 key 服务用 **`2223`**
- 用户名：`keyuser`
- 认证：与下方 `PUBLIC_KEY` 配对的 **OpenSSH 私钥 PEM**（仓库内**不**在 Markdown 中嵌入 PEM 明文，避免 push protection / secret 扫描误报）

与 `ssh-key` 服务环境变量 `PUBLIC_KEY` 一致的公钥行：

`ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHdOtIbc4G8PIRvJ/4hdsyc+gVftBS+01nNw71Q66z5K spf-local-test-key-not-for-production`

私钥素材为 Base64（UTF-8 PEM 全文）单行文件，见仓库 [`scripts/fixtures/spf-local-test-ed25519-openssh-inline.pem.b64`](../scripts/fixtures/spf-local-test-ed25519-openssh-inline.pem.b64)（说明见同目录 [`README.md`](../scripts/fixtures/README.md)）。

宿主机解码并连通性测试：

```bash
openssl base64 -d -in scripts/fixtures/spf-local-test-ed25519-openssh-inline.pem.b64 -out /tmp/spf-local-test-key.pem
chmod 600 /tmp/spf-local-test-key.pem
ssh -i /tmp/spf-local-test-key.pem -p 2223 -o StrictHostKeyChecking=no keyuser@127.0.0.1 true
```

在 SPF UI 中粘贴私钥时，使用解码后的 **`/tmp/spf-local-test-key.pem` 全文**（或你自行解码得到的 PEM 文件内容）。

## MySQL（方式二）

- 宿主机：`127.0.0.1:3307`
- 容器内主机名：`mysql`
- 库名：`spf_local_test`，用户 `root`，密码 `spf_local_test_root`
- 后端 DSN：`root:spf_local_test_root@tcp(mysql:3306)/spf_local_test?charset=utf8mb4&parseTime=true&loc=Local`

## 排障

- **端口占用**：修改 YAML 中宿主机侧映射（`8080`、`5173`、`3307`、`2222`、`2223`）或停止占用进程。
- **Docker 内 SPF 无法连 SSH fixture**：Host 填服务名 `ssh-password` / `ssh-key`，端口 `2222`，勿填 `127.0.0.1`（除非 SPF 跑在宿主机）。
- **私钥权限**：宿主机使用私钥文件前执行 `chmod 600`。
- **前端 API 失败（方式二）**：确认 `backend` 与 `frontend` 均已启动；`frontend` 使用 `network_mode: service:backend` 以便 Vite 将 `/api` 代理到本容器内的 `localhost:8080`。
- **清空方式二数据**：`docker compose -f spf-local-dev.yml --project-name spf-local-dev down -v`。
- **镜像拉取失败**：检查网络；涉及镜像含 `mysql:8.0`、`golang:1.25-alpine`、`node:22-alpine`、`lscr.io/linuxserver/openssh-server:latest`。

## 与正式部署文档的关系

一体化演示、K8s、公开镜像等见 [DEPLOYMENT.md](./DEPLOYMENT.md)。
