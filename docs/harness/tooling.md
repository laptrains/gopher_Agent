# 工具能力（Tooling）

> 目标：让 Agent "能行动"——明确可用工具边界，保障执行稳定性。

## 1. 工具清单

### 1.1 CLI 工具

| 工具 | 必需 | 检测条件 | 用途 |
|------|------|---------|------|
| `go` | 是 | `go.mod` 存在 | 编译、运行、依赖管理 |
| `swag` | 是 | 接口变更时 | 从注解生成 Swagger 文档 |
| `git` | 是 | 始终 | 版本控制 |
| `node` / `npm` | 否 | `vue-frontend/package.json` 存在 | 前端构建与运行 |
| `redis-cli` | 否 | RAG 排查时 | 检查 RediSearch 索引 |
| `docker` | 否 | 部署时 | 可选容器化 |

### 1.2 项目自有命令

| 命令 | 作用 |
|------|------|
| `go run main.go` | 启动后端（先初始化 MySQL → Redis → RabbitMQ） |
| `go build ./...` | 编译检查（**提交前必做**） |
| `gofmt -w .` | 格式化代码 |
| `goimports -w .` | 格式化 + 整理 import（需安装 goimports） |
| `swag init` | 更新 Swagger 文档（生成 `docs/docs.go` 等） |
| `go vet ./...` | 静态检查（可选） |

MCP 子项目（独立 `go.mod`）：

| 命令 | 作用 |
|------|------|
| `cd common/mcp && go run main.go --mode server --http-addr :8081` | 启动 MCP Server |
| `cd common/mcp && go run main.go --mode client --city "北京"` | 客户端调用 get_weather |

前端：

| 命令 | 作用 |
|------|------|
| `cd vue-frontend && npm install` | 安装依赖 |
| `cd vue-frontend && npm run serve` | 启动开发服务器 |

> 本项目**无 Makefile**、**无 `make` 目标**。请直接使用上表中的 `go` / `npm` / `swag` 命令。

### 1.3 代码生成（唯一允许的生成方式）

| 产物 | 生成命令 | 说明 |
|------|---------|------|
| `docs/docs.go` / `swagger.json` / `swagger.yaml` | `swag init` | 从 controller 注解生成，**禁止手改** |

## 2. 工具接口规范

### 2.1 外部系统客户端约定

`common/` 下的客户端当前多为全局单例 + 懒加载（`Init` / `GetConfig` 模式）。新增客户端时建议：

- HTTP 客户端必须设置 `Timeout`
- 配置缺失时以日志降级，**不得 panic**（参照 `main.go` 对 RabbitMQ 的 3 次重试后退出、以及对可选外部系统的容错处理）
- 响应必须解析为强类型结构体，不得在业务层操作 `map[string]any`

### 2.2 前端请求约定

- 统一通过 `vue-frontend/src/utils/api.js` 的 axios 实例发起请求
- 认证走 `Authorization: Bearer`（axios 拦截器自动携带 JWT）
- 响应解包统一处理 `{status_code, status_msg, ...}`，业务代码不重复解包

## 3. 稳定性保障

### 3.1 执行环境

| 环境 | 说明 |
|------|------|
| 本地开发机 | Agent 默认执行环境，可读写仓库目录 |
| 依赖服务 | MySQL / Redis（含 RediSearch）/ RabbitMQ 需本地或测试环境就绪 |
| 生产环境 | 仅可读不可写，见 §3.3 红线 |

### 3.2 容错策略

| 策略 | 配置 | 适用 |
|------|------|------|
| 超时 | 外部 HTTP 调用设置 Timeout | `common/*` 外部客户端 |
| 重试 | RabbitMQ 初始化重试 3 次（间隔 2s） | `main.go` 启动 |
| 降级 | 外部系统未配置时 Warn 并禁用相关功能 | TTS / RAG / MCP 等可选组件 |

### 3.3 操作红线（禁止执行）

| 红线 | 说明 | 正确做法 |
|------|------|---------|
| **禁止直连生产库执行写操作** | 包括手工 DDL、`UPDATE`/`DELETE` 生产数据 | 通过 GORM `AutoMigrate` 声明，由部署流程执行 |
| **禁止手工编辑生成代码** | `docs/docs.go`、`swagger.json`、`swagger.yaml` | 改 controller 注解后 `swag init` |
| **禁止提交密钥与凭证** | 包括 API Key、邮箱授权码、真实生产地址 | 用环境变量/占位符；`config.toml` 只留占位 |
| **禁止向主分支强推或改写历史** | `push --force`、`reset --hard`、删除分支需确认 | 正常提交、合并 |

补充禁止项：

- 不得提交 `uploads/`、日志、临时文件等运行时产物（确认 `.gitignore` 覆盖）
- 不得修改 `.git/config` 或 git 全局配置

## 4. 工具扩展规范

### 4.1 新增外部组件

1. 在 `common/<name>/` 下新建包，实现客户端
2. 配置项加到 `config/config.go` 的 `Config` 结构体 + `config.toml`
3. 在 `main.go` 或对应 service 中初始化，遵循"配置缺失即降级"原则

### 4.2 扩展原则

- 一个组件只做一件事，功能边界清晰
- 新工具不修改既有分层规则
- 工具文档与代码同仓库版本控制

## 检查清单

- [ ] 所有标记为「必需」的工具在当前环境已就绪
- [ ] 本次操作未触碰 §3.3 任一条红线
- [ ] 新增外部调用配置了超时，并遵循降级原则
- [ ] 新增组件已同步 `config` 结构与 `config.toml`
