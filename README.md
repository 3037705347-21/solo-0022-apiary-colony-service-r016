# 蜂场蜂群协作服务

本服务帮助蜂场值守人与片区协调人协作录入蜂场与蜂箱、提交现场巡检、更换蜂王、
推进蜂群状态，并读取单个蜂场的状态汇总。服务是一个 Go 后端，全部数据保存在进程内存中。

## 目录

- `cmd/server`：HTTP 服务入口。
- `cmd/check`：按 workflow 执行真实 HTTP 路径的生产校验入口。
- `internal/domain`：实体、枚举、校验与蜂群状态迁移表。
- `internal/store`：并发安全的内存仓储与一致性快照。
- `internal/apiary`：蜂场与蜂箱录入、蜂箱状态迁移。
- `internal/inspection`：巡检接收与关注项判定。
- `internal/queen`：蜂王更换与标记色推导。
- `internal/status`：蜂场状态汇总聚合。
- `internal/httpapi`：路由、请求解析与响应映射。
- `internal/app`：应用组装与配置。

## 构建与运行

```bash
go build ./...
go run ./cmd/server
```

服务默认监听 `127.0.0.1:18093`，可用环境变量 `APIARY_ADDR` 覆盖监听地址。

```bash
APIARY_ADDR=127.0.0.1:18094 go run ./cmd/server
```

健康检查与只读入口：

```bash
go run ./cmd/server
# 另开一个终端
go run ./cmd/check -workflow read-apiary-status
```

## 接口与输入输出

- `GET /healthz`：返回 `{"status":"healthy"}`。
- `POST /apiaries`：输入 `id`、`name`、`district`、`keeper`，成功返回 201 与蜂场。
- `GET /apiaries/{apiaryId}`：读回蜂场。
- `POST /apiaries/{apiaryId}/hives`：输入 `id`、`model`（`langstroth`、`top_bar`、`warre`），
  成功返回 201 与状态为 `colonized` 的蜂箱。
- `GET /hives/{hiveId}`：读回蜂箱。
- `POST /hives/{hiveId}/state`：输入 `target`，返回迁移前后的状态。
- `POST /hives/{hiveId}/inspections`：输入 `observedAt`（可选，RFC3339）、`colonyStrength`、
  `strengthUnit`、`broodPattern`、`temper`、`miteCount`、`miteUnit`、`note`，
  服务在接收时把量测换算到规范口径（群势 `grade`、螨计数 `per100`）后保存并派生关注项，
  返回内容同时包含原始值/原始单位与规范值/规范单位及 `rulesVersion`。
- `GET /hives/{hiveId}/inspections`：列出巡检。
- `POST /hives/{hiveId}/queens`：输入 `origin`、`introducedAt`（可选）、`reason`，
  返回带标记色的新蜂王记录。
- `GET /hives/{hiveId}/queens`：列出蜂王记录。
- `GET /apiaries/{apiaryId}/status`：可选查询参数 `asOf`（RFC3339），返回状态计数、
  待巡检蜂箱、有未处理关注项的蜂箱与逐蜂箱摘要；摘要中的最近量测使用与巡检相同的
  规范口径（`latestMeasurements`：`grade` / `per100`）。

错误响应统一为：

```json
{"error": {"code": "invalid_request", "message": "说明", "field": "可选字段名"}}
```

错误分类：`invalid_request`、`not_found`、`conflict`、`invalid_state`、`storage_error`。

## 量测单位与口径

巡检的群势与螨计数必须携带单位，不同岗位可用不同单位记录，服务在接收时统一换算：

| 维度 | 规范单位 | 兼容单位（原始范围） | 换算 |
| --- | --- | --- | --- |
| 群势 | `grade`（1..5 级） | `frame`（1..10 满框数） | `grade = round(frame / 2)` |
| 螨计数 | `per100`（每百只 0..60） | `per300`（0..180，三百只抽样） | `per100 = round(per300 / 3)` |

- 单位缺失、未知或跨维度不兼容（例如群势带 `per100`、螨带 `grade`）一律 400 拒绝。
- 原始值与换算后的规范值都会做范围校验。
- 换算只在接收时执行一次，巡检记录固化原始值、规范值与 `rulesVersion`（当前 `v1`）；
  汇总与关注项只读固化结果，规则将来升级不会追溯改写历史巡检与关注项。
- 同一蜂箱的巡检与状态汇总呈现始终使用同一规范口径。

## 业务规则摘要

- 蜂群状态：`colonized` 可到 `production`，`production` 与 `wintering` 可互相切换，
  三者都可到终态 `retired`；迁移到自身视为冲突。
- 已退役蜂箱不再接受巡检与换王。
- 同一蜂箱任意时刻最多一只 `laying` 蜂王，换王会先关闭旧记录。
- 蜂王标记色由引入年份个位推导：`1/6` 白、`2/7` 黄、`3/8` 红、`4/9` 绿、`5/0` 蓝。
- 巡检关注项阈值（均基于换算后的规范口径）：螨计数（每百只）≥ 9 记 `mite_load`，
  子脾为 `none` 记 `brood_gap`，性情为 `aggressive` 记 `temper_risk`，
  群势（级）≤ 2 记 `low_strength`。
- 待巡检：从未巡检，或距 `asOf` 超过 14 天。

## 环境变量

- `APIARY_ADDR`：HTTP 监听地址，默认 `127.0.0.1:18093`。

## 校验

每个 workflow 都有可执行的生产校验入口，均走真实 HTTP 请求：

```bash
go run ./cmd/check -workflow register-apiary
go run ./cmd/check -workflow install-hive
go run ./cmd/check -workflow record-inspection
go run ./cmd/check -workflow replace-queen
go run ./cmd/check -workflow advance-hive-state
go run ./cmd/check -workflow read-apiary-status
```

## 测试边界

本项目按初始化基线交付，**测试刻意推迟**：不包含单元测试、测试夹具或浏览器端校验文件，
`.project-runtime.json` 也不声明 `test_command`。后续工程任务阶段会在保留现有行为的前提下
补充红绿验证用例；当前以 `cmd/check` 的 workflow 校验入口作为可执行的生产路径证明。
