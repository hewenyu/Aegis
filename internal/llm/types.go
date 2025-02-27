package llm

import (
	"context"
	"errors"
)

// 定义错误
var (
	ErrLLMNotAvailable = errors.New("llm service not available")
	ErrInvalidRequest  = errors.New("invalid llm request")
	ErrRequestTimeout  = errors.New("llm request timed out")
	ErrRateLimited     = errors.New("llm rate limit exceeded")
)

// Message 表示一条消息
type Message struct {
	Role    string                 `json:"role"`
	Content string                 `json:"content"`
	Name    string                 `json:"name,omitempty"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// CompletionRequest 表示完成请求
type CompletionRequest struct {
	Prompt           string                 `json:"prompt"`
	MaxTokens        int                    `json:"max_tokens,omitempty"`
	Temperature      float64                `json:"temperature,omitempty"`
	TopP             float64                `json:"top_p,omitempty"`
	FrequencyPenalty float64                `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64                `json:"presence_penalty,omitempty"`
	Stop             []string               `json:"stop,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// ChatRequest 表示聊天请求
type ChatRequest struct {
	Messages         []Message              `json:"messages"`
	MaxTokens        int                    `json:"max_tokens,omitempty"`
	Temperature      float64                `json:"temperature,omitempty"`
	TopP             float64                `json:"top_p,omitempty"`
	FrequencyPenalty float64                `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64                `json:"presence_penalty,omitempty"`
	Stop             []string               `json:"stop,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// EmbeddingRequest 表示嵌入请求
type EmbeddingRequest struct {
	Input    string                 `json:"input"`
	Model    string                 `json:"model,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// CompletionResponse 表示完成响应
type CompletionResponse struct {
	Text      string                 `json:"text"`
	Usage     Usage                  `json:"usage"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

// ChatResponse 表示聊天响应
type ChatResponse struct {
	Message   Message                `json:"message"`
	Usage     Usage                  `json:"usage"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

// EmbeddingResponse 表示嵌入响应
type EmbeddingResponse struct {
	Embedding []float64              `json:"embedding"`
	Usage     Usage                  `json:"usage"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Usage 表示API使用情况
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ModelInfo 表示LLM模型信息
type ModelInfo struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Provider     string                 `json:"provider"`
	Capabilities []string               `json:"capabilities"`
	MaxTokens    int                    `json:"max_tokens"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Provider 表示LLM服务提供者接口
type Provider interface {
	// 获取提供者名称
	Name() string

	// 获取可用模型列表
	ListModels(ctx context.Context) ([]ModelInfo, error)

	// 获取指定模型信息
	GetModel(ctx context.Context, modelID string) (ModelInfo, error)

	// 文本补全
	Complete(ctx context.Context, modelID string, request CompletionRequest) (CompletionResponse, error)

	// 聊天补全
	Chat(ctx context.Context, modelID string, request ChatRequest) (ChatResponse, error)

	// 文本嵌入
	Embed(ctx context.Context, modelID string, request EmbeddingRequest) (EmbeddingResponse, error)
}

// Service 表示LLM服务接口
type Service interface {
	// 注册LLM提供者
	RegisterProvider(provider Provider) error

	// 获取LLM提供者
	GetProvider(name string) (Provider, error)

	// 列出所有可用的LLM提供者
	ListProviders() []string

	// 获取所有可用模型
	ListModels(ctx context.Context) (map[string][]ModelInfo, error)

	// 获取模型信息
	GetModel(ctx context.Context, providerName, modelID string) (ModelInfo, error)

	// 执行文本补全
	Complete(ctx context.Context, providerName, modelID string, request CompletionRequest) (CompletionResponse, error)

	// 执行聊天补全
	Chat(ctx context.Context, providerName, modelID string, request ChatRequest) (ChatResponse, error)

	// 执行文本嵌入
	Embed(ctx context.Context, providerName, modelID string, request EmbeddingRequest) (EmbeddingResponse, error)
}
