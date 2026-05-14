# CI 与发版（Docker Hub）

本文说明 **GitHub Actions** 下的版本号文件、镜像构建与推送规则（对齐 ZoeGate 思路：根目录 `VERSION`、SemVer tag、双路径自动推 semver 镜像、手动 `dev` / `sha-*`）。**不引入 Python**；校验脚本为 Shell。

## 版本源与本地校验

- 仓库根目录 **`VERSION`**：单行 SemVer，**无** `v` 前缀（例如 `1.2.3` 或 `1.0.0-rc.1`）。
- Git 发布 tag 必须为 **`v` + `VERSION` 内容**（例如 `VERSION` 为 `1.2.3` 时打 tag `v1.2.3`）。
- 本地或 CI 前自检：

```bash
bash scripts/check_version.sh
```

GitLab CI 若复用同一仓库，可设置环境变量 **`CI_COMMIT_TAG`**（与 GitHub 的 `GITHUB_REF_NAME` 二选一），脚本逻辑与 ZoeGate `check_version` 校验一致。

## GitHub Secrets（Docker Hub）

在仓库 **Settings → Secrets and variables → Actions** 中配置（名称需与 workflow 一致）：

| Secret | 说明 |
|--------|------|
| `DOCKERHUB_USERNAME` | Docker Hub 用户名 |
| `DOCKERHUB_TOKEN` | Docker Hub Access Token 或密码（推荐 Token） |

镜像仓库固定为 **`dockersenseyang/ssh_port_forwarder`**（定义在 workflow `env.IMAGE_NAME`）。**不在 CI 中推 GHCR**。

## 自动发布（semver 镜像）

工作流：[`.github/workflows/ci.yml`](../.github/workflows/ci.yml)。

在满足 **release-gate**（见下）且 **全部现有 CI job**（前端 lint、web 构建、后端 lint、单测、SQLite/MySQL 集成测）**成功后**，若目标 **`dockersenseyang/ssh_port_forwarder:$(cat VERSION)`** 在 Docker Hub **尚不存在**，则 **build 并 push** 该 tag；**已存在则跳过**（与 ZoeGate `docker manifest inspect` 语义一致）。

**不**在自动流水线中创建或更新 **`latest`**；`latest` 由维护者单独处理（与 semver tag 解耦）。

**审阅说明（`:latest` 与示例清单）**：根目录 [`docker-compose.yml`](../docker-compose.yml) 中镜像 tag 通过环境变量 **`SPF_IMAGE_TAG`** 引用（未设置时默认 `latest`），[`deploy/kubernetes/deployment.yaml`](../deploy/kubernetes/deployment.yaml) 中镜像 tag 与 **`VERSION`** 对齐的 semver，均为**有意设计**：Compose 侧保留「未设置 `.env` 时仍可尝试拉 tag」的快捷默认值，而 CI 仍**只**自动推送 semver；二者并不矛盾。若 reviewer 或用户误以为「仓库应自动更新 `latest`」，请以本节为准——**不会**在 CI 中推送 `latest`。

### 触发条件（双路径，与 ZoeGate 对齐）

1. **推送 Git tag `v*`**（排除无意义的 `vdev` 特例）：例如推送 `v1.0.0` 且 `VERSION` 文件内容为 `1.0.0`，且 CI 全绿 → 推 `dockersenseyang/ssh_port_forwarder:1.0.0`。
2. **向 `main` 或 `release/**` 推送提交，且该提交相对上一提交改动了 `VERSION` 文件**：CI 全绿 → 同样按当前 `VERSION` 推 semver 镜像。

**release-gate** 由 [`scripts/ci_release_gate.sh`](../scripts/ci_release_gate.sh) 根据 `github` 上下文输出 `publish=true/false`。

### 触发 CI 的分支 / tag

- `pull_request` → `main`
- `push` → `main`、`release/**`、以及 tags `v*`

## 手动发布（`dev` / `sha-*`）

工作流：[`.github/workflows/docker-manual.yml`](../.github/workflows/docker-manual.yml)（**Run workflow**）。

输入 **`image_tag`** 仅允许：

- 固定字符串 **`dev`**，或  
- 符合 **`^sha-[A-Za-z0-9._-]+$`** 的标签（例如 `sha-abc1234`）。

推送 **`dockersenseyang/ssh_port_forwarder:<image_tag>`**。不要求等待完整 CI 通过（与 ZoeGate manual dev 镜像一致）。

## 发版操作示例（semver）

1. 更新根目录 **`VERSION`**（例如 `0.2.0`），提交并合并到 `main`（或 `release/*`）。
2. 打 tag **`v0.2.0`** 并推送：`git tag v0.2.0 && git push origin v0.2.0`（须与 `VERSION` 一致）。
3. 在 Actions 中等待 **CI** workflow 成功；若 `0.2.0` 镜像已存在则该次不会重复推送。

仅改 `VERSION` 而不打 tag 时：推送到 `main` / `release/**` 且 diff 包含 `VERSION`，也会在 CI 全绿后触发一次 semver 推送（适用于希望「合并即出镜像」的流程）。

## 相关文档

- 部署与 Compose：见 [DEPLOYMENT.md](./DEPLOYMENT.md)。
- 根目录 [README.md](../README.md) 中的容器与升级说明。
