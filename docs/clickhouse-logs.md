# New API ClickHouse 日志接入

`CLICKHOUSE_DSN` 为空时，保持原来的 `LOG_SQL_DSN` / 主库日志路径。
配置后，Go 后端直接连接 ClickHouse，正常情况下日志写入、明细、统计数据读取及删除均使用 ClickHouse；写入故障时自动降级到原日志库的 `logs` 表。
主库仍可以是 PostgreSQL、MySQL 或 SQLite，费用、利润、缓存命中和趋势处理继续在 Go 后端完成。

## 本地 Compose 启动

现有 Compose 默认 PostgreSQL 配置保持不变。ClickHouse 是可选的 `clickhouse` profile，未启用时不会启动该容器。

账号密码直接填写在 `docker-compose.yml` 的 `clickhouse.environment` 中，当前默认值为：

```yaml
CLICKHOUSE_USER: newapi
CLICKHOUSE_PASSWORD: "12345678"
CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT: "1"
```

无需额外设置 `.env`。启用时，在 `new-api.environment` 中注释 `CLICKHOUSE_DSN=` 空值行，取消下面这行的注释：

```yaml
- CLICKHOUSE_DSN=clickhouse://newapi:12345678@clickhouse:9000/logs
- CLICKHOUSE_NODE_ID=0
```

修改账号密码时，同时修改连接地址中的对应值；密码包含 URL 特殊字符时，需要在 DSN 中进行 URL 编码。Compose 内置密码仅供本地测试。健康检查读取的是本服务上方配置的账号密码，不需要外部环境变量。

```sh
docker compose --profile clickhouse up -d --wait clickhouse
docker compose up -d --build new-api
```

也可以连接已有 ClickHouse，直接配置 `CLICKHOUSE_DSN`，无需启动本地 ClickHouse 容器。
支持 `clickhouse://host:9000/database` 和 HTTP(S) 连接；用户名和密码按示例放入 URL。
地址需要显式端口。数据库名使用字母、数字和下划线，首字符为字母或下划线；省略库名时使用 `logs`。

默认初始化 MergeTree 日志表，也可连接由外部团队预先提供的同字段 Distributed 表。不会部署或管理分片、副本和 Keeper。Distributed 表的历史删除需由 ClickHouse 服务端操作底层表，当前应用删除接口会明确报错。

## 启动初始化

1. 先按原逻辑初始化主库及原日志库，保留现有 `logs` 表和连接；原库的初始化、迁移方式沿用项目现有逻辑。
2. `CLICKHOUSE_DSN` 去除首尾空白后为空，继续原日志数据库逻辑。
3. 有配置时连接 ClickHouse，检查目标库及 `logs` 表。
4. 只创建不存在的库和表；已有表仅检查字段兼容性，不执行 AutoMigrate、不改字段、不清数据。
5. ClickHouse 无法连接或表暂不可用时，原日志库可用则允许启动并进入降级写入；后续写入会重试 ClickHouse 初始化。两边都不可用则返回错误。原主库仍是应用启动所必需的依赖。

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
对已有 Distributed 入口同时设置 `insert_distributed_sync=1`、`skip_unavailable_shards=0`，等待分片转发结果；上游返回的写入失败可触发原库降级。
排查时可在 DSN 加 `?async_insert=0` 使用同步插入。
启用 ClickHouse 后，运行中的写入失败会自动触发下述降级逻辑，无需增加灾备配置。

## 第二层灾备：复用原日志库

正常日志优先写 ClickHouse。ClickHouse 故障时，直接写回启用 ClickHouse 之前使用的 MySQL/PostgreSQL 的 `logs` 表。复用原来的 `LOG_DB` 连接，不增加备份数据库、容器、数据卷或连接参数；原 SQLite 部署也沿用相同逻辑。Go 不管理 ClickHouse 集群的分片和副本。

原日志库的选择规则保持不变：

| 已有配置 | ClickHouse 故障时写入位置 |
| --- | --- |
| MySQL 主库，未配置 `LOG_SQL_DSN` | 该 MySQL 主库原有的 `logs` 表 |
| PostgreSQL 主库，未配置 `LOG_SQL_DSN` | 该 PostgreSQL 主库原有的 `logs` 表 |
| 已配置 `LOG_SQL_DSN` | 该配置对应的原日志库的 `logs` 表 |
| SQLite，未配置 `LOG_SQL_DSN` | 原 SQLite 数据库的 `logs` 表 |

现有 Compose 默认使用 PostgreSQL，因此只需启用上面的 ClickHouse 连接，故障时就会自动写回现有 `postgres` 服务的 `logs` 表。使用现有 MySQL 配置时同理。

运行行为：

- 正常写 ClickHouse 成功，只保存到 ClickHouse。
- ClickHouse 写入失败或超时，立即尝试原日志库；两次写入分别使用 5 秒超时，原库写入不受前一次超时取消影响。
- 进入降级后，5 秒窗口内的新日志直接写原日志库，避免每条日志都等待不可用的 ClickHouse。
- 窗口结束后的首条日志尝试 ClickHouse，成功则后续新日志恢复写入 ClickHouse；失败则继续使用原日志库。无新日志时不会额外执行后台探活。
- 启动时原库初始化成功、ClickHouse 不可用，同样进入降级模式。原日志表之后暂不可用而 ClickHouse 可用时，继续写 ClickHouse 并记录原日志表不可用信息。
- 两个存储都写入失败时返回错误，并由现有日志调用方报告；不会把失败当作成功。

查询和统计仍读取 ClickHouse。原库只保留原有历史和降级期间日志，无法代表启用 ClickHouse 后的完整记录，因此不会自动把查询切回原库。故障期间日志查询可以失败，但降级写入继续保存日志。

降级写入保留原业务字段和 `other`，使用原库自身的 ID 生成方式，避免与原有历史冲突。恢复后不自动回灌：ClickHouse 超时可能发生在服务端已经提交之后，此时两边可能都存在该条日志。降级日志继续留存在原库，后续补迁需要核对内容、去重并安排目标 ID；不能直接把两个库的 `id` 当作同一标识。库表字段没有新增。

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

测试覆盖缺库缺表初始化、已有表保持原样、全字段往返、历史 ID 后续生成、空用户范围、统计对账，以及超过 1,000 条日志的导出和删除。测试服务器具有 `default` 本地集群配置时，还验证已有 Distributed 表的读写及表定义保留。
可额外设置 `TEST_LOG_PRIMARY_DSN` 为 PostgreSQL 或 MySQL 的独立 `new_api_test` 数据库，验证关系库与 ClickHouse 返回的一致性；该测试会创建并清理该库中的三个测试表，勿指向业务数据库。

灾备集成测试使用同一组测试参数，通过测试 TCP 代理中断 ClickHouse 连接，验证原 `logs` 表写入、原历史保留、运行中恢复、启动时故障和两边同时失败。分别使用 MySQL、PostgreSQL 的测试 DSN 运行：

```sh
TEST_CLICKHOUSE_DSN='clickhouse://newapi:12345678@127.0.0.1:9000/default' \
TEST_LOG_PRIMARY_DSN='postgresql://user:password@127.0.0.1:5432/new_api_test' \
  go test ./model -run 'TestClickHouseRuntimeOriginal|TestClickHouseStartup|TestClickHouseClose|TestInitLogDB' -count=1
```
