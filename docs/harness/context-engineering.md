# 上下文工程（Context Engineering）

> 目标：让 Agent "知道该知道的信息"——确保 Agent 在任务执行中能获取准确、及时、适量的上下文。

## 1. 知识来源定义

### 1.1 唯一知识来源（Single Source of Truth）

| 知识类型 | 存储位置 | 权威性 | 更新时机 |
|---------|---------|-------|---------|
| Agent 入口与全局导航 | `AGENTS.md` | 权威 | 目录结构或关键规范变更时 |
| Harness 规范（本目录） | `docs/harness/` | 权威 | 运行 harness-engineering 时 |
| 术语定义 | `docs/glossary.md` | 权威 | 引入新概念时 |
| 项目详细说明 | `README.md` | 权威 | 模块或架构变更时 |
| RAG 详细设计 | `RAG详细设计.md` | 权威 | RAG 方案变更时 |
| 数据模型 | `model/*.go`（GORM 实体） | 权威 | 表结构变更时 |
| 接口契约 | `router/*.go` + `controller/*.go` 的 Swagger 注解 | 权威 | 接口变更时，随后 `swag init` |
| 错误码定义 | `common/code/code.go` | 权威 | 新增错误码时 |
| 配置结构 | `config/config.go` + `config/config.toml` | 权威 | 配置项增删时 |
| 生成产物 | `docs/docs.go` / `docs/swagger.json` / `docs/swagger.yaml` | 派生（禁止手改） | 由 `swag init` 生成 |

### 1.2 已知过期、不得直接采信的来源

以下文件当前内容可能与代码不符，读取时须交叉验证代码，不得作为决策唯一依据：

| 文件 | 已知偏差 | 处置 |
|------|---------|------|
| `num1.go` | 单例模式学习示例，非业务代码 | 忽略，不据此推断项目结构 |
| `vue-frontend/` 内旧文档 | 前端可能未覆盖全部后端接口 | 以 `router/` 注册的实际路由为准 |

### 1.3 禁止的知识来源

以下渠道的信息不应作为 Agent 决策依据（容易过时或缺乏版本控制）：

- 即时通讯记录（微信、企微等）
- 未纳入版本控制的外部 Wiki 页面
- 口头约定或会议记录
- 本地未提交的临时文件、日志、`uploads/` 运行时产物

## 2. 渐进式上下文披露

### 2.1 三层结构

```text
第一层（入口，~70 行）：AGENTS.md
  ├── 项目定位与核心能力
  ├── 目录结构（二级）
  ├── 关键规范索引（指向第二层）
  └── 铁律（6 条核心约束）

第二层（导航）：docs/harness/README.md + docs/glossary.md
  ├── 五大组件导航与摘要
  └── 各文档控制在 300 行以内

第三层（详情）：组件文档 / 代码 / 设计文档 / config.toml
  └── 仅在任务需要时访问，不主动加载到上下文
```

### 2.2 按任务类型的最小加载集

| 任务类型 | 必读 | 按需 |
|---------|------|------|
| 新增 HTTP 接口 | `AGENTS.md`、`architectural-constraints.md` | `router/`、`controller/common.go`、`common/code/code.go` |
| 接入新模型 | 同上 + `README.md` §模型工厂 | `common/aihelper/model.go`、`common/aihelper/factory.go` |
| RAG / 向量检索改动 | 同上 + `RAG详细设计.md` | `common/rag/`、`common/redis/` |
| MCP 工具扩展 | `AGENTS.md`、`common/mcp/` | `README.md` §MCP 子项目 |
| 前端页面开发 | `AGENTS.md` | `vue-frontend/src/`、`README.md` §API 接口 |
| 缺陷修复 | `AGENTS.md` + 缺陷涉及模块的规范 | 相关代码与 `config.toml` |

### 2.3 上下文预算管理

- 上下文窗口视为有限资源，优先加载与当前任务直接相关的文档
- 大文件先读目录/索引定位段落，避免全量加载；`docs/swagger.json` 等生成文件**禁止全量读取**
- 检索代码优先用符号/关键字搜索，不要逐目录列举

## 3. 动态上下文接入

### 3.1 运行时数据源

| 数据源 | 接入方式 | 用途 |
|-------|---------|------|
| 应用日志 | Gin 默认日志 → stdout | 本地 `go run` 观察请求与错误 |
| 服务存活 | HTTP `GET /api/v1/...` | 接口连通性自测 |
| Redis 索引状态 | `redis-cli FT.INFO rag_docs:*` | RAG 向量索引排查 |
| RabbitMQ 队列 | 管理控制台 / 客户端 | 消息落库排查 |
| Swagger 文档 | `GET /swagger/index.html` | 接口联调 |

### 3.2 外部系统（需配置才能访问）

| 外部系统 | 配置位置 | 说明 |
|---------|---------|------|
| MySQL | `config.toml` → `mysqlConfig` | 本地/测试库，生产库不可直连 |
| Redis | `config.toml` → `redisConfig` | 需开启 RediSearch 模块 |
| RabbitMQ | `config.toml` → `rabbitmqConfig` | 消息队列 |
| OpenAI 兼容模型 | 环境变量 `OPENAI_API_KEY` / `OPENAI_MODEL_NAME` / `OPENAI_BASE_URL` | 模型 type=1/2/3 共用 |
| 百度 TTS | `config.toml` → `voiceServiceConfig` | 语音合成 |
| 阿里 DashScope | `config.toml` → `ragModelConfig` | RAG 的 embedding + qwen |

> 平台自身**未暴露** Prometheus `/metrics` 端点，Agent 不应假设可以抓取应用指标。

## 4. 上下文更新机制

### 4.1 触发条件

- `router/` / `controller/` 接口变更 → 更新 Swagger 注解并 `swag init`，检查 `README.md` §API 接口是否需同步
- 新增/删除目录或调整分层 → 更新 `AGENTS.md` 目录树与 `architectural-constraints.md` §1.3
- 新增错误码 → 更新 `common/code/code.go`，必要时补充 `docs/glossary.md`
- 引入新概念/缩写 → 追加 `docs/glossary.md`
- 分层规则或依赖约定变更 → 更新 `architectural-constraints.md`

### 4.2 更新流程

1. 变更方在同一提交中同步更新受影响文档
2. 代码评审时核查文档是否同步（见 [entropy-management.md](entropy-management.md) §1）
3. 定期扫描遗漏（见熵管理 §1.1）

## 检查清单

- [ ] 所有知识类型都有明确的存储位置和权威等级
- [ ] `AGENTS.md` 控制在 100 行以内
- [ ] 已知过期文档已列入 §1.2，不被误采信
- [ ] 外部系统已标注配置位置
- [ ] 上下文更新触发条件覆盖接口 / 目录 / 错误码 / 术语四类变更
