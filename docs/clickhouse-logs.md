# New API ClickHouse 日志接入

`CLICKHOUSE_DSN` 为空时，保持原来的 `LOG_SQL_DSN` / 主库日志路径。
配置后，Go 后端直接连接 ClickHouse，日志写入、明细、统计数据读取及删除使用同一日志库。
主库仍可以是 PostgreSQL、MySQL 或 SQLite，费用、利润、缓存命中和趋势处理继续在 Go 后端完成。

## 本地 Compose 启动

现有 Compose 默认 PostgreSQL 配置保持不变。ClickHouse 是可选的 `clickhouse` profile，未启用时不会启动该容器。

在部署 `.env` 中配置匹配的用户、密码和连接地址，例如：

```dotenv
CLICKHOUSE_USER=newapi
CLICKHOUSE_PASSWORD=replace-with-your-password
CLICKHOUSE_DSN=clickhouse://newapi:replace-with-your-password@clickhouse:9000/logs
CLICKHOUSE_NODE_ID=0
```

密码包含 URL 特殊字符时，需要在 DSN 中进行 URL 编码。Compose 内置密码仅供本地测试。

```sh
docker compose --profile clickhouse up -d --wait clickhouse
docker compose up -d --build new-api
```

也可以连接已有 ClickHouse，直接配置 `CLICKHOUSE_DSN`，无需启动本地 ClickHouse 容器。
支持 `clickhouse://host:9000/database` 和 HTTP(S) 连接；用户名和密码按示例放入 URL。
地址需要显式端口。数据库名使用字母、数字和下划线，首字符为字母或下划线；省略库名时使用 `logs`。

本次实现聚焦 MergeTree 日志表接入。不会部署或管理分片、副本、Keeper 和 Distributed 表。

## 启动初始化

1. 先按原逻辑初始化主库。
2. `CLICKHOUSE_DSN` 去除首尾空白后为空，继续原日志数据库逻辑。
3. 有配置时连接 ClickHouse，检查目标库及 `logs` 表。
4. 只创建不存在的库和表；已有表仅检查字段兼容性，不执行 AutoMigrate、不改字段、不清数据。
5. 连接失败或已有表不兼容，返回明确初始化错误，不静默改用其他日志库。

自动创建的表使用 `MergeTree`，按 UTC 月分区，按 `(created_at, id)` 排序。
字段与 `model.Log` 对应：

```text
id user_id created_at type content username token_name model_name quota
prompt_tokens completion_tokens use_time is_stream channel_id channel_name
token_id group ip request_id upstream_request_id other
```

整数列默认创建为 `Int64`，`is_stream` 为 `Bool`，文本为 `String`，使用原有零值默认值。
`other` 保存原始 JSON 字符串。`channel_name` 保持原来的只读/名称回填方式，写入不覆盖该字段。
已有表的整数列允许兼容整数类型，`id` 必须为 `Int64` 或 `UInt64`；缺列或类型不兼容会报错。

## 日志 ID 与写入

MySQL/PostgreSQL 默认路径仍使用原来的 ID 生成方式。
ClickHouse 写入使用原有 `id` 字段生成整数 ID，限制在 JavaScript 可精确表示的 53 位范围内。
启动时读取已有最大 ID，避免与导入的历史记录、同一实例重启后的 ID 冲突。

每个同时向同一日志库写入的 New API 实例需要不同的 `CLICKHOUSE_NODE_ID`（0–1023）。
单实例默认使用 0。每个实例每秒支持 4,096 个 ID，达到上限后等待下一秒；不会回绕生成重复 ID。
这是 New API 写入实例的标识，不是数据库表字段。

默认启用 ClickHouse 服务端异步插入批处理，但 `wait_for_async_insert=1`，Go 写入在服务端完成后才报告成功。
排查时可在 DSN 加 `?async_insert=0` 使用同步插入。
写入失败沿用现有日志错误处理，没有新增应用侧双写或持久重试队列。

## 查询及统计

- 日志明细保持现有分页、筛选和 JSON 字段；数据库 `channel_id` 对应明细响应 `channel`。
- 费用、利润通过流式读取完整匹配记录继续执行原 Go 聚合，导出不会被明细分页截断。
- 日志 RPM/TPM 保持独立最近 60 秒口径；渠道和令牌趋势在 Go 中形成原有小时结果。
- 删除按选中的 ID 分批执行，并等待删除可见后返回条数。
- 文本 `LIKE` 使用 ClickHouse 的大小写敏感语义；原关系库的列排序规则不会随数据导入自动转移。`%`、`_` 和反斜杠按既有接口的显式模糊/字面包含规则转义。

## 验证

普通测试不需要 ClickHouse：

```sh
go test ./model ./service
go build ./...
docker compose config --quiet
docker compose --profile clickhouse config --quiet
```

真实 ClickHouse 测试在目标服务上创建独立的 `new_api_test_*` 数据库并在结束时删除这些测试数据库：

```sh
TEST_CLICKHOUSE_DSN='clickhouse://user:password@127.0.0.1:9000/default' \
  go test ./model -run 'TestClickHouse|TestInitLogDB' -count=1
```

测试覆盖缺库缺表初始化、已有表保持原样、全字段往返、历史 ID 后续生成、空用户范围、统计对账，以及超过 1,000 条日志的导出和删除。
可额外设置 `TEST_LOG_PRIMARY_DSN` 为 PostgreSQL 或 MySQL 的独立 `new_api_test` 数据库，验证关系库与 ClickHouse 返回的一致性；该测试会创建并清理该库中的三个测试表，勿指向业务数据库。
