# ClickHouse 日志访问统一规范需求

## 1. 文档目的

本文档用于统一 New API 中日志查询、日志聚合和日志统计的数据访问方式，解决 ClickHouse 接入后新增接口需要重复编写 ClickHouse/原日志库分支的问题。

本文档作为本阶段的实施和验收依据；源码修改必须遵守以下范围与约束。

## 2. 已确认的产品决策

### 2.1 不增加统一的手工切库开关

不增加“强制使用 ClickHouse”“强制使用 MySQL/原日志库”之类的后台开关。数据源继续由系统自动判断：

```text
ClickHouse 已配置且健康
    → 使用 ClickHouse

ClickHouse 读取失败
    → 当前实例进入读取降级，使用原日志库

后台探针确认 ClickHouse 恢复
    → 后续请求重新使用 ClickHouse
```

写入和读取继续保持独立状态。一次写入失败只影响写入链路，一次读取失败只影响读取链路，不能用一个全局状态互相覆盖。

### 2.2 自动探测仍是唯一恢复机制

业务明细、费用统计、利润统计和其他聚合请求不能充当 ClickHouse 恢复探针。恢复只能由后台探测任务完成，探测间隔和超时继续使用“故障处理”配置。

### 2.3 备用库不是完整历史库

ClickHouse 故障期间写入原日志库的记录只代表降级期间新增数据，原日志库不保证包含 ClickHouse 中的全部历史记录。统一访问层不得将两个库的结果自动拼接成一个“完整历史”。

## 3. 当前实现问题

当前已有以下基础能力：

- `createLog` 统一处理日志写入和写入故障降级；
- `clickHouseLogStore.readLogs` 统一处理一次读取中的 ClickHouse/原日志库回退；
- 日志列表、费用统计、利润统计、渠道趋势、Token 趋势和 Token Cache 统计已经接入部分 ClickHouse 读取逻辑；
- `LogReadState` 可以让同一次多步统计固定使用同一个数据源。

但上层 Model 仍然存在多处：

```go
if clickHouseLogs != nil {
    // ClickHouse 查询
}

// 原日志库 SQL 查询
```

这会带来以下问题：

1. 新增日志聚合接口时，开发人员容易遗漏 fallback、降级状态和探针规则。
2. 直接使用 `LOG_DB.Table("logs")` 的新接口会绕过 ClickHouse。
3. 直接使用 `clickHouseLogs.Scan` 的新接口可能绕过原日志库回退。
4. 不同接口可能使用不同的排序、分页、过滤和数据源一致性规则。
5. Controller 需要了解 `X-Log-Source`、`LogReadState` 等底层细节的使用约定。
6. `quota_data` 等原有汇总表与 `logs` 表不是同一个数据源体系，容易被误认为已经完成 ClickHouse 切换。

## 4. 目标架构

### 4.1 统一日志读取入口

在 Model 层提供一个统一的日志访问边界，业务统计函数只描述过滤条件和聚合逻辑，不再自行判断 ClickHouse 或原日志库。

本需求确定采用以下三类最小能力，名称可以在实现阶段按现有代码风格调整：

```go
// 分页查询，供日志明细和列表接口使用。
QueryLogs(ctx, filter, offset, limit, order)

// 流式遍历，供费用、利润、趋势和其他聚合接口使用。
ForEachLogs(ctx, filter, visit)

// 查询聚合所需的去重 ID，例如渠道 ID。
ListLogChannelIDs(ctx, filter)
```

这些入口内部统一负责：

- ClickHouse 是否已配置；
- 当前读取降级状态；
- ClickHouse 读取失败后的原日志库回退；
- 后台探针恢复后的自动切回；
- 一次请求内的数据源固定；
- ClickHouse 部分读取失败时禁止与原库结果混合；
- 原库查询失败时的统一错误处理和服务端日志记录。

本阶段不增加通用 `SUM`、`COUNT`、`GROUP BY` 聚合 API。费用、利润、趋势等业务聚合继续在 Go 中通过流式遍历完成。只有经过实际性能测量确认某一类聚合存在瓶颈时，才为该场景增加明确的专用聚合能力，不提前建设通用查询 DSL。

业务代码不应直接依赖 `clickHouseLogs` 或直接为 `logs` 表编写第二套数据源判断。

### 4.2 统一写入入口

继续使用现有 `createLog(ctx, log)` 作为所有日志写入函数的唯一入口。新增日志类型或新增日志记录场景必须调用该入口，不得直接调用 `LOG_DB.Create` 或 ClickHouse 原生插入。

写入入口内部继续保持：

- ClickHouse 正常时优先写 ClickHouse；
- ClickHouse 写入失败时写入原日志库；
- 进入降级后按现有重试间隔恢复尝试；
- 恢复后新日志写入 ClickHouse；
- 不自动回灌降级期间的原库日志。

### 4.3 统一过滤模型

统一入口应使用结构化过滤条件，不允许新接口拼接用户输入形成 SQL。过滤模型至少覆盖当前接口已经支持的条件：

- 日志类型；
- 用户 ID 或用户 ID 集合；
- Token ID、Token 名称；
- 用户名；
- 模型名称及显式模糊匹配；
- 时间范围；
- 渠道 ID；
- 分组；
- 请求 ID、上游请求 ID；
- 流式错误状态；
- 非零渠道、非零用户和存在 Token 等聚合筛选条件。

ClickHouse 和 MySQL/PostgreSQL/SQLite 的过滤实现可以不同，但对外必须保持相同的过滤语义、权限范围和空集合语义。

### 4.4 统一请求数据源状态

每个 Controller 请求最多创建一个 `LogReadState`，传递给该请求涉及的所有日志访问操作。统一入口负责维护该状态。

要求：

- 请求首次读取 ClickHouse 成功时，后续同一请求继续使用 ClickHouse；
- 请求首次回退原日志库后，后续同一请求继续使用原日志库；
- 同一次费用或利润统计不得前半段读取 ClickHouse、后半段读取原库；
- ClickHouse 流式扫描已经向调用方交付部分记录后发生错误时，不得自动改用原库继续累计；
- Controller 继续通过 `X-Log-Source` 返回本次结果的实际来源；
- 统一入口应保留调用方的取消和超时上下文，不能用新的 `context.Background()` 丢失请求取消信号。

### 4.5 统一聚合扩展方式

未来新增日志聚合接口时，开发人员只需要完成三部分：

1. 定义业务过滤条件到统一日志过滤模型的映射；
2. 在 `ForEachLogs` 回调中实现聚合；
3. 将聚合结果转换为接口响应。

新增接口不需要重复实现：

- `if clickHouseLogs != nil` 分支；
- `LOG_DB` fallback 查询；
- 读取降级状态更新；
- 恢复探测；
- 告警触发；
- 结果来源响应头；
- 多步骤统计的数据源锁定。

推荐的业务代码形态：

```go
func GetNewLogStatistics(query NewLogStatisticsQuery) (..., error) {
    result := newAccumulator()
    err := ForEachLogs(query.Context, newLogFilter(query), func(log Log) error {
        result.Add(log)
        return nil
    })
    if err != nil {
        return ..., err
    }
    return result.Result(), nil
}
```

示例只表达访问边界，实际实现应沿用项目现有错误、权限和上下文约定。

## 5. 现有接口迁移范围

统一规范实施后，应检查并迁移所有直接访问日志表的业务路径：

| 业务范围 | 目标 |
| --- | --- |
| 管理员、用户、托管用户日志列表 | 统一使用分页查询入口 |
| Token 日志明细 | 统一使用分页查询入口 |
| 日志额度、RPM、TPM 统计 | 统一使用流式遍历入口 |
| 费用统计 | 统一使用流式遍历入口 |
| 利润统计 | 统一使用流式遍历入口 |
| 渠道、Token、用户/Token 趋势 | 统一使用流式遍历入口 |
| Token Cache 统计 | 统一使用流式遍历入口 |
| 其他基于 `logs` 表的新增或历史接口 | 禁止绕过统一入口 |
| 日志删除 | 保持独立删除策略，不纳入普通读取入口 |

### `quota_data` 的边界

`quota_data` 是原有关系库中的独立汇总表，不属于 ClickHouse `logs` 表统一访问范围。实施时必须明确标注：

- 读取 `quota_data` 的接口继续使用原关系库；
- 不得在接口文档中声称这些接口已经具备 ClickHouse 日志切换能力；
- `quota_data` 不迁移、不改表结构、不改变现有缓存聚合和定时落库方式；
- 如果未来需要让 `quota_data` 也具备 ClickHouse 能力，应另立数据模型和迁移需求，不能在本需求中隐式扩展。

当前 `/api/data` 路由下的数据源并不完全相同，实施和文档必须按实际来源区分：

| 接口范围 | 当前数据源 | 本需求处理方式 |
| --- | --- | --- |
| `/api/data/`、`/api/data/users`、`/api/data/self`、`/api/data/managed` | 主关系库中的 `quota_data` | 保持不变 |
| `/api/data/channels`、`/api/data/tokens/self`、`/api/data/tokens/users` | `logs` 日志数据源 | 迁移到统一日志读取入口 |
| `/api/data/token_cache_hits*` | `logs` 日志数据源 | 迁移到统一日志读取入口 |

`quota_data` 按“用户、用户名、模型、小时”聚合，保存请求数、额度和 Token 总量，不按每次请求写一条明细，因此通常远小于 `logs` 表。它属于主数据库 `DB`，不属于独立数据库，也不跟随 `LOG_SQL_DSN`。

## 6. 代码约束

### 必须遵守

- 新增日志写入必须调用统一写入入口。
- 新增基于 `logs` 表的读取或聚合必须调用统一读取入口。
- 统一入口必须使用参数化查询和结构化过滤条件。
- 统一入口必须保留请求上下文的取消和超时。
- 同一请求中的多步聚合必须共享一个读取状态。
- ClickHouse 和原日志库的结果不能在同一次统计中无条件拼接。
- 新增接口必须覆盖 ClickHouse 正常、ClickHouse 故障、后台恢复三个场景。

### 禁止事项

- 不新增后台手工强制切换数据源开关。
- 不在 Controller 中复制 ClickHouse/原库切换判断。
- 不在业务聚合函数中直接访问 `clickHouseLogs`。
- 不在业务聚合函数中直接对 `LOG_DB` 的 `logs` 表写第二套 fallback。
- 不让明细、统计或导出请求充当 ClickHouse 恢复探针。
- 不把备用库结果包装成完整 ClickHouse 历史数据。
- 不自动把降级期间的原库日志回灌到 ClickHouse。

## 7. 实施步骤

实施时按以下顺序进行，每一步都应保持项目可编译、可测试：

1. 固化统一读取入口的参数、返回值、错误和数据源状态约定。
2. 将现有 `Query`、`Scan`、`ChannelIDs` 的公共行为收敛到统一入口，保留现有自动探测和 fallback 逻辑。
3. 迁移日志列表、费用、利润、趋势和 Token Cache 等现有调用方。
4. 清理业务层新增的直接 `LOG_DB`/`clickHouseLogs` 日志访问，只保留统一入口内部的数据库实现。
5. 为统一入口补充单元测试和故障集成测试，覆盖来源固定、部分扫描失败不混合、请求取消和恢复探测。
6. 在开发规范和 ClickHouse 文档中增加“新增日志接口必须使用统一入口”的约束。

本需求不要求重做 ClickHouse 表结构、ID 生成、写入降级、企业微信告警、后台故障处理页面或日志导出格式。

## 8. 验收标准

### 功能验收

- ClickHouse 健康时，所有已迁移的日志列表和聚合接口读取 ClickHouse。
- ClickHouse 读取故障时，所有已迁移接口自动读取原日志库。
- ClickHouse 故障期间，业务请求不会因重复探测而产生额外等待。
- 后台探测确认恢复后，后续请求自动回到 ClickHouse。
- 同一个费用、利润或趋势请求全程使用一个数据源。
- 新增一个仅包含业务聚合逻辑的测试接口时，不需要增加第二套数据库切换代码。

### 质量验收

- 业务 Model 中不再出现新的 `if clickHouseLogs != nil` 日志访问分支。
- 业务 Model 中不再出现新的 `LOG_DB.Table("logs")` 或 `LOG_DB.Model(&Log{})` 直连日志查询。
- 统一入口测试覆盖 ClickHouse、原日志库、故障回退和恢复状态。
- 现有日志、费用、利润、趋势和导出测试保持通过。
- ClickHouse 未配置时，统一入口仍能正常使用原有 MySQL、PostgreSQL 或 SQLite 日志库。

## 9. 非目标和后续议题

以下事项不包含在本需求中：

- 增加后台手工强制切库开关；
- 将多个 New API 实例的读写故障状态做成共享状态；
- 将 ClickHouse 历史数据和原日志库数据自动合并；
- 自动回灌和去重迁移降级期间日志；
- 将 `quota_data` 表迁移到 ClickHouse；
- 重新设计 ClickHouse 分片、副本或 Keeper 架构；
- 为所有历史非日志业务表建立统一数据源抽象。

## 10. 已确认实施决策

本需求已确认：

1. 统一入口采用“分页查询 + 流式遍历 + 渠道 ID 查询”三类最小 API，不增加通用聚合 API。
2. `quota_data` 继续保留在原主关系库，不修改、不迁移到 ClickHouse。
