# SSH Port Forwarder Load Balancing Service — Architecture Design

## 1. 整体架构设计

### 1.1 系统定位

本服务（简称 **SPF**）部署于公网或可控网络环境中，作为一个 **SSH Client** 主动连接多台位于目标网络内部的 SSH Host，通过 SSH 端口转发（LocalPortForwarding / DynamicForwarding）将目标网络内的 TCP 服务暴露到本服务的指定端口上。系统具备多 Host 横向扩展能力，支持在多个等价 SSH Host 之间进行负载均衡与故障切换，保证转发服务的高可用。

### 1.2 整体架构图

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              SPF Service                                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────────────┐   │
│  │  Web UI  │  │  REST API│  │ Prometheus│  │  SSH Connection Manager  │   │
│  │  (Vue/Go)│  │  (Gin)   │  │ /metrics   │  │                          │   │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └──────────┬───────────────┘   │
│       │              │              │                    │                  │
│       └──────────────┬┴──────────────┴────────────────────┘                  │
│                      │                                                        │
│              ┌───────┴────────┐                                              │
│              │   Core Engine  │                                              │
│              │  ┌──────────┐  │                                              │
│              │  │ Scheduler │  │                                              │
│              │  │  (Go Routines)│ │                                            │
│              │  └──────────┘  │                                              │
│              │  ┌──────────┐  │                                              │
│              │  │  Health  │  │                                              │
│              │  │  Checker  │  │                                              │
│              │  └──────────┘  │                                              │
│              │  ┌──────────┐  │                                              │
│              │  │  LB Pool │  │                                              │
│              │  │          │  │                                              │
│              │  └──────────┘  │                                              │
│              └───────┬────────┘                                              │
│                      │                                                        │
│              ┌───────┴────────┐                                              │
│              │  DB Abstraction│                                             │
│              │  (GORM + Dialect)│                                            │
│              └───────┬────────┘                                              │
│                      │                                                        │
└──────────────────────┼──────────────────────────────────────────────────────┘
                       │
          ┌────────────┼──────────────────────────────────────┐
          │            │                                       │
  ┌───────┴───────┐    │   ┌───────────────┐                   │
  │    SQLite     │    │   │     MySQL     │                   │
  └───────────────┘    │   └───────────────┘                   │
                        │
          ┌─────────────┴──────────────────────────────────────┐
          │            Target Network (LAN)                     │
          │                                                       │
   ┌──────┴───────┐          ┌────────┐          ┌────────┐      │
   │  SSH Host A  │◄─────────│ SSH Host B│◄────────│SSH Host C│     │
   │ (10.0.0.101) │          │(10.0.0.102)│         │(10.0.0.103)│    │
   │  port 22     │          │  port 22    │         │  port 22   │    │
   └──────┬───────┘          └─────┬──────┘          └─────┬────┘    │
          │                         │                        │          │
          │  ┌──────────────────────┼────────────────────────┤          │
          │  │                     │                        │          │
          │  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  │          │
          │  └──│ Service  │  │ Service  │  │ Service  │  │          │
          │     │  :3306    │  │  :6379   │  │  :8080   │  │          │
          │     └──────────┘  └──────────┘  └──────────┘  │          │
          └─────────────────────────────────────────────────┘          │
                                                                       │
                     ┌──────────────────────────────────────────────┐  │
                     │              SPF Listener Ports              │  │
                     │   :12001 ──► MySQL  │  :12002 ──► Redis      │  │
                     │   :12003 ──► HTTP   │  :12004 ──► PostgreSQL  │  │
                     └──────────────────────────────────────────────┘  │
                                                                       │
                     ┌──────────────────────────────────────────────┐  │
                     │                  End Users                    │  │
                     │  mysql -h SPF_IP -P 12001 ...               │  │
                     └──────────────────────────────────────────────┘  │
```

### 1.3 数据流概述

1. **配置下发**：用户通过 Web UI 或 REST API 创建转发规则（target_host, target_port, local_port, SSH_host_id）
2. **连接建立**：SSH Manager 根据规则连接到指定 SSH Host，建立 Local Port Forwarding
3. **流量转发**：用户连接 SPF 的 local_port，数据经 SSH Tunnel 到达目标网络服务
4. **健康检查**：Health Checker 定期检测各 SSH Host 连通性和端口可达性
5. **故障切换**：当某 SSH Host 断线，LB Pool 自动将流量切换到其他健康节点
6. **状态持久化**：所有配置、状态通过 DB Abstraction 层写入 SQLite/MySQL；审计日志同时输出到 stdout 供外部采集

---

## 2. 技术栈选型

### 2.1 语言与运行时

| 层级 | 选项 | 推荐选择 | 说明 |
|------|------|----------|------|
| **核心引擎** | Go 1.22+ | **Go** | 原生协程适合高并发 SSH 连接管理，标准库 `crypto/ssh` 提供 SSH 客户端能力，编译为单一二进制，部署简单 |
| **Web 框架** | Gin / Echo / stdlib | **Gin** | 成熟稳定，中间件生态丰富，性能足够 |
| **ORM** | GORM | **GORM v2** | 支持 SQLite/MySQL，插件化设计，迁移工具完善 |
| **前端** | Vue3 + Vite / React | **Vue3 + Vite** | 轻量、学习曲线平缓，适合内部管理界面 |
| **HTTP 客户端** | net/http / golang.org/x/net | **net/http** | SSH 连接检测使用 net.Dial |
| **SSH 库** | golang.org/x/crypto/ssh | **golang.org/x/crypto/ssh** | 官方标准库，支持 KeepAlive、JumpHost |
| **加密库** | crypto/aes + cipher | **AES-256-GCM** | 用于 SSH 凭证加密存储，密钥通过 K8S Secret / Vault 注入 |
| **指标采集** | prometheus/client_golang | **prometheus/client_golang** | 事实标准，集成 Gin 中间件 |
| **配置管理** | Viper / stdlib json | **Viper** | 支持 YAML/JSON/ENV，多数据源配置 |

### 2.2 数据库选型

| 数据库 | 适用场景 | 说明 |
|--------|----------|------|
| **SQLite** | 单机部署、开发测试、小规模使用 | 无依赖，文件级持久化，性能足够（读密集场景 QPS < 1000） |
| **MySQL 8.0+** | 生产环境、多实例部署 | 主从复制支持，连接池成熟，高并发写入支持更好 |

> **关键设计**：通过 GORM 的 Dialector 抽象层，两种数据库使用同一套 Model 定义和 Repository 接口，业务代码零感知。

### 2.3 技术栈汇总

```
┌─────────────────────────────────────────────┐
│              Frontend (Optional SPA)         │
│         Vue3 + Vite + TailwindCSS           │
└──────────────────┬──────────────────────────┘
                   │ HTTP/REST
┌──────────────────▼──────────────────────────┐
│           Backend (Go 1.22+)                 │
│  ┌─────────────────────────────────────┐   │
│  │  Gin Router + Middleware             │   │
│  │  - Auth (JWT)                        │   │
│  │  - CORS                              │   │
│  │  - Prometheus Middleware              │   │
│  └─────────────────────────────────────┘   │
│                                             │
│  ┌─────────┐ ┌─────────┐ ┌──────────────┐  │
│  │ REST API│ │ WebSocket│ │ /metrics     │  │
│  │ Handler │ │ Handler  │ │ Endpoint     │  │
│  └────┬────┘ └────┬────┘ └──────┬───────┘  │
│       └───────────┴──────────────┘          │
│                    │                        │
│  ┌─────────────────▼──────────────────────┐  │
│  │         Core Engine (Go Routines)     │  │
│  │  ┌────────┐ ┌────────┐ ┌──────────┐  │  │
│  │  │Scheduler│ │Health  │ │  LB Pool  │  │  │
│  │  │        │ │Checker │ │          │  │  │
│  │  └────────┘ └────────┘ └──────────┘  │  │
│  │  ┌────────────────────────────────┐  │  │
│  │  │   SSH Connection Manager       │  │  │
│  │  │   (crypto/ssh + KeepAlive)     │  │  │
│  │  └────────────────────────────────┘  │  │
│  └─────────────────┬────────────────────┘  │
│                    │                        │
│  ┌─────────────────▼────────────────────┐    │
│  │      DB Abstraction (GORM v2)      │    │
│  │   ┌──────────┐   ┌──────────────┐  │    │
│  │   │ SQLite 3 │   │ MySQL 8.0+   │  │    │
│  │   └──────────┘   └──────────────┘  │    │
│  └────────────────────────────────────┘    │
└─────────────────────────────────────────────┘
```

---

## 3. 核心模块设计

### 3.1 模块总览

```
ssh-port-forwarder/
├── cmd/                    # 入口
├── internal/
│   ├── config/            # 配置加载
│   ├── model/             # 数据模型（数据库实体）
│   ├── repository/        # 数据访问层（DB 抽象）
│   ├── service/           # 业务逻辑层
│   │   ├── ssh_manager/   # SSH 连接管理（核心）
│   │   ├── health/        # 健康检查
│   │   ├── scheduler/     # 调度器
│   │   └── lb/            # 负载均衡器
│   ├── handler/           # HTTP Handlers
│   ├── middleware/        # Gin Middleware
│   └── pkg/               # 内部工具包
├── migrations/            # 数据库迁移
├── web/                   # Vue 前端源码（可选）
└── Makefile / Dockerfile
```

### 3.2 SSH Connection Manager（`service/ssh_manager/`）

**职责**：管理与目标网络 SSH Host 的连接生命周期。

**关键设计**：

```go
// 连接状态机
type ConnState int
const (
    ConnStateDisconnected ConnState = iota
    ConnStateConnecting
    ConnStateConnected
    ConnStateReconnecting
    ConnStateFailed
)

// SSHClient 代表与一个 SSH Host 的一个连接会话
type SSHClient struct {
    mu         sync.RWMutex
    client     *ssh.Client
    config     *ssh.ClientConfig
    host       *model.SSHHost        // 从 DB 加载
    state      ConnState
    reconnCh   chan struct{}           // 重连信号
    forwards   map[uint]*ForwardEntry // 该 SSH 连接上的转发映射
}

// ForwardEntry 本地监听端口到远程目标的映射
type ForwardEntry struct {
    ID          uint
    LocalAddr   string  // "0.0.0.0:12001"
    RemoteAddr  string  // "10.0.0.101:3306"
    listener    net.Listener
    stopCh      chan struct{}
}
```

**核心行为**：

1. **建立连接**：通过 SSH Key 或 Password 认证，连接到 SSH Host
2. **端口转发**：`ssh.NewClient` + `client.Listen` 组合实现 Local Port Forwarding（类似于 `ssh -L local_port:remote_host:remote_port jump_host`）
3. **KeepAlive**：每 15s 发送 `ssh.Ping` 包，检测存活
4. **自动重连**：连接断开后，backoff 重试（1s → 2s → 4s → 8s → max 60s）
5. **优雅关闭**：收到关闭信号后，停止监听端口并关闭 SSH Client

**与 Health Checker 的协作**：
- Health Checker 将检测结果写入 `model.SSHHost.HealthStatus`
- SSH Manager 订阅状态变化，当 Host 被标记为 `unhealthy` 时主动断开并进入重连循环

### 3.3 Health Checker（`service/health/`）

**职责**：周期性检测 SSH Host 和端口转发的健康状态。

**检测策略**：

| 检测对象 | 检测方式 | 频率 | 说明 |
|----------|----------|------|------|
| SSH Host 连通性 | TCP 握手 SSH 端口 | 每 10s | 检测 SSH Host 网络层是否可达 |
| SSH 认证连通性 | 实际 SSH 连接 + 简单 exec | 每 30s | 验证认证凭证有效性 |
| 端口转发可用性 | 通过 SSH Tunnel 对目标地址执行 `net.Dial` TCP 握手 | 每 15s | **转发健康 = TCP 连通**，即通过已建立的 SSH Tunnel 向目标 `TargetHost:TargetPort` 发起 TCP 连接，成功即为健康 |

> **关键定义**：端口转发是否健康，等价于通过 SSH Tunnel 对目标服务端口的 TCP 连通性检测。不检测应用层协议，只验证 TCP 三次握手是否成功。

**健康度计算**：

```
health_score = (success_count / total_checks_in_window) * 100

窗口：最近 5 分钟（可配置）
阈值：health_score < 60% → unhealthy
      health_score >= 60% → healthy
      连续 3 次失败 → immediately unhealthy
```

**事件发布**：
- 检测结果通过 Go Channel 通知 `LB Pool` 和 `SSH Manager`
- 状态变更写入 DB，用于 Web UI 展示历史健康度

### 3.4 Load Balancer Pool（`service/lb/`）

**职责**：在多个等价 SSH Host 之间**以规则为粒度**分配转发，实现故障切换与负载分担。

> **关键定义**：LB 的调度粒度为 **规则级（Rule-Level）**，即每条 ForwardRule 在创建/故障切换时选择一个 Host 建立 Tunnel，该规则的所有用户 TCP 连接均通过同一条 Tunnel 转发。LB 不在单条规则内的多个连接之间做分配。

**关键概念**：

```go
// ForwardGroup 一组指向同一目标服务的转发规则（可绑定多个 SSH Host）
type ForwardGroup struct {
    ID           uint
    name         string
    strategy     LBStrategy   // round_robin / least_rules / weighted
    hosts        []*SSHHostRef // 参与该组的所有 SSH Host
    currentIndex int           // round_robin 计数器
    mu           sync.Mutex
}

type LBStrategy int
const (
    RoundRobin LBStrategy = iota  // 同 Group 下多条 Rule 轮流分配到不同 Host
    LeastRules                     // 当前承载最少活跃规则的 Host 优先
    Weighted                       // 按权重分配 Rule 到 Host
)
```

**策略语义说明（规则级）**：

| 策略 | 含义 | 适用场景 |
|------|------|----------|
| `RoundRobin` | 同一 Group 下多条 Rule 依次轮流分配到不同 Host | Group 内有多条 Rule 需要均匀分散 |
| `LeastRules` | 当前承载最少活跃 Rule 的 Host 优先被选中 | 希望 Host 负载均衡（按规则数量） |
| `Weighted` | 高权重 Host 承载更多 Rule；故障恢复后高权重 Host 优先切回 | Host 性能/带宽不对等时使用 |

> **注意**：当一个 Group 只有 1 条 Rule 时，LB 策略实质退化为 **优先级排序 + 故障切换**（Active-Standby）。

**连接模型**：

- 每个 `ForwardRule` 绑定一个 `ForwardGroup`，Group 内包含多个等价 SSH Host
- **只有当前 Active Host 建立 SSH Tunnel**，其他 Host 不建立备用连接
- 一条 Rule 的所有用户 TCP 连接均通过该 Active Tunnel 多路复用转发（SSH Channel）
- 当 Active Host 故障时，从 Group 中按策略选择下一个健康 Host 建立新 Tunnel

**故障切换流程**：

```
转发规则启动时：
        │
        ▼
LB Pool 按策略从 ForwardGroup 中选取一个 Host 作为 Active
        │
        ▼
SSH Manager 建立到 Active Host 的 SSH Tunnel
        │
        ▼
用户流量通过该 Tunnel 转发
        │
    Active Host 健康检查失败 / 连接断开 →
        │
        ▼
LB Pool 从 Group 中选取下一个健康 Host
        │
        ▼
SSH Manager 建立新 Tunnel，切换转发
        │
    所有 Host 均不可用 → 标记规则为 inactive，等待恢复
```

**动态更新**：SSH Manager 检测到原 Active Host 连接恢复后，根据策略决定是否切回（weighted 模式下高权重 Host 恢复后切回，round_robin 模式下不主动切回）。

### 3.5 Scheduler（`service/scheduler/`）

**职责**：协调各模块的运行节奏，处理定时任务。

**调度任务列表**：

| 任务 | 周期 | 说明 |
|------|------|------|
| Health Check | 10s | 遍历所有 SSH Host，执行健康检测 |
| Reconnect Loop | 5s | 检查断开的 SSH 连接，触发重连 |
| Metrics Flush | 15s | 将内存中的运行时指标写入 DB（健康度历史） |
| Config Reload | 30s | 检查配置文件变更（支持 SIGHUP 热重载） |
| Cleanup DeadConn | 60s | 清理已标记为 dead 超过 5 分钟的连接记录 |

### 3.6 Web 管理界面

**功能模块**：

| 模块 | 功能 |
|------|------|
| **登录认证** | 用户名/密码登录，JWT Token（24h 有效期），RefreshToken（7d），支持 JWT Secret 轮转 |
| **SSH Host 管理** | 添加/编辑/删除 SSH Host（Host, Port, AuthMethod, 权重） |
| **转发规则管理** | 添加/编辑/删除转发规则（LocalPort, TargetHost, TargetPort, 绑定 Host 组） |
| **健康状态面板** | 实时显示各 Host/规则状态（WebSocket 推送） |
| **健康度历史** | 折线图展示最近 24h / 7d 健康度 |
| **操作日志** | 记录所有配置变更和重要系统事件（日志输出到控制台 stdout/stderr，由外部工具如 Loki 采集管理） |
| **Prometheus 指标** | 跳转链接到 `/metrics` |

**API 认证**：所有 REST API（除 `/api/auth/login`）需要携带 `Authorization: Bearer <jwt_token>` Header。

### 3.7 Prometheus 指标接口

`GET /metrics` 由 `promhttp.Handler` 提供，除 Go/进程/抓取器相关默认指标外，业务指标在 `internal/pkg/metrics/metrics.go` 中注册，命名空间为 `spf_`：

| 指标 | 类型 | 标签 | 说明 |
|------|------|------|------|
| `spf_host_health` | Gauge | `host_id`, `host_name` | 1/0 表示健康检查判定是否 healthy |
| `spf_host_latency_seconds` | Histogram | `host_id`, `host_name` | SSH keepalive 往返时延（秒），仅成功时 Observe |
| `spf_host_rule_load` | Gauge | `host_id`, `host_name` | 当前 `active` 且 `active_host` 指向该主机的规则数；在分配、failover、规则创建/删除/重启等路径刷新 |
| `spf_rule_health` | Gauge | `rule_id`, `rule_name`, `group_id` | 与健康检查中 rule 端探测结果一致，检查周期与 Scheduler 中健康任务一致 |
| `spf_rule_host_switch_total` | Counter | `rule_id`, `rule_name`, `from_host_id`, `to_host_id` | `lb.Pool.failoverRule` 在更新 active host 成功时递增 |
| `spf_host_group_info` | Gauge | `host_id`, `host_name`, `group_id`, `group_name` | 恒为 1；在将 Host 加入/移出 Group 的 API 中增删；删除 Host 时若未先移出组，该映射可能需依赖运维侧清理或后续扩展 |

```
spf_host_health{host_id="1",host_name="node-a"} 1
spf_rule_health{group_id="1",rule_id="1",rule_name="rule_30000"} 1
spf_host_group_info{group_id="1",group_name="default",host_id="1",host_name="node-a"} 1
```

---

## 4. 数据模型设计（数据库抽象层）

### 4.1 实体关系图

```
User (1) ──────< (N) AuditLog
  │
  │ 1:N
  ▼
SSHHost (1) ──< (N) ForwardRule
  │
  │ N:M（通过 ForwardGroupHost 关联）
  ▼
ForwardGroup (1) ──< (N) ForwardRule
  │
  ▼
HealthHistory ── 记录各 Host 的历史健康状态
```

### 4.2 模型定义

**关键约束**：

- 所有表使用 `BIGINT` 作为主键（`id`），避免 MySQL 8.0 之前的 auto_increment 问题
- 时间字段统一使用 `BIGINT`（Unix 秒）存储（跨时区一致）
- 软删除：配置类数据（Host、Rule）使用 `deleted_at` 软删除，防止误删

```go
// model/user.go
type User struct {
    ID           uint64     `gorm:"primaryKey;autoIncrement"`
    Username     string     `gorm:"size:64;uniqueIndex;not null"`
    PasswordHash string     `gorm:"size:256;not null"` // bcrypt
    Role         string     `gorm:"size:32;default:'operator'"` // admin / operator（operator 为占位角色，仅具备登录权限，不具备任何操作权限，用于未来多角色扩展）
    CreatedAt    int64      `gorm:"autoCreateTime:millis"`
    UpdatedAt    int64      `gorm:"autoUpdateTime:millis"`
    DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// model/ssh_host.go
type SSHHost struct {
    ID            uint64     `gorm:"primaryKey;autoIncrement"`
    Name          string     `gorm:"size:128;not null"`
    Host          string     `gorm:"size:255;not null"` // IP or domain
    Port          int        `gorm:"default:22"`
    AuthMethod    string     `gorm:"size:32;not null"` // "password" / "private_key"
    AuthData      string     `gorm:"size:2048"`         // AES-256-GCM 加密存储（密码或私钥内容），密钥通过 K8S Secret / Vault 注入
    AuthNonce     string     `gorm:"size:64"`            // AES-256-GCM nonce（Base64 编码）
    Weight        int        `gorm:"default:100"`       // LB 权重 1-100
    HealthStatus  string     `gorm:"size:16;default:'unknown'"` // healthy / unhealthy / unknown
    HealthScore   float64    `gorm:"default:0"`          // 0-100
    LastCheckAt   int64
    LastSuccessAt int64
    CreatedAt     int64
    UpdatedAt     int64
    DeletedAt     gorm.DeletedAt
}

// model/forward_group.go
type ForwardGroup struct {
    ID        uint64     `gorm:"primaryKey;autoIncrement"`
    Name      string     `gorm:"size:128;not null"`
    Strategy  string     `gorm:"size:32;default:'round_robin'"` // round_robin / least_rules / weighted
    Hosts     []SSHHost  `gorm:"many2many:forward_group_hosts;"`
    Rules     []ForwardRule `gorm:"foreignKey:GroupID"`
    CreatedAt int64
    UpdatedAt int64
    DeletedAt gorm.DeletedAt
}

// model/forward_rule.go（节选；完整字段见仓库源码）
type ForwardRule struct {
    ID           uint64     `gorm:"primaryKey;autoIncrement"`
    GroupID      uint64     `gorm:"index"`
    Name         string     `gorm:"size:128;not null;default:''"` // 管理员可读名称，创建时必填
    LocalPort    int        `gorm:"not null;uniqueIndex"` // 监听端口，全局唯一，范围见配置
    TargetHost   string     `gorm:"size:255;not null"`
    TargetPort   int        `gorm:"not null"`
    Protocol     string     `gorm:"size:16;default:'tcp'"`
    Status       string     // active / inactive
    ActiveHostID uint64     // 当前承载该规则的 SSH Host
    // 关联 Preload：Group、ActiveHost
    CreatedAt    int64
    UpdatedAt    int64
    DeletedAt    gorm.DeletedAt
}

// model/health_history.go
type HealthHistory struct {
    ID        uint64 `gorm:"primaryKey;autoIncrement"`
    HostID    uint64 `gorm:"index"`
    Score     float64
    IsHealthy bool
    LatencyMs float64
    CheckedAt int64  `gorm:"index"`
}

// model/audit_log.go
type AuditLog struct {
    ID         uint64 `gorm:"primaryKey;autoIncrement"`
    UserID     uint64 `gorm:"index"`
    Action     string `gorm:"size:64;not null"`  // "host.create" / "rule.delete" / ...
    TargetType string `gorm:"size:32"`            // "ssh_host" / "forward_rule"
    TargetID   uint64
    Detail     string `gorm:"type:text"`
    CreatedAt  int64
}
```

### 4.3 DB 抽象层设计

**目标**：业务层通过 Repository 接口访问数据，切换 SQLite ↔ MySQL 只需修改配置，无需改动任何业务代码。

```go
// repository/interfaces.go
type UserRepository interface {
    FindByID(id uint64) (*model.User, error)
    FindByUsername(username string) (*model.User, error)
    Create(user *model.User) error
    Update(user *model.User) error
    Delete(id uint64) error
}

type SSHHostRepository interface {
    FindAll() ([]*model.SSHHost, error)
    FindByID(id uint64) (*model.SSHHost, error)
    FindByIDs(ids []uint64) ([]*model.SSHHost, error)
    Create(host *model.SSHHost) error
    Update(host *model.SSHHost) error
    UpdateHealthStatus(id uint64, status string, score float64) error
    Delete(id uint64) error
}

type ForwardRuleRepository interface {
    FindAll() ([]*model.ForwardRule, error)
    FindByID(id uint64) (*model.ForwardRule, error)
    FindByLocalPort(port int) (*model.ForwardRule, error)
    Create(rule *model.ForwardRule) error
    Update(rule *model.ForwardRule) error
    Delete(id uint64) error
}

type ForwardGroupRepository interface {
    FindAll() ([]*model.ForwardGroup, error)
    FindByID(id uint64) (*model.ForwardGroup, error)
    Create(group *model.ForwardGroup) error
    Update(group *model.ForwardGroup) error
    Delete(id uint64) error
}

type HealthHistoryRepository interface {
    Create(h *model.HealthHistory) error
    LatestByHostID(hostID uint64, limit int) ([]*model.HealthHistory, error)
    RangeByHostID(hostID uint64, start, end int64) ([]*model.HealthHistory, error)
}

type AuditLogRepository interface {
    Create(log *model.AuditLog) error
    List(limit, offset int) ([]*model.AuditLog, int64, error)
}
```

**GORM 实现（SQLite / MySQL 通用）**：

```go
// repository/gorm_adapter.go
type GORMAdapter struct {
    db *gorm.DB
}

func NewGORMAdapter(dsn string, dialect string) (*GORMAdapter, error) {
    var gormConfig = &gorm.Config{
        NamingStrategy: schema.NamingStrategy{
            TablePrefix:   "spf_",           // 表前缀 spf_
            SingularTable: false,
        },
        CreateBatchSize: 100,
    }

    var db *gorm.DB
    var err error

    switch dialect {
    case "mysql":
        db, err = gorm.Open(mysql.Open(dsn), gormConfig)
    case "sqlite":
        db, err = gorm.Open(sqlite.Open(dsn), gormConfig)
    default:
        return nil, fmt.Errorf("unsupported dialect: %s", dialect)
    }
    if err != nil {
        return nil, err
    }

    // 自动迁移
    if err := db.AutoMigrate(
        &model.User{},
        &model.SSHHost{},
        &model.ForwardGroup{},
        &model.ForwardRule{},
        &model.HealthHistory{},
        &model.AuditLog{},
    ); err != nil {
        return nil, err
    }

    return &GORMAdapter{db: db}, nil
}

// 实现各 Repository 接口 ...
```

**依赖注入**：

```go
// service/container.go
type Container struct {
    SSHHostRepo     repository.SSHHostRepository
    ForwardRuleRepo repository.ForwardRuleRepo
    ForwardGroupRepo repository.ForwardGroupRepo
    HealthHistoryRepo repository.HealthHistoryRepository
    AuditLogRepo    repository.AuditLogRepository
    UserRepo        repository.UserRepository

    SSHManager  *ssh_manager.Manager
    HealthCheck *health.Checker
    LBPool      *lb.Pool
    Scheduler   *scheduler.Scheduler
}
```

---

## 5. 项目目录结构

```
ssh-port-forwarder/
├── cmd/
│   └── server/
│       └── main.go              # 程序入口
├── internal/
│   ├── config/
│   │   └── config.go            # 配置结构体 + Viper 加载
│   ├── model/
│   │   ├── user.go
│   │   ├── ssh_host.go
│   │   ├── forward_group.go
│   │   ├── forward_rule.go
│   │   ├── health_history.go
│   │   └── audit_log.go
│   ├── repository/
│   │   ├── interfaces.go         # Repository 接口定义
│   │   ├── gorm_adapter.go      # GORM DB 适配器（含自动迁移）
│   │   ├── user_repo.go
│   │   ├── ssh_host_repo.go
│   │   ├── forward_rule_repo.go
│   │   ├── forward_group_repo.go
│   │   ├── health_history_repo.go
│   │   └── audit_log_repo.go
│   ├── service/
│   │   ├── container.go         # 依赖注入容器
│   │   ├── auth.go              # 认证服务（JWT）
│   │   ├── ssh_manager/
│   │   │   ├── manager.go       # SSH Client 管理器
│   │   │   ├── client.go        # 单个 SSH Client 封装
│   │   │   ├── forward.go       # 端口转发逻辑
│   │   │   └── reconnect.go     # 重连策略
│   │   ├── health/
│   │   │   ├── checker.go       # 健康检查器
│   │   │   └── detector.go      # 检测器实现
│   │   ├── lb/
│   │   │   ├── pool.go          # 负载均衡池
│   │   │   └── strategy.go      # 策略实现（RR/LC/Weighted）
│   │   └── scheduler/
│   │       └── scheduler.go      # 定时任务调度器
│   ├── handler/
│   │   ├── router.go            # Gin 路由注册
│   │   ├── auth_handler.go      # 登录/登出
│   │   ├── host_handler.go      # SSH Host CRUD
│   │   ├── rule_handler.go      # 转发规则 CRUD
│   │   ├── group_handler.go     # 转发组 CRUD
│   │   ├── status_handler.go    # 状态查询
│   │   ├── health_handler.go    # 健康度历史
│   │   └── ws_handler.go        # WebSocket（实时状态推送）
│   ├── middleware/
│   │   ├── auth.go              # JWT 校验中间件
│   │   ├── audit.go             # 操作日志中间件
│   │   ├── cors.go
│   │   └── recovery.go
│   └── pkg/
│       ├── crypto/               # 密码加密/解密工具
│       ├── response/            # 统一 HTTP 响应封装
│       └── validator/           # 参数校验
├── migrations/                   # SQL 迁移脚本（备用）
│   └── 001_init.sql
├── web/                         # Vue3 前端（可选，构建为静态文件后嵌入 binary）
│   ├── src/
│   │   ├── api/                 # Axios 封装
│   │   ├── views/               # 页面
│   │   │   ├── Login.vue
│   │   │   ├── Dashboard.vue
│   │   │   ├── HostList.vue
│   │   │   ├── RuleList.vue
│   │   │   └── HealthHistory.vue
│   │   ├── router/
│   │   ├── stores/              # Pinia 状态管理
│   │   └── main.js
│   ├── index.html
│   ├── vite.config.ts
│   └── package.json
├── config/
│   └── config.yaml.example      # 配置文件示例
├── Makefile
├── Dockerfile
├── go.mod
├── go.sum
└── ARCHITECTURE.md              # 本文档
```

---

## 6. 关键接口设计

### 6.1 REST API

**Base URL**: `/api/v1`

#### 认证

```
POST /api/v1/auth/login
Request:  { "username": "admin", "password": "xxx" }
Response: { "code": 0, "data": { "token": "eyJ...", "expires_at": 1713542400 } }

POST /api/v1/auth/refresh
Request:  { "refresh_token": "xxx" }
Response: { "code": 0, "data": { "token": "eyJ..." } }

POST /api/v1/auth/logout
Response: { "code": 0 }
```

#### SSH Host 管理

```
GET    /api/v1/hosts             → List all hosts
POST   /api/v1/hosts             → Create host
GET    /api/v1/hosts/:id         → Get host detail
PUT    /api/v1/hosts/:id         → Update host
DELETE /api/v1/hosts/:id         → Delete host (soft)
POST   /api/v1/hosts/:id/test    → Test connection (SSH handshake)
POST   /api/v1/hosts/:id/copy    → Copy host (reuse ciphertext server-side; optional new plaintext in body)
```

列表/详情响应中的 Host **不包含** `auth_data` / `auth_nonce` 密文字段。

#### 转发组管理

```
GET    /api/v1/groups            → List all groups
POST   /api/v1/groups             → Create group
GET    /api/v1/groups/:id         → Get group detail
PUT    /api/v1/groups/:id         → Update group
DELETE /api/v1/groups/:id         → Delete group
POST   /api/v1/groups/:id/hosts   → Add host to group
DELETE /api/v1/groups/:id/hosts/:host_id → Remove host from group
```

#### 转发规则管理

```
GET    /api/v1/rules              → List all rules
POST   /api/v1/rules              → Create rule (body 需含 `name` 等字段)
GET    /api/v1/rules/:id          → Get rule detail
PUT    /api/v1/rules/:id          → Update rule
DELETE /api/v1/rules/:id         → Delete rule
POST   /api/v1/rules/:id/restart  → Restart forward for this rule
```

#### 系统状态

```
GET    /api/v1/status/overview    → Overall status summary
GET    /api/v1/status/hosts       → Host status list (health, score, last_check)
GET    /api/v1/status/rules       → Rule status list (active, bytes, connections)
GET    /api/v1/health-history/:host_id?start=&end=&interval=5m
                               → Health score history
```

#### 操作日志

```
GET    /api/v1/audit-logs?page=1&page_size=20&action=&user_id=
```

#### Prometheus

```
GET    /metrics                   → Prometheus scrape endpoint
```

### 6.2 WebSocket 接口

```
WS /api/v1/ws/status
→ 实时推送 Host 和 Rule 状态变更
Payload:
{
  "type": "host_status_change",
  "data": { "host_id": 1, "status": "unhealthy", "score": 45.5 }
}
{
  "type": "rule_status_change",
  "data": { "rule_id": 3, "local_port": 12001, "status": "active" }
}
```

### 6.3 统一响应格式

```go
type APIResponse struct {
    Code    int         `json:"code"`     // 0=成功，非0=错误码
    Message string      `json:"message"`  // 错误描述
    Data    interface{} `json:"data,omitempty"`
    TraceID string      `json:"trace_id,omitempty"` // 请求追踪
}

// 分页响应
type PagedResponse struct {
    Code    int         `json:"code"`
    Data    interface{} `json:"data"`
    Total   int64       `json:"total"`
    Page    int         `json:"page"`
    PageSize int        `json:"page_size"`
}
```

---

## 附录：关键设计决策说明

### Q1：为什么使用 Local Port Forwarding 而不是 Reverse Tunnel？

目标网络内 SSH Host 是主动连接**本服务**（本服务在公网，SSH Host 在内网），所以本服务作为 SSH Client 发起连接，使用 Local Port Forwarding（`-L`）将内网端口映射到本地。这是标准 SSH 端口转发模式。

### Q2：如何保证多个 Host 之间的端口转发一致性？

同一个 `ForwardRule` 绑定一个 `ForwardGroup`，Group 内有多个等价 SSH Host。用户连接时 LB Pool 选取最优 Host 建立转发。如果 Host 故障，LB 自动切换，用户会短暂断开（< 1s）后自动重连恢复。

### Q3：为什么选择 Go 而不是 Python？

- SSH 连接需要维护长连接池，Go 的 Goroutine + Channel 模型更轻量（数千连接开销极低）
- `crypto/ssh` 是官方标准库，无需额外依赖
- 最终编译为单一二进制，部署体验好

### Q4：SQLite 在高并发写入场景是否足够？

- 健康检查历史写入频率：每个 Host 每 10s 一条（假设 10 个 Host = 1 条/s）
- 这对 SQLite 完全没有压力
- 如果需要更高写入吞吐，可切换 MySQL（同代码，同接口）

### Q5：认证方案安全性？

- 密码存储使用 bcrypt（cost=12）
- JWT Secret 通过环境变量注入，启动时校验 Secret 存在
- **JWT Secret 轮转机制**：支持同时配置 `JWT_SECRET_CURRENT` 和 `JWT_SECRET_PREVIOUS` 两个 Secret。验证 Token 时优先使用 Current，失败后 fallback 到 Previous。轮转流程：
  1. 生成新 Secret，设置为 `JWT_SECRET_CURRENT`
  2. 将旧 Secret 设置为 `JWT_SECRET_PREVIOUS`
  3. 等待旧 Token 全部过期后（≥24h），移除 `JWT_SECRET_PREVIOUS`
- RefreshToken 存储在 DB 中，支持主动撤销
- 所有 API HTTPS 强制（通过反向代理如 Nginx / K8S Ingress 处理 TLS）

### Q6：SSH 凭证如何安全存储？

- SSH Host 的密码或私钥使用 **AES-256-GCM** 对称加密后存入数据库 `AuthData` 字段
- 加密密钥（32 字节）通过环境变量 `SPF_ENCRYPTION_KEY` 注入，**不存入数据库**
- 生产环境密钥来源：K8S Secret 或 HashiCorp Vault
- 每条记录使用独立随机 Nonce（存储在 `AuthNonce` 字段），确保相同明文产生不同密文
- 密钥轮转：更新 `SPF_ENCRYPTION_KEY` 后，系统启动时自动检测并重新加密所有凭证（需同时提供 `SPF_ENCRYPTION_KEY_PREVIOUS`）

### Q7：RBAC 权限模型？

- 当前版本仅支持两个角色：`admin` 和 `operator`
- **admin**：拥有所有操作权限（SSH Host / 转发规则 / 转发组 / 用户管理）
- **operator**：仅具备登录权限，不具备任何业务操作权限（占位角色）
- 设计目的：保证系统对未来多角色细粒度权限扩展的兼容性（如只读角色、特定 Group 管理员等）
- 权限校验在 Middleware 层统一拦截，非 admin 角色访问受保护 API 返回 403

### Q8：端口管理策略？

- 转发端口范围通过环境变量限制：`SPF_PORT_RANGE_MIN`（默认 30000）和 `SPF_PORT_RANGE_MAX`（默认 33000）
- 创建转发规则时，系统执行端口冲突检测：检查 DB 中是否已存在相同 LocalPort 的活跃规则
- 端口规划由用户负责，系统仅阻止冲突发生
- 超出允许范围的端口在创建时直接拒绝

### Q9：审计日志保留策略？

- 本服务**不负责日志保留和清理**
- 审计日志以结构化格式输出到 stdout（JSON 格式），供外部日志管理工具（如 Loki、ELK、Fluentd）采集
- DB 中的 AuditLog 表仅用于 Web UI 近期展示（保留最近 7 天，超期自动清理）
- 长期日志归档和检索由外部基础设施承担

### Q10：ForwardGroup 中多 Host 的连接策略？

- **单 Active 模式**：每个 ForwardRule 在同一时刻只通过一个 SSH Host 建立 Tunnel（Active Host）
- 其他 Host 不预建连接，不占用 SSH Session 资源
- 当 Active Host 故障（健康检查失败或连接断开），LB Pool 从 Group 中选取下一个健康 Host
- 切换期间用户连接会短暂中断（取决于健康检查频率，最长 ~15s）
- 这种设计的优势：减少 SSH Host 侧连接数压力，简化状态管理
