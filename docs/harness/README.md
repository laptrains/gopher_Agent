# Harness Engineering 规范

> 本目录定义了项目的 AI Agent 运行环境规范，是 Agent 理解项目边界、工具能力和行为约束的唯一来源。
>
> 驾驭工程（Harness Engineering）：为 AI Agent 构建可靠运行环境的系统性方法。不优化模型本身，而是通过上下文、约束、工具、验证等环境要素提升 Agent 表现。

## 项目概述

- **项目名称**：GopherAI
- **Go module**：`GopherAI`
- **技术栈**：Go 1.24 + Gin + GORM(MySQL) + cloudwego/eino；Redis(RediSearch) + RabbitMQ；Vue 3 + Element Plus
- **Agent 适用场景**：需求理解与评估、后端接口开发、AI 模型/工具接入、前端页面开发、RAG 与向量检索调试、缺陷定位修复、文档一致性维护

## 规范导航

| 组件 | 文档 | 概要 |
|------|------|------|
| 上下文工程 | [context-engineering.md](context-engineering.md) | 知识来源、渐进式披露、上下文预算 |
| 架构约束 | [architectural-constraints.md](architectural-constraints.md) | 分层模型、依赖规则、Parse Don't Validate |
| 熵管理 | [entropy-management.md](entropy-management.md) | 文档园艺、技术债追踪、门禁现状 |
| 工具能力 | [tooling.md](tooling.md) | CLI 清单、操作红线、扩展规范 |
| 执行与验证 | [execution-verification.md](execution-verification.md) | 执行循环、验证清单、任务漂移检测 |

配套规范：

- 术语定义 → [`../glossary.md`](../glossary.md)
- 项目详细说明 → [`../../README.md`](../../README.md)
- RAG 详细设计 → [`../../RAG详细设计.md`](../../RAG详细设计.md)

## 使用说明

1. Agent 首次接触项目时，先读 `AGENTS.md` 获取全局视图，再读本文件获取 Harness 全貌
2. 执行具体任务时，按需深入阅读对应组件文档，不要一次性全量加载
3. 规范更新后需同步检查关联组件的一致性（见 [entropy-management.md](entropy-management.md)）

## 本项目的三条关键事实

这三点与项目根 `README.md` 的静态描述可能不完全同步，以本处为准（已核对代码）：

1. **分层是 `router → controller → service → dao → model` 五层**，`common/` 为共享基础设施、`config` 在最底层被全项目引用。controller 不得直连 dao，必须经 service。
2. **统一响应是 `controller.Response{StatusCode, StatusMsg}`**，错误码集中在 `common/code/code.go`（`Code` 类型 + `Msg()` 方法），不在各接口散落定义。
3. **配置使用 TOML 解析（`config/config.toml`）**，通过 `config.GetConfig()` 懒加载单例访问；OpenAI 兼容模型的密钥走环境变量（`OPENAI_API_KEY` 等），不入库、不入 config.toml。

## 版本记录

| 版本 | 日期 | 变更说明 |
|------|------|---------|
| 1.0.0 | 2026-08-26 | 初始版本，基于代码实况生成 |
