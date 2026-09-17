# New API ClickHouse 日志表设计

本文档用于后端、数据库和运维人员设计 ClickHouse 日志库及集群。字段定义以 New API 当前 `model.Log` 和 ClickHouse 读写代码为准，表名、字段名和字段类型不能随意改名，否则应用启动时会拒绝使用该表。

## 1. 逻辑对象

| 项目 | 约定 |
| --- | --- |
| 数据库 | 默认 `logs`，也可以在 `CLICKHOUSE_DSN` 路径中指定其他合法名称 |
| 应用目标表 | `logs.logs` |
| 单机存储引擎 | `MergeTree` |
| 集群本地表 | `ReplicatedMergeTree`，每个 shard 的每个 replica 一份 |
| 集群访问表 | `Distributed`，名称仍为 `logs.logs` |
| 分区 | 按 `created_at` 的 UTC 月分区：`toYYYYMM(toDateTime(created_at, 'UTC'))` |
| 排序键 | `(created_at, id)` |
| 时间单位 | `created_at` 为 Unix 时间戳，单位为秒 |
| 可空性 | 所有业务字段均为非 Nullable，使用默认零值 |
| JSON 扩展字段 | `other` 使用 `String` 保存 JSON 文本，不使用 ClickHouse `JSON` 类型 |

应用查询和写入的是 DSN 指向的 `logs` 表。集群部署时，建议让 DSN 指向路由地址或 ClickHouse 集群中的 `Distributed` 表，不要直接指向某个 replica 的本地表。

## 2. 字段定义

| 字段 | ClickHouse 类型 | 默认值 | 来源/含义 | 写入方式 |
| --- | --- | --- | --- | --- |
| `id` | `Int64` | `0` | 日志唯一 ID。ClickHouse 写入时由 New API 生成，必须保持全局不冲突 | 应用写入 |
| `user_id` | `Int64` | `0` | 关联用户 ID | 应用写入 |
| `created_at` | `Int64` | `0` | 创建时间，Unix 秒级时间戳 | 应用写入 |
| `type` | `Int64` | `0` | 日志类型，见下方枚举 | 应用写入 |
| `content` | `String` | `''` | 日志内容 | 应用写入 |
| `username` | `String` | `''` | 用户名快照 | 应用写入 |
| `token_name` | `String` | `''` | API Token 名称快照 | 应用写入 |
| `model_name` | `String` | `''` | 请求使用的模型名称 | 应用写入 |
| `quota` | `Int64` | `0` | 本次请求消耗的额度 | 应用写入 |
| `prompt_tokens` | `Int64` | `0` | 输入 Token 数量 | 应用写入 |
| `completion_tokens` | `Int64` | `0` | 输出 Token 数量 | 应用写入 |
| `use_time` | `Int64` | `0` | 请求耗时，沿用 New API 日志模型的整数单位 | 应用写入 |
| `is_stream` | `Bool` | `false` | 是否为流式请求 | 应用写入 |
| `channel_id` | `Int64` | `0` | 渠道 ID；接口响应中通常映射为 `channel` | 应用写入 |
| `channel_name` | `String` | `''` | 渠道名称，来自 New API 业务库关联查询 | 应用不写入，查询时回填 |
| `token_id` | `Int64` | `0` | API Token ID | 应用写入 |
| `group` | `String` | `''` | 分组标识 | 应用写入 |
| `ip` | `String` | `''` | 请求 IP；用户关闭 IP 记录时为空 | 应用写入 |
| `request_id` | `String` | `''` | New API 请求 ID | 应用写入 |
| `upstream_request_id` | `String` | `''` | 上游服务请求 ID | 应用写入 |
| `other` | `String` | `''` | 扩展 JSON；包含审计、流式状态等附加信息 | 应用写入 |

### 日志类型枚举

`type` 不要使用数据库自增或字符串枚举，保持整数值：

| 值 | 常量 | 含义 |
| ---: | --- | --- |
| `0` | `LogTypeUnknown` | 未指定 |
| `1` | `LogTypeTopup` | 充值 |
| `2` | `LogTypeConsume` | 消费/请求 |
| `3` | `LogTypeManage` | 管理操作 |
| `4` | `LogTypeSystem` | 系统日志 |
| `5` | `LogTypeError` | 错误日志 |
| `6` | `LogTypeRefund` | 退款 |
| `7` | `LogTypeLogin` | 登录 |

## 3. 标准建表 DDL

### 3.1 单机或单副本

适用于开发、测试或不需要 ClickHouse 副本的部署：

```sql
CREATE DATABASE IF NOT EXISTS logs;

CREATE TABLE IF NOT EXISTS logs.logs
(
    `id` Int64 DEFAULT 0,
    `user_id` Int64 DEFAULT 0,
    `created_at` Int64 DEFAULT 0,
    `type` Int64 DEFAULT 0,
    `content` String DEFAULT '',
    `username` String DEFAULT '',
    `token_name` String DEFAULT '',
    `model_name` String DEFAULT '',
    `quota` Int64 DEFAULT 0,
    `prompt_tokens` Int64 DEFAULT 0,
    `completion_tokens` Int64 DEFAULT 0,
    `use_time` Int64 DEFAULT 0,
    `is_stream` Bool DEFAULT false,
    `channel_id` Int64 DEFAULT 0,
    `channel_name` String DEFAULT '',
    `token_id` Int64 DEFAULT 0,
    `group` String DEFAULT '',
    `ip` String DEFAULT '',
    `request_id` String DEFAULT '',
    `upstream_request_id` String DEFAULT '',
    `other` String DEFAULT ''
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(toDateTime(created_at, 'UTC'))
ORDER BY (created_at, id);
```

`is_stream` 也可以使用 `UInt8 DEFAULT 0`；当前应用兼容 `Bool` 和 `UInt8`。集群新建表建议优先使用上面的 `Bool` 定义。

### 3.2 集群本地表和 Distributed 表

以下示例假设：

- ClickHouse 集群名为 `newapi_cluster`；
- 集群配置中已经声明 shard、replica 和 Keeper/ZooKeeper；
- 数据库名为 `logs`；
- 每个 shard 有一个或多个 replica；
- `{shard}`、`{replica}` 是 ClickHouse 集群宏，且每个 replica 的值唯一。

先在所有节点创建数据库和本地复制表：

```sql
CREATE DATABASE IF NOT EXISTS logs ON CLUSTER newapi_cluster;

CREATE TABLE IF NOT EXISTS logs.logs_local ON CLUSTER newapi_cluster
(
    `id` Int64 DEFAULT 0,
    `user_id` Int64 DEFAULT 0,
    `created_at` Int64 DEFAULT 0,
    `type` Int64 DEFAULT 0,
    `content` String DEFAULT '',
    `username` String DEFAULT '',
    `token_name` String DEFAULT '',
    `model_name` String DEFAULT '',
    `quota` Int64 DEFAULT 0,
    `prompt_tokens` Int64 DEFAULT 0,
    `completion_tokens` Int64 DEFAULT 0,
    `use_time` Int64 DEFAULT 0,
    `is_stream` Bool DEFAULT false,
    `channel_id` Int64 DEFAULT 0,
    `channel_name` String DEFAULT '',
    `token_id` Int64 DEFAULT 0,
    `group` String DEFAULT '',
    `ip` String DEFAULT '',
    `request_id` String DEFAULT '',
    `upstream_request_id` String DEFAULT '',
    `other` String DEFAULT ''
)
ENGINE = ReplicatedMergeTree(
    '/clickhouse/tables/{shard}/logs/logs_local',
    '{replica}'
)
PARTITION BY toYYYYMM(toDateTime(created_at, 'UTC'))
ORDER BY (created_at, id);
```

再创建供 New API 连接的 Distributed 入口表：

```sql
CREATE TABLE IF NOT EXISTS logs.logs ON CLUSTER newapi_cluster
AS logs.logs_local
ENGINE = Distributed(
    'newapi_cluster',
    'logs',
    'logs_local',
    cityHash64(id)
);
```

如果一个 shard 内有多个 replica，`Distributed` 表会将数据写入 shard；副本同步由 `ReplicatedMergeTree` 负责。生产环境必须确认 Keeper/ZooKeeper 可用，否则复制表可能无法正常提交副本日志。

> 若集群已经存在 `logs.logs`，不要直接执行会改变结构的 `CREATE OR REPLACE` 或 `ALTER`。先用 `SHOW CREATE TABLE logs.logs`、`system.columns` 和 `system.tables` 对照本文档，确认现有表兼容后再接入应用。New API 启动时只检查兼容性，不会自动修改已有表。

## 4. New API 连接配置

应用应连接 `Distributed` 表所在的数据库入口：

```yaml
environment:
  CLICKHOUSE_DSN: clickhouse://newapi:<password>@clickhouse-router:9000/logs
  CLICKHOUSE_NODE_ID: "0"
  NODE_NAME: sh-prod-api-01
  NODE_HOST_IP: 10.10.20.31
```

集群中每个同时写入同一 ClickHouse 日志库的 New API 实例，必须使用不同的 `CLICKHOUSE_NODE_ID`，取值范围为 `0–1023`。该值是 New API 日志 ID 生成器的写入节点编号，不是 ClickHouse 的 shard、replica 或 server ID，也不是 ClickHouse 表字段。

应用连接时会启用以下写入语义：

- `async_insert=1`，并设置 `wait_for_async_insert=1`，服务端异步批处理完成后才向应用确认；
- `insert_distributed_sync=1`，等待 Distributed 表转发到目标 shard；
- `skip_unavailable_shards=0`，任一目标 shard 不可用时返回失败，触发 New API 的备用日志库降级；
- 应用默认连接端口为原生协议 `9000`，也支持 HTTP/HTTPS DSN。

## 5. 集群设计约束

1. **不能删除或重命名字段。** 应用按固定字段列表写入和读取；缺字段或类型不兼容会导致 ClickHouse 日志初始化失败。
2. **不要把 `channel_name` 设为必填业务字段。** 它由 New API 从业务数据库补齐，ClickHouse 中允许保持空字符串。
3. **不要使用 `Nullable` 代替默认值。** 当前应用按非 Nullable 字段扫描，历史兼容检查虽然允许部分 Nullable 包装类型，但新建表应按本文档的非 Nullable 结构创建。
4. **不要将 `other` 改成 ClickHouse `JSON` 类型。** 应用保存的是 JSON 字符串，且该字段需要保留历史格式和未知扩展键。
5. **排序键必须包含 `created_at, id`。** 日志列表、导出、统计和删除都依赖时间及 ID 顺序；更换排序键会明显影响查询性能。
6. **分区按月即可。** 不建议按天或按用户分区，避免产生过多小分区和元数据。
7. **`id` 必须全局不冲突。** New API 使用时间、节点编号和序列生成 ID；不要在多实例部署中复用同一个 `CLICKHOUSE_NODE_ID`。
8. **应用故障降级不会自动回灌。** ClickHouse 故障期间新增日志会写入原 MySQL/PostgreSQL/SQLite 日志表；恢复后新日志回到 ClickHouse，但历史备用库日志不会自动迁移。后续迁移必须按 `request_id`、时间、业务字段核对并去重，不能仅凭两个库的 `id` 判断同一条记录。

## 6. 上线前检查

```sql
SHOW CREATE TABLE logs.logs;

SELECT name, type, default_kind, default_expression
FROM system.columns
WHERE database = 'logs' AND table = 'logs'
ORDER BY position;

SELECT database, name, engine
FROM system.tables
WHERE database = 'logs' AND name IN ('logs', 'logs_local');
```

上线前至少确认：

- `logs.logs` 的引擎为 `MergeTree`、`ReplicatedMergeTree` 或 `Distributed`；
- 目标表包含本文档的全部 21 个字段；
- `logs.logs` 为 Distributed 时，其底层表为所有 shard 上的 `logs.logs_local`；
- New API 使用的账号具备目标数据库的 `SELECT`、`INSERT` 权限，以及初始化阶段所需的数据库/表检查权限；
- 所有 New API 实例的 `CLICKHOUSE_NODE_ID` 唯一；
- 先写入一条测试日志，再验证 Distributed 表查询、replica 同步和故障时分片不可用返回错误；
- 应用连接的 DSN 不要包含明文生产密码提交到代码仓库或文档外部系统。

