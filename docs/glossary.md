# 词汇表（Glossary）

> 本项目涉及的核心概念、术语和缩写定义。Agent 和人类成员均以此为术语的唯一解释来源。

## Harness Engineering 核心概念

| 术语 | 英文 | 定义 |
|------|------|------|
| 驾驭工程 | Harness Engineering | 为 AI Agent 构建可靠运行环境的系统性方法。不优化模型本身，而是通过上下文、约束、工具、验证等环境要素提升 Agent 表现 |
| 上下文工程 | Context Engineering | 确保 Agent 能获取准确、及时、适量信息的机制。见 [`harness/context-engineering.md`](harness/context-engineering.md) |
| 架构约束 | Architectural Constraints | 通过刚性规则约束 Agent 的代码结构决策。见 [`harness/architectural-constraints.md`](harness/architectural-constraints.md) |
| 熵管理 | Entropy Management | 控制系统无序度增长的机制，包括文档园艺、技术债追踪。见 [`harness/entropy-management.md`](harness/entropy-management.md) |
| 工具能力 | Tooling | Agent 可调用的能力集合及其稳定性保障。见 [`harness/tooling.md`](harness/tooling.md) |
| 执行与验证 | Execution & Verification | Agent 的执行循环与完成前的强制验证。见 [`harness/execution-verification.md`](harness/execution-verification.md) |
| 渐进式上下文披露 | Progressive Context Disclosure | 分层文档结构（入口 → 导航 → 详情），Agent 按需深入，避免一次性加载全部信息 |
| 文档园艺 | Documentation Gardening | 定期扫描文档与代码的一致性并修复偏差 |
| 任务漂移 | Task Drift | Agent 偏离原始任务范围，处理无关内容或自主更换方案 |

## 架构与设计模式

| 术语 | 英文/缩写 | 定义 |
|------|----------|------|
| 分层架构 | Layered Architecture | 本项目后端为 `router → controller → service → dao → model` 五层，依赖只向下流动 |
| 接口适配层 | Controller | `controller/`，Gin HTTP 接口层，只做请求解析与响应封装，不含业务逻辑 |
| 业务逻辑层 | Service | `service/`，承载校验、编排、外部系统调用 |
| 数据访问层 | DAO | `dao/`，GORM 查询封装 |
| 共享基础设施 | Common | `common/`，AI 助手核心、RAG、MCP、TTS 等可复用组件，不依赖业务层 |
| 工厂模式 | Factory Pattern | `AIModelFactory` 按 `modelType` 字符串注册并创建模型，模型可插拔 |
| 策略模式 | Strategy Pattern | `AIModel` 接口 + 4 种实现（OpenAI / RAG / MCP / Ollama），运行时按类型选择 |
| 单例模式 | Singleton Pattern | `AIHelperManager`、`config.GetConfig()` 通过懒加载 + 同步原语保证全局唯一 |
| 回调 | Callback | `AIHelper.saveFunc` 回调，消息保存解耦为异步 |

## 业务领域术语

| 术语 | 英文/缩写 | 定义 |
|------|----------|------|
| 会话 | Session | 一次对话的容器，一个会话绑定一个 AIHelper（内存消息历史） |
| AI 助手 | AIHelper | 绑定单个会话，维护消息历史，持有具体 AIModel 实现 |
| 模型类型 | modelType | 字符串标识（"1"~"4"），决定使用哪个模型实现 |
| RAG | Retrieval-Augmented Generation | 检索增强生成，本项目基于 Redis RediSearch 向量索引 |
| MCP | Model Context Protocol | 模型上下文协议，本项目通过 `mark3labs/mcp-go` 实现工具调用 |
| TTS | Text To Speech | 语音合成，本项目接入百度语音 |
| 流式输出 | SSE (Server-Sent Events) | 服务端通过 `data:` 事件 + `[DONE]` 逐块推送 AI 回复 |
| 向量索引 | Vector Index | Redis `FT.CREATE` 创建的向量索引，前缀 `rag_docs:<文件名>:` |

## 技术组件

| 术语 | 英文/缩写 | 定义 |
|------|----------|------|
| Eino | cloudwego/eino | 字节跳动开源的 Go AI 框架，提供 ChatModel、Embedding、Indexer、Retriever 等抽象 |
| Gin | gin-gonic/gin | Go Web 框架，本项目 HTTP 层 |
| GORM | GORM | Go ORM 框架，连接 MySQL |
| RediSearch | RediSearch | Redis 模块，提供全文检索与向量检索能力 |
| RabbitMQ | RabbitMQ | 消息队列，本项目 Work 模式异步落库 |
| ONNX Runtime | onnxruntime_go | 本地推理运行时，本项目用于 MobileNetV2 图像分类 |
| Swagger | Swagger | 接口文档，本项目由 `swag` 从注解生成 |
| DashScope | 阿里百炼 | 阿里云大模型服务，本项目 RAG 的 Embedding 与 qwen 模型来源 |
| OpenAI | OpenAI | OpenAI 兼容接口，模型 type=1 使用 |
| Ollama | Ollama | 本地大模型运行时，模型 type=4 使用 |

## 工程实践术语

| 术语 | 英文 | 定义 |
|------|------|------|
| 人工门禁 | Manual Gating | 本项目当前无 CI，质量约束依赖开发者/Agent 自觉执行验证命令加人工评审 |
| 预完成检查 | Pre-completion Check | Agent 声明任务完成前必须执行的验证清单，见执行与验证 §2.1 |
| 技术债 | Technical Debt | 为短期交付而接受的长期维护成本，清单见熵管理 §3.2 |

## 标识与编码约定

| 约定 | 格式 | 说明 |
|------|------|------|
| 架构规则编号 | `ARCH-NNN` | 架构约束规则标识，如 `ARCH-001` 禁止 controller 直连 dao |
| 错误码段 | 1xxx / 2xxx / 3xxx / 4xxx / 5xxx / 6xxx | 成功 / 用户 / 权限 / 服务 / AI 模型 / TTS |
| 响应结构 | `{status_code, status_msg, ...}` | `controller.Response` 统一响应 |
| 鉴权头 | `Authorization: Bearer <token>` | 或 URL 参数 `?token=` |
| 接口前缀 | `/api/v1` | 所有业务接口统一前缀 |

---

*持续补充中——遇到新术语时请直接在对应分类下添加。*
