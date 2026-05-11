# MySQL：`forward_rules.active_host_id` 外键与 NULL 迁移说明

## 背景

历史上未绑定承载 Host 时，应用将 `active_host_id` 写为 **0**。在启用 MySQL 外键 `fk_forward_rules_active_host`（引用 `ssh_hosts.id`）时，**0** 通常不是合法父表主键，会导致 **Error 1452**。修复后，未绑定状态在数据库中为 **`NULL`**，由 GORM 模型 `*uint64` 映射。

## 上线前

1. 备份数据库（至少包含 `forward_rules`、`ssh_hosts`）。
2. 在预发或从库验证：新版本启动后 `AutoMigrate` 会将 `active_host_id` 调整为可空（以实际 DDL 为准），确认耗时与锁表现可接受。

## 部署新版本

按常规定制发布与滚动/蓝绿策略部署应用二进制。

## 上线后数据清洗（推荐执行一次）

将历史 **0** 统一改为 **`NULL`**，避免旧数据与语义不一致：

```sql
UPDATE forward_rules
SET active_host_id = NULL
WHERE active_host_id = 0;
```

若存在 `status = 'active'` 且 `active_host_id = 0` 的异常行，应先按业务评估是否将状态改为 `inactive` 或补齐正确 Host，再执行清洗。

## 验证

- 新建或复制 Rule（先 `inactive`、无承载 Host）：应成功插入，无 1452。
- 重启 Rule 且组内无健康 Host：应能清空绑定且无外键错误。
- 故障切换无备选 Host：同上。

## 回滚说明

回滚到老版本二进制时，若老版本仍向 `active_host_id` 写入 **0**，在 MySQL 外键约束下仍可能失败。若已清洗为 `NULL`，老版本对「无 Host」字段的读写行为需单独评估；一般建议出现问题时保持新版本或协调临时放宽约束（运维决策）。

## API 行为

JSON 中 `active_host_id` 在无绑定时可能为 **`null`** 或省略（`omitempty`），与历史上数字 **0** 不完全等价；依赖脚本若以 `0` 判断需改为判断缺省/`null`。
