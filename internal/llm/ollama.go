package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/ollama/ollama/api"
)

// OllamaConfig 定义Ollama配置选项
type OllamaConfig struct {
	// Ollama服务的URL地址
	BaseURL string `json:"base_url"`
	// 超时时间（秒）
	Timeout int `json:"timeout"`
}

// DefaultOllamaConfig 返回默认Ollama配置
func DefaultOllamaConfig() OllamaConfig {
	return OllamaConfig{
		BaseURL: "http://localhost:11434",
		Timeout: 30,
	}
}

// ollamaProvider 是Ollama提供者的实现
type ollamaProvider struct {
	config   OllamaConfig
	client   *api.Client
	models   map[string]ModelInfo
	modelsMu sync.RWMutex
}

// NewOllamaProvider 创建一个新的Ollama提供者
func NewOllamaProvider(config OllamaConfig) Provider {
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:11434"
	}
	if config.Timeout <= 0 {
		config.Timeout = 30
	}

	baseURL, err := url.Parse(config.BaseURL)
	if err != nil {
		// 回退到默认 URL
		baseURL, _ = url.Parse("http://localhost:11434")
	}

	httpClient := &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Second,
	}

	client := api.NewClient(baseURL, httpClient)

	return &ollamaProvider{
		config: config,
		client: client,
		models: make(map[string]ModelInfo),
	}
}

// Name 返回提供者名称
func (p *ollamaProvider) Name() string {
	return "ollama"
}

// ListModels 获取可用模型列表
func (p *ollamaProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	resp, err := p.client.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	// 转换为通用模型信息
	p.modelsMu.Lock()
	defer p.modelsMu.Unlock()

	models := make([]ModelInfo, 0, len(resp.Models))
	for _, model := range resp.Models {
		modelInfo := ModelInfo{
			ID:           model.Name,
			Name:         model.Name,
			Provider:     p.Name(),
			Capabilities: []string{"chat", "completion", "embedding"},
			MaxTokens:    4096, // 一个合理的默认值，实际上应该根据模型规格确定
			Metadata: map[string]interface{}{
				"size":        model.Size,
				"modified_at": model.ModifiedAt,
				"details":     model.Details,
			},
		}
		models = append(models, modelInfo)
		p.models[model.Name] = modelInfo
	}

	return models, nil
}

// GetModel 获取指定模型信息
func (p *ollamaProvider) GetModel(ctx context.Context, modelID string) (ModelInfo, error) {
	// 先从缓存中查找
	p.modelsMu.RLock()
	modelInfo, exists := p.models[modelID]
	p.modelsMu.RUnlock()

	if exists {
		return modelInfo, nil
	}

	// 如果缓存中不存在，重新获取所有模型并缓存
	_, err := p.ListModels(ctx)
	if err != nil {
		return ModelInfo{}, err
	}

	p.modelsMu.RLock()
	defer p.modelsMu.RUnlock()

	modelInfo, exists = p.models[modelID]
	if !exists {
		return ModelInfo{}, fmt.Errorf("model %s not found", modelID)
	}

	return modelInfo, nil
}

// Complete 执行文本补全
func (p *ollamaProvider) Complete(ctx context.Context, modelID string, request CompletionRequest) (CompletionResponse, error) {
	if modelID == "" {
		return CompletionResponse{}, fmt.Errorf("model ID cannot be empty")
	}

	// 构建Ollama请求
	options := map[string]interface{}{}
	if request.Temperature > 0 {
		options["temperature"] = request.Temperature
	}
	if request.TopP > 0 {
		options["top_p"] = request.TopP
	}
	if len(request.Stop) > 0 {
		options["stop"] = request.Stop
	}

	// 设置 stream 为 false
	stream := false

	generateReq := &api.GenerateRequest{
		Model:   modelID,
		Prompt:  request.Prompt,
		Stream:  &stream,
		Options: options,
	}

	var finalResp api.GenerateResponse
	err := p.client.Generate(ctx, generateReq, func(resp api.GenerateResponse) error {
		finalResp = resp
		return nil
	})

	if err != nil {
		return CompletionResponse{}, fmt.Errorf("failed to complete text: %w", err)
	}

	// 计算毫秒级持续时间
	durationMs := finalResp.TotalDuration.Milliseconds()

	// 构建响应
	return CompletionResponse{
		Text: finalResp.Response,
		Usage: Usage{
			PromptTokens:     finalResp.PromptEvalCount,
			CompletionTokens: finalResp.EvalCount,
			TotalTokens:      finalResp.PromptEvalCount + finalResp.EvalCount,
		},
		Metadata: map[string]interface{}{
			"model":             finalResp.Model,
			"created_at":        finalResp.CreatedAt,
			"total_duration_ms": durationMs,
			"eval_count":        finalResp.EvalCount,
		},
		Timestamp: time.Now().Unix(),
	}, nil
}

// Chat 执行聊天补全
func (p *ollamaProvider) Chat(ctx context.Context, modelID string, request ChatRequest) (ChatResponse, error) {
	if modelID == "" {
		return ChatResponse{}, fmt.Errorf("model ID cannot be empty")
	}

	if len(request.Messages) == 0 {
		return ChatResponse{}, fmt.Errorf("messages cannot be empty")
	}

	// 转换消息格式
	messages := make([]api.Message, len(request.Messages))
	for i, msg := range request.Messages {
		messages[i] = api.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// 添加可选参数
	options := map[string]interface{}{}
	if request.Temperature > 0 {
		options["temperature"] = request.Temperature
	}
	if request.TopP > 0 {
		options["top_p"] = request.TopP
	}
	if len(request.Stop) > 0 {
		options["stop"] = request.Stop
	}

	// 设置 stream 为 false
	stream := false

	chatReq := &api.ChatRequest{
		Model:    modelID,
		Messages: messages,
		Stream:   &stream,
		Options:  options,
	}

	var finalResp api.ChatResponse
	err := p.client.Chat(ctx, chatReq, func(resp api.ChatResponse) error {
		finalResp = resp
		return nil
	})

	if err != nil {
		return ChatResponse{}, fmt.Errorf("failed to complete chat: %w", err)
	}

	// 计算毫秒级持续时间
	durationMs := finalResp.TotalDuration.Milliseconds()

	// 构建响应
	return ChatResponse{
		Message: Message{
			Role:    finalResp.Message.Role,
			Content: finalResp.Message.Content,
		},
		Usage: Usage{
			PromptTokens:     finalResp.PromptEvalCount,
			CompletionTokens: finalResp.EvalCount,
			TotalTokens:      finalResp.PromptEvalCount + finalResp.EvalCount,
		},
		Metadata: map[string]interface{}{
			"model":             finalResp.Model,
			"created_at":        finalResp.CreatedAt,
			"total_duration_ms": durationMs,
			"eval_count":        finalResp.EvalCount,
		},
		Timestamp: time.Now().Unix(),
	}, nil
}

// Embed 执行文本嵌入
func (p *ollamaProvider) Embed(ctx context.Context, modelID string, request EmbeddingRequest) (EmbeddingResponse, error) {
	if modelID == "" {
		return EmbeddingResponse{}, fmt.Errorf("model ID cannot be empty")
	}

	embedReq := &api.EmbeddingRequest{
		Model:  modelID,
		Prompt: request.Input,
	}

	resp, err := p.client.Embeddings(ctx, embedReq)
	if err != nil {
		return EmbeddingResponse{}, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// 构建响应
	return EmbeddingResponse{
		Embedding: resp.Embedding,
		Usage: Usage{
			PromptTokens:     len(request.Input) / 4, // 粗略估算
			CompletionTokens: 0,
			TotalTokens:      len(request.Input) / 4,
		},
		Metadata: map[string]interface{}{
			"model":      modelID,
			"dimensions": len(resp.Embedding),
		},
	}, nil
}
