package aihelper

import (
	"GopherAI/common/rag"
	"GopherAI/config"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

type StreamCallback func(msg string)

// AIModel 定义AI模型接口
type AIModel interface {
	GenerateResponse(ctx context.Context, messages []*schema.Message) (*schema.Message, error)
	StreamResponse(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error)
	GetModelType() string
}

// =================== OpenAI 实现 ===================
type OpenAIModel struct {
	llm model.ToolCallingChatModel
}

func NewOpenAIModel(ctx context.Context) (*OpenAIModel, error) {
	key := os.Getenv("OPENAI_API_KEY")
	modelName := os.Getenv("OPENAI_MODEL_NAME")
	baseURL := os.Getenv("OPENAI_BASE_URL")

	llm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: baseURL,
		Model:   modelName,
		APIKey:  key,
	})
	if err != nil {
		return nil, fmt.Errorf("create openai model failed: %v", err)
	}
	return &OpenAIModel{llm: llm}, nil
}

func (o *OpenAIModel) GenerateResponse(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	resp, err := o.llm.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("openai generate failed: %v", err)
	}
	return resp, nil
}

func (o *OpenAIModel) StreamResponse(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
	stream, err := o.llm.Stream(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("openai stream failed: %v", err)
	}
	defer stream.Close()

	var fullResp strings.Builder

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("openai stream recv failed: %v", err)
		}
		if len(msg.Content) > 0 {
			fullResp.WriteString(msg.Content) // 聚合

			cb(msg.Content) // 实时调用cb函数，方便主动发送给前端
		}
	}

	return fullResp.String(), nil //返回完整内容，方便后续存储
}

func (o *OpenAIModel) GetModelType() string { return "1" }

// =================== Ollama 实现 ===================

// OllamaModel Ollama模型实现
type OllamaModel struct {
	llm model.ToolCallingChatModel
}

func NewOllamaModel(ctx context.Context, baseURL, modelName string) (*OllamaModel, error) {
	llm, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		BaseURL: baseURL,
		Model:   modelName,
	})
	if err != nil {
		return nil, fmt.Errorf("create ollama model failed: %v", err)
	}
	return &OllamaModel{llm: llm}, nil
}

func (o *OllamaModel) GenerateResponse(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	resp, err := o.llm.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("ollama generate failed: %v", err)
	}
	return resp, nil
}

func (o *OllamaModel) StreamResponse(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
	stream, err := o.llm.Stream(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("ollama stream failed: %v", err)
	}
	defer stream.Close()
	var fullResp strings.Builder
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("openai stream recv failed: %v", err)
		}
		if len(msg.Content) > 0 {
			fullResp.WriteString(msg.Content) // 聚合
			cb(msg.Content)                   // 实时调用cb函数，方便主动发送给前端
		}
	}
	return fullResp.String(), nil //返回完整内容，方便后续存储
}

func (o *OllamaModel) GetModelType() string { return "4" }

// =================== RAG 实现 ===================
type AliRAGModel struct {
	llm      model.ToolCallingChatModel
	username string // 用于获取用户的文档
}

func NewAliRAGModel(ctx context.Context, username string) (*AliRAGModel, error) {
	key := os.Getenv("OPENAI_API_KEY")
	conf := config.GetConfig()
	modelName := conf.RagModelConfig.RagChatModelName
	baseURL := conf.RagModelConfig.RagBaseUrl

	llm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: baseURL,
		Model:   modelName,
		APIKey:  key,
	})
	if err != nil {
		return nil, fmt.Errorf("create ali rag model failed: %v", err)
	}
	return &AliRAGModel{
		llm:      llm,
		username: username,
	}, nil
}

func (o *AliRAGModel) GenerateResponse(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	// 1. 创建 RAG 查询器
	ragQuery, err := rag.NewRAGQuery(ctx, o.username)
	if err != nil {
		log.Printf("Failed to create RAG query (user may not have uploaded file): %v", err)
		// 如果用户没有上传文件，直接使用原始问题
		resp, err := o.llm.Generate(ctx, messages)
		if err != nil {
			return nil, fmt.Errorf("ali rag generate failed: %v", err)
		}
		return resp, nil
	}

	// 2. 获取用户最后一条消息作为查询
	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages provided")
	}
	lastMessage := messages[len(messages)-1]
	query := lastMessage.Content

	// 3. 检索相关文档
	docs, err := ragQuery.RetrieveDocuments(ctx, query)
	if err != nil {
		log.Printf("Failed to retrieve documents: %v", err)
		// 检索失败，使用原始问题
		resp, err := o.llm.Generate(ctx, messages)
		if err != nil {
			return nil, fmt.Errorf("ali rag generate failed: %v", err)
		}
		return resp, nil
	}

	// 4. 构建包含检索结果的提示词
	ragPrompt := rag.BuildRAGPrompt(query, docs)

	// 5. 替换最后一条消息为 RAG 提示词
	ragMessages := make([]*schema.Message, len(messages))
	copy(ragMessages, messages)
	ragMessages[len(ragMessages)-1] = &schema.Message{
		Role:    schema.User,
		Content: ragPrompt,
	}

	// 6. 调用 LLM 生成回答
	resp, err := o.llm.Generate(ctx, ragMessages)
	if err != nil {
		return nil, fmt.Errorf("ali rag generate failed: %v", err)
	}
	return resp, nil
}

func (o *AliRAGModel) StreamResponse(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
	// 1. 创建 RAG 查询器
	ragQuery, err := rag.NewRAGQuery(ctx, o.username)
	if err != nil {
		log.Printf("Failed to create RAG query (user may not have uploaded file): %v", err)
		// 如果用户没有上传文件，直接使用原始问题
		return o.streamWithoutRAG(ctx, messages, cb)
	}

	// 2. 获取用户最后一条消息作为查询
	if len(messages) == 0 {
		return "", fmt.Errorf("no messages provided")
	}
	lastMessage := messages[len(messages)-1]
	query := lastMessage.Content

	// 3. 检索相关文档
	docs, err := ragQuery.RetrieveDocuments(ctx, query)
	if err != nil {
		log.Printf("Failed to retrieve documents: %v", err)
		// 检索失败，使用原始问题
		return o.streamWithoutRAG(ctx, messages, cb)
	}

	// 4. 构建包含检索结果的提示词
	ragPrompt := rag.BuildRAGPrompt(query, docs)

	// 5. 替换最后一条消息为 RAG 提示词
	ragMessages := make([]*schema.Message, len(messages))
	copy(ragMessages, messages)
	ragMessages[len(ragMessages)-1] = &schema.Message{
		Role:    schema.User,
		Content: ragPrompt,
	}

	// 6. 流式调用 LLM
	stream, err := o.llm.Stream(ctx, ragMessages)
	if err != nil {
		return "", fmt.Errorf("ali rag stream failed: %v", err)
	}
	defer stream.Close()

	var fullResp strings.Builder

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("ali rag stream recv failed: %v", err)
		}
		if len(msg.Content) > 0 {
			fullResp.WriteString(msg.Content)
			cb(msg.Content)
		}
	}

	return fullResp.String(), nil
}

// streamWithoutRAG 当没有 RAG 文档时的流式响应
func (o *AliRAGModel) streamWithoutRAG(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
	stream, err := o.llm.Stream(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("ali rag stream failed: %v", err)
	}
	defer stream.Close()

	var fullResp strings.Builder

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("ali rag stream recv failed: %v", err)
		}
		if len(msg.Content) > 0 {
			fullResp.WriteString(msg.Content)
			cb(msg.Content)
		}
	}

	return fullResp.String(), nil
}

func (o *AliRAGModel) GetModelType() string { return "2" }

// =================== MCP 实现 ===================

// MCPModel MCP模型实现，集成MCP服务
// 采用原生function calling：将MCP工具的schema绑定到LLM，由模型自主决策是否调用工具
type MCPModel struct {
	llm        model.ToolCallingChatModel
	toolLLM    model.ToolCallingChatModel // 绑定MCP工具后的模型（惰性初始化）
	mcpClient  *client.Client
	username   string
	mcpBaseURL string
	mu         sync.Mutex
}

// NewMCPModel 创建MCP模型实例
func NewMCPModel(ctx context.Context, username string) (*MCPModel, error) {
	key := os.Getenv("OPENAI_API_KEY")
	conf := config.GetConfig()
	modelName := conf.RagModelConfig.RagChatModelName
	baseURL := conf.RagModelConfig.RagBaseUrl

	// 创建LLM
	llm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: baseURL,
		Model:   modelName,
		APIKey:  key,
	})
	if err != nil {
		return nil, fmt.Errorf("create mcp model failed: %v", err)
	}

	mcpBaseURL := "http://localhost:8081/mcp"

	return &MCPModel{
		llm:        llm,
		mcpBaseURL: mcpBaseURL,
		username:   username,
	}, nil
}

// ensureMCPTools 惰性初始化MCP客户端并把远端工具绑定到LLM，只生效一次
func (m *MCPModel) ensureMCPTools(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.toolLLM != nil {
		return nil
	}

	// 创建并初始化MCP客户端
	httpTransport, err := transport.NewStreamableHTTP(m.mcpBaseURL)
	if err != nil {
		return fmt.Errorf("create mcp transport failed: %v", err)
	}
	mcpClient := client.NewClient(httpTransport)

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "MCP-Go AIHelper Client",
		Version: "1.0.0",
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	if _, err := mcpClient.Initialize(ctx, initRequest); err != nil {
		mcpClient.Close()
		return fmt.Errorf("mcp client initialize failed: %v", err)
	}

	// 拉取远端工具列表并转换为eino工具描述
	listResp, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		mcpClient.Close()
		return fmt.Errorf("mcp list tools failed: %v", err)
	}
	toolInfos, err := convertMCPTools(listResp.Tools)
	if err != nil {
		mcpClient.Close()
		return fmt.Errorf("convert mcp tools failed: %v", err)
	}

	// 绑定工具：WithTools返回新实例，不影响未绑定的llm
	toolLLM, err := m.llm.WithTools(toolInfos)
	if err != nil {
		mcpClient.Close()
		return fmt.Errorf("bind mcp tools failed: %v", err)
	}

	m.mcpClient = mcpClient
	m.toolLLM = toolLLM
	return nil
}

// GenerateResponse 生成响应，集成MCP工具
func (m *MCPModel) GenerateResponse(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages provided")
	}

	llm := m.llm
	if err := m.ensureMCPTools(ctx); err != nil {
		// MCP服务不可用时降级为普通对话
		log.Printf("MCP tools unavailable, fallback to plain chat: %v", err)
	} else {
		llm = m.toolLLM
	}

	// 第一次调用：模型根据绑定的工具schema自主决定是否调用工具
	resp, err := llm.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("mcp generate failed: %v", err)
	}

	// 情况1：模型未请求工具调用，直接返回
	if len(resp.ToolCalls) == 0 {
		return resp, nil
	}

	// 情况2：执行工具调用，把结果以tool消息回传给模型生成最终回答
	toolMsgs := m.executeToolCalls(ctx, resp.ToolCalls)
	followUp := make([]*schema.Message, 0, len(messages)+1+len(toolMsgs))
	followUp = append(followUp, messages...)
	followUp = append(followUp, resp)
	followUp = append(followUp, toolMsgs...)

	finalResp, err := llm.Generate(ctx, followUp)
	if err != nil {
		return nil, fmt.Errorf("mcp final generate failed: %v", err)
	}
	return finalResp, nil
}

// StreamResponse 流式响应，集成MCP工具
func (m *MCPModel) StreamResponse(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
	if len(messages) == 0 {
		return "", fmt.Errorf("no messages provided")
	}

	llm := m.llm
	if err := m.ensureMCPTools(ctx); err != nil {
		// MCP服务不可用时降级为普通对话
		log.Printf("MCP tools unavailable, fallback to plain chat: %v", err)
	} else {
		llm = m.toolLLM
	}

	// 第一次调用使用同步接口，判断模型是否请求工具调用
	resp, err := llm.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("mcp generate failed: %v", err)
	}

	// 情况1：模型未请求工具调用，内容一次性回调给前端
	if len(resp.ToolCalls) == 0 {
		if len(resp.Content) > 0 {
			cb(resp.Content)
		}
		return resp.Content, nil
	}

	// 情况2：执行工具调用，把结果回传给模型后流式生成最终回答
	toolMsgs := m.executeToolCalls(ctx, resp.ToolCalls)
	followUp := make([]*schema.Message, 0, len(messages)+1+len(toolMsgs))
	followUp = append(followUp, messages...)
	followUp = append(followUp, resp)
	followUp = append(followUp, toolMsgs...)

	stream, err := llm.Stream(ctx, followUp)
	if err != nil {
		return "", fmt.Errorf("mcp stream failed: %v", err)
	}
	defer stream.Close()

	var finalResp strings.Builder

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("mcp stream recv failed: %v", err)
		}
		if len(msg.Content) > 0 {
			finalResp.WriteString(msg.Content)
			cb(msg.Content)
		}
	}

	return finalResp.String(), nil
}

// executeToolCalls 逐个执行模型请求的工具调用，返回对应的tool消息
// 单个工具失败时把错误信息写入tool消息（让模型知情），不中断整个流程
func (m *MCPModel) executeToolCalls(ctx context.Context, toolCalls []schema.ToolCall) []*schema.Message {
	toolMsgs := make([]*schema.Message, 0, len(toolCalls))
	for _, tc := range toolCalls {
		var content string
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			log.Printf("MCP tool %s arguments parse failed: %v", tc.Function.Name, err)
			content = fmt.Sprintf("tool arguments parse failed: %v", err)
		} else if result, err := m.callMCPTool(ctx, tc.Function.Name, args); err != nil {
			log.Printf("MCP tool %s call failed: %v", tc.Function.Name, err)
			content = fmt.Sprintf("tool call failed: %v", err)
		} else {
			content = result
		}
		toolMsgs = append(toolMsgs, schema.ToolMessage(content, tc.ID))
	}
	return toolMsgs
}

// callMCPTool 调用MCP工具并提取文本结果
func (m *MCPModel) callMCPTool(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
	if m.mcpClient == nil {
		return "", fmt.Errorf("mcp client not initialized")
	}

	result, err := m.mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	})
	if err != nil {
		return "", fmt.Errorf("mcp tool call failed: %v", err)
	}

	// 提取工具结果文本
	var text string
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			text += textContent.Text + "\n"
		}
	}

	return text, nil
}

// convertMCPTools 将MCP工具schema转换为eino工具描述
func convertMCPTools(tools []mcp.Tool) ([]*schema.ToolInfo, error) {
	toolInfos := make([]*schema.ToolInfo, 0, len(tools))
	for _, t := range tools {
		params, err := convertMCPProperties(t.InputSchema.Properties, t.InputSchema.Required)
		if err != nil {
			return nil, fmt.Errorf("tool %s: %v", t.Name, err)
		}
		toolInfos = append(toolInfos, &schema.ToolInfo{
			Name:        t.Name,
			Desc:        t.Description,
			ParamsOneOf: schema.NewParamsOneOfByParams(params),
		})
	}
	return toolInfos, nil
}

// convertMCPProperties 转换JSON Schema properties为eino参数描述
func convertMCPProperties(props map[string]any, required []string) (map[string]*schema.ParameterInfo, error) {
	requiredSet := make(map[string]bool, len(required))
	for _, r := range required {
		requiredSet[r] = true
	}
	params := make(map[string]*schema.ParameterInfo, len(props))
	for name, raw := range props {
		prop, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid schema of property %s", name)
		}
		info, err := convertMCPProperty(prop)
		if err != nil {
			return nil, fmt.Errorf("property %s: %v", name, err)
		}
		info.Required = requiredSet[name]
		params[name] = info
	}
	return params, nil
}

// convertMCPProperty 递归转换单个JSON Schema属性（支持enum/数组元素/嵌套对象）
func convertMCPProperty(prop map[string]any) (*schema.ParameterInfo, error) {
	typ, _ := prop["type"].(string)
	desc, _ := prop["description"].(string)
	info := &schema.ParameterInfo{
		Type: schema.DataType(typ),
		Desc: desc,
	}

	if rawEnum, ok := prop["enum"].([]any); ok {
		for _, e := range rawEnum {
			if s, ok := e.(string); ok {
				info.Enum = append(info.Enum, s)
			}
		}
	}

	switch schema.DataType(typ) {
	case schema.Array:
		if rawItems, ok := prop["items"].(map[string]any); ok {
			elem, err := convertMCPProperty(rawItems)
			if err != nil {
				return nil, err
			}
			info.ElemInfo = elem
		}
	case schema.Object:
		rawProps, _ := prop["properties"].(map[string]any)
		rawRequired, _ := prop["required"].([]any)
		subRequired := make([]string, 0, len(rawRequired))
		for _, r := range rawRequired {
			if s, ok := r.(string); ok {
				subRequired = append(subRequired, s)
			}
		}
		sub, err := convertMCPProperties(rawProps, subRequired)
		if err != nil {
			return nil, err
		}
		info.SubParams = sub
	}
	return info, nil
}

// GetModelType 获取模型类型
func (m *MCPModel) GetModelType() string { return "3" }

// Close 关闭MCP客户端
func (m *MCPModel) Close() {
	if m.mcpClient != nil {
		m.mcpClient.Close()
	}
}
