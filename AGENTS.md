# AGENTS.md

> Agent 认知本项目的第一站——快速了解项目全貌、找到代码和规范、知道规矩。

## 项目概述

- **项目名称**：GopherAI
- **Go module**：`GopherAI`
- **定位**：基于 Eino 框架的全栈 AI 助手，覆盖多模型对话、RAG 问答、MCP 工具调用、语音合成、图像识别、邮箱注册登录
- **技术栈**：Go 1.24 + Gin + GORM(MySQL) + cloudwego/eino；Redis(RediSearch 向量库) + RabbitMQ；Vue 3 + Element Plus

## 目录结构

```text
GopherAI-v2/
├── main.go            入口（初始化 MySQL → Redis → RabbitMQ → 加载历史 → 启动 HTTP）
├── config/            TOML 配置 + 懒加载单例 GetConfig()
├── router/            Gin 路由注册（/api/v1 分组 + JWT 鉴权）
├── controller/        HTTP 接口层（请求解析 + 统一响应 Response）
├── service/           业务逻辑层
├── dao/               数据访问层（GORM）
├── model/             数据模型（GORM AutoMigrate）
├── middleware/jwt/    JWT 鉴权中间件
├── common/            共享基础设施：aihelper / rag / mcp / tts / image / email / redis / rabbitmq / mysql / code
├── utils/             工具函数（utils.go、myjwt）
├── vue-frontend/      Vue 3 前端（Element Plus + axios + vue-router）
├── docs/              Swagger 生成产物 + Harness 规范
└── num1.go            单例模式示例（学习文件，非业务代码）
```

请求链路：HTTP → Gin 路由 → JWT 中间件 → controller → service → dao → model。

## 关键规范

- Harness 规范（工具能力、架构约束、验证要求）→ [`docs/harness/README.md`](docs/harness/README.md)
- 术语定义 → [`docs/glossary.md`](docs/glossary.md)
- 项目详细说明 → [`README.md`](README.md)
- RAG 详细设计 → [`RAG详细设计.md`](RAG详细设计.md)

## 常用命令

| 场景 | 命令 |
|------|------|
| 启动后端 | `go run main.go` |
| 启动 MCP Server（模型 type=3 依赖） | `cd common/mcp && go run main.go --mode server --http-addr :8081` |
| 启动前端 | `cd vue-frontend && npm install && npm run serve` |
| 编译检查 | `go build ./...` |
| 代码格式化 | `gofmt -w .` / `goimports -w .` |
| 更新 Swagger 文档 | `swag init` |

> 本项目暂无 Makefile、无 CI、无单元测试（见 [`docs/harness/entropy-management.md`](docs/harness/entropy-management.md)）。

## 铁律

1. **分层不跨越**：router → controller → service → dao → model，controller 不得直接访问 dao
2. **统一响应**：接口返回 `controller.Response` + `common/code` 错误码，禁止自造响应格式
3. **错误码统一**：错误码集中在 `common/code/code.go`，禁止硬编码数字
4. **鉴权前置**：需登录的接口在 router 挂 `jwt.Auth()`，不要在 controller 里重复校验
5. **模型可插拔**：新增模型实现 `AIModel` 接口并在工厂注册，禁止硬编码 if-else 分发
6. **提交前自查**：无 CI 兜底，提交前必须 `go build ./...` 通过（见 [`docs/harness/execution-verification.md`](docs/harness/execution-verification.md)）
