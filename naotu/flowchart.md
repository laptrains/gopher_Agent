# GopherAI 整体流程图

> 根据 README.md 生成。使用 Mermaid 语法，可在 GitHub / VSCode / Typora 等支持 Mermaid 的编辑器中直接预览。

```mermaid
graph TB
    %% ============ 1. 启动流程 ============
    subgraph STARTUP["🚀 启动流程 (main.go)"]
        S1([main.go]) --> S2[加载 TOML 配置<br/>GetConfig 懒加载单例]
        S2 --> S3[(MySQL 初始化<br/>GORM AutoMigrate 建表)]
        S3 --> S4[readDataFromDB<br/>从 DB 加载历史消息<br/>重建 AIHelperManager]
        S4 --> S5[(Redis 初始化<br/>验证码 + RAG 向量索引)]
        S5 --> S6[RabbitMQ 初始化<br/>启动 Message 队列消费者]
        S6 --> S7[启动 HTTP 服务<br/>host:9090]
    end

    %% ============ 2. HTTP 请求链路 ============
    subgraph HTTP["📡 HTTP 请求链路"]
        H1[前端 Vue3 + Element Plus<br/>axios 自动携带 JWT] -->|REST / SSE| H2[Gin Router /api/v1]
        H2 --> H3{JWT 鉴权中间件<br/>Bearer Token / ?token=}
        H3 -->|校验通过| H4[Controller 层<br/>user / session / tts / image / file]
        H3 -.->|/user/* 无需鉴权| H4
    end

    %% ============ 3. AI 对话链路 ============
    subgraph AICHAIN["🧠 AI 对话链路"]
        C1[Session Service<br/>会话 CRUD + 编排] --> C2[AIHelperManager<br/>user × session 映射 · 全局单例]
        C2 --> C3[AIHelper<br/>内存消息历史 + saveFunc 回调]
        C3 --> C4[AIModelFactory<br/>按 modelType 工厂创建]
        C4 --> M1[OpenAIModel<br/>type=1]
        C4 --> M2[AliRAGModel<br/>type=2]
        C4 --> M3[MCPModel<br/>type=3]
        C4 --> M4[OllamaModel<br/>type=4]
        C3 -->|GenerateResponse / StreamResponse| C5[LLM Provider]
    end

    %% ============ 4. 消息异步落库 ============
    subgraph SAVE["💾 消息异步落库"]
        A1[AIHelper.saveFunc<br/>消息 JSON 化后发布] --> A2[RabbitMQ<br/>Message 队列]
        A2 --> A3[消费者协程]
        A3 --> A4[dao/message.CreateMessage]
        A4 --> A5[(MySQL t_message)]
    end

    %% ============ 5. RAG 机制 ============
    subgraph RAG["📚 RAG 检索增强"]
        R1[上传文档 .md/.txt] --> R2[校验扩展名<br/>删除旧文件与旧索引]
        R2 --> R3[存入 uploads/用户名/UUID.ext]
        R3 --> R4[NewRAGIndexer<br/>创建 Redis 向量索引 FT.CREATE]
        R4 --> R5[Embedding + Store<br/>rag_docs:文件名]
        R6[用户提问] --> R7[NewRAGQuery<br/>Redis 余弦检索 TopK=5]
        R7 --> R8[BuildRAGPrompt 拼入提示词]
        R8 --> R9[替换最后一条用户消息]
        R9 --> R10[qwen 生成回答]
    end

    %% ============ 6. MCP 两阶段调用 ============
    subgraph MCPCHAIN["🔌 MCP 两阶段调用"]
        P1[用户问题] --> P2[第一次 LLM 调用<br/>要求返回固定 JSON]
        P2 --> P3{isToolCall?}
        P3 -->|false| P4[直接返回回答]
        P3 -->|true| P5[调用 MCP 工具<br/>get_weather]
        P5 --> P6[第二次 LLM 调用<br/>结合工具结果总结]
        P6 --> P7[流式 / 同步返回]
    end

    %% ============ 7. 外部服务 ============
    subgraph EXT["🌐 外部服务"]
        E1[DashScope<br/>embedding + qwen]
        E2[MCP Server<br/>StreamableHTTP :8081]
        E3[百度 TTS<br/>access_token → 创建任务 → 查询]
        E4[ONNX Runtime<br/>MobileNetV2 本地推理]
    end

    %% ============ 跨子图连接 ============
    S7 --> H1
    H4 -->|聊天 / 会话 / 历史| C1
    H4 -->|上传文档| R1
    H4 -->|图片识别| E4
    H4 -->|TTS 创建 / 查询| E3
    M2 --> E1
    M2 --> R6
    M3 --> E2
    C3 --> A1
    RAG --> E1
```

---

## 图说明

| 子图 | 对应 README 章节 |
|------|------------------|
| 🚀 启动流程 | 「模块详解 → main.go 启动入口」 |
| 📡 HTTP 请求链路 | 「架构图」+「API 接口」 |
| 🧠 AI 对话链路 | 「核心概念 1. AIHelperManager + AIHelper」|
| 💾 消息异步落库 | 「核心概念 5. 消息异步落库」 |
| 📚 RAG 检索增强 | 「核心概念 3. RAG 问答机制」+「数据流 → RAG 文件上传」 |
| 🔌 MCP 两阶段调用 | 「核心概念 4. MCP 模型（两阶段调用）」 |
| 🌐 外部服务 | 「技术栈」+「模块详解」 |
