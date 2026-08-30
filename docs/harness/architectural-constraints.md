# 架构约束（Architectural Constraints）

> 目标：让 Agent "做正确的事"——通过刚性约束确保代码结构的一致性和可维护性。
>
> 本文所有规则均已核对代码实况。标注「事实」的规则当前代码严格遵守；标注「约定」的规则为团队期望、可能存在未核实的例外。

## 1. 分层架构模型

### 1.1 后端层次定义

```text
router/                路由层（Gin 路由注册 + JWT 中间件挂载）
        ↓
controller/            接口适配层（请求解析 + 统一响应封装，无业务逻辑）
        ↓
service/               业务逻辑层（校验、编排、外部系统调用）
        ↓
dao/                   数据访问层（GORM 查询封装）
        ↓
model/                 实体层（GORM 表映射，无业务依赖）
```

横切支撑（被各层按需依赖，自身不依赖业务层）：`common/`（aihelper / rag / mcp / tts / image / email / redis / rabbitmq / mysql / code）、`utils/`（utils、myjwt）、`middleware/`、`config/`。

### 1.2 请求路径（事实）

```text
HTTP → Gin Router（/api/v1 分组）
     → JWT 中间件（需鉴权的分组）
     → controller（请求解析、参数绑定）
     → service（业务逻辑 + 校验）
     → dao（GORM）
     → model（表映射）
```

**旁路路径**（已知存在，非违规）：

- `common/rabbitmq/` 消费者协程**直接调用** `dao/message` 落库，不经过 service 层（异步消息落库设计）
- `main.go` 启动阶段直接调用 `dao/message.GetAllMessages()` 重建 AIHelperManager，不经过 controller/service
- `common/aihelper/` 通过回调 `saveFunc` 触发 RabbitMQ 发布，不感知上层

### 1.3 目录与层次映射

| 层 | 目录 | 职责 | 允许依赖 |
|----|------|------|---------|
| 路由 | `router/` | 路由注册、中间件挂载 | `middleware`、`controller` |
| 适配 | `controller/` | 请求解析、响应封装 | `service`、`common/code` |
| 业务 | `service/` | 校验、编排、外部调用 | `dao`、`common/*`、`model`、`config` |
| 数据访问 | `dao/` | GORM CRUD | `model`、`common/mysql` |
| 实体 | `model/` | GORM 表结构 | 标准库 + gorm |
| 基础设施 | `common/` | AI 助手核心、RAG、MCP 等 | 标准库 + 各 SDK，**不依赖业务层** |
| 装配 | `main.go` | 初始化、依赖注入、启动 | 任意（唯一允许自上而下装配的入口） |

### 1.4 前端层次定义

```text
vue-frontend/src/views/          页面
        ↓
vue-frontend/src/router/         路由 + 登录守卫
        ↓
vue-frontend/src/utils/api.js    axios 封装（自动携带 JWT）
```

## 2. 依赖与结构规则

### 2.1 规则清单

| 规则编号 | 名称 | 描述 | 状态 | 修复指引 |
|---------|------|------|------|---------|
| ARCH-001 | 禁止 controller 直连 dao | `controller/**` 不得 import `dao` | 事实（已核实，无违例） | 在 `service` 中新增方法，controller 调用 service |
| ARCH-002 | 禁止 service 反向依赖 controller | `service/**` 不得 import `controller/**` | 事实（已核实，无违例） | 通过参数传递，不得反向引用 |
| ARCH-003 | 禁止 common 依赖业务层 | `common/**` 不得 import `controller` / `service` / `dao` | 事实（已核实，无违例） | 共享类型下沉到 `common` 或 `model` |
| ARCH-004 | 实体无业务依赖 | `model/**` 不得 import 业务层 | 事实（已核实，无违例） | 保持 model 只依赖标准库 + gorm |
| ARCH-005 | 统一响应结构 | 接口返回 `controller.Response`，错误码用 `common/code.Code` | 事实 | 复用 `Response.CodeOf()`，禁止自造 JSON 结构 |
| ARCH-006 | 错误码集中定义 | 错误码统一在 `common/code/code.go`，禁止硬编码数字 | 约定 | 新增 `CodeXxx` 常量 + `msg` 映射 |
| ARCH-007 | 鉴权前置到路由 | 需登录的接口在 `router` 挂 `jwt.Auth()`，controller 不重复校验 | 事实 | 在 `router/*.go` 分组挂中间件 |
| ARCH-008 | 模型可插拔 | 新模型实现 `AIModel` 接口 + 工厂注册，禁止 if-else 分发 | 事实 | 见 `common/aihelper/factory.go` `RegisterModel` |
| ARCH-009 | 密钥不入库不入配置文件 | OpenAI API Key 走环境变量，不写进 `config.toml` 或代码 | 事实 | 用 `os.Getenv("OPENAI_API_KEY")` |
| ARCH-010 | 生成代码只读 | `docs/docs.go` / `swagger.json` / `swagger.yaml` 禁止手改 | 事实（`swag init` 生成） | 改 controller 注解后 `swag init` |

### 2.2 约束的执行方式（现状）

本项目**没有**自定义架构 Linter，也没有 CI。ARCH-001～ARCH-010 的执行依赖以下通道：

| 通道 | 覆盖规则 | 命令 |
|------|---------|------|
| Go 编译器 | ARCH-001～004（import 环）、通用编译 | `go build ./...` |
| 人工/Agent 评审 | ARCH-005～010 | 提交前自查 |

Agent 在提交前**必须**自行逐条核对 §2.1 中与本次变更相关的规则，不能依赖 CI 拦截——项目当前无 CI 流水线（见 [entropy-management.md](entropy-management.md) §2）。

### 2.3 错误信息格式约定

新增约束检查时，错误信息应直接包含修复指引：

```text
[ARCH-001] 违反依赖规则：controller/session/session.go 引用了 dao/session
修复方式：在 service/session/session.go 中新增方法封装该数据访问，controller 调用 service
参考文档：docs/harness/architectural-constraints.md#21-规则清单
```

## 3. Parse, Don't Validate

### 3.1 原则

在数据进入系统的边界处，将原始数据**解析**为强类型，后续代码只操作解析后的类型，无需重复验证。

### 3.2 数据边界

| 边界 | 输入类型 | 解析目标 | 处理位置 |
|------|---------|---------|---------|
| HTTP 请求 | JSON / form | Gin 绑定到结构体 | `controller`（`c.ShouldBindJSON` 等） |
| 业务语义校验 | 绑定后的结构体 | 校验后的领域参数 | `service/`（**必须在此完成**，controller 不做校验） |
| 服务配置 | TOML | `config.Config` struct | 启动阶段 `config.GetConfig()` |
| 数据库行 | SQL rows | `model` 实体 | GORM（自动） |
| LLM 消息 | DB 消息 / Eino schema | `schema.Message` | `utils.ConvertToSchemaMessages` 集中转换 |
| 外部系统响应 | JSON | 各 `common/*` 内的类型 | 各组件客户端内部 |

### 3.3 错误表达

- 统一返回 `controller.Response`，错误码用 `common/code.Code`
- 新增错误场景时**选择合适的 `CodeXxx` 段**（1xxx 成功 / 2xxx 用户 / 3xxx 权限 / 4xxx 服务 / 5xxx AI 模型 / 6xxx TTS），不要一律返回 `CodeServerBusy`

## 4. 架构决策记录（ADR）

本项目**没有** `docs/adr/` 目录。架构决策实际承载在 `README.md` §关键设计决策 与 `RAG详细设计.md`。Agent 做架构决策前须先检索这两处，确保不与历史决策冲突。

## 检查清单

- [ ] 变更涉及的分层边界已确认，未引入反向依赖
- [ ] 未手工编辑 `docs/swagger.json` 等生成文件
- [ ] 新增接口使用 `controller.Response` + `common/code`（ARCH-005/006）
- [ ] 新增需鉴权接口已在 router 挂中间件（ARCH-007）
- [ ] 新增模型走 AIModel 接口 + 工厂注册（ARCH-008）
- [ ] 密钥未写入代码或 config.toml（ARCH-009）
- [ ] 业务校验落在 service 层，不在 controller
