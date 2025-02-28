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

// GenerateTextParams 定义生成文本的参数
type GenerateTextParams struct {
	Prompt          string                // 提示文本
	SystemPrompt    string                // 系统提示文本
	Model           string                // 使用的模型名称
	Temperature     float64               // 生成的随机度 (0.0-1.0)
	MaxTokens       int                   // 最大生成token数
	TopP            float64               // 核采样概率阈值
	Stop            []string              // 停止生成的标记
	HistoryMessages []ChatMessage         // 聊天历史消息
	Attachments     []ProcessedAttachment // 处理后的附件
}

// ChatMessage 表示聊天历史中的单条消息
type ChatMessage struct {
	Role    string // 角色 (system, user, assistant)
	Content string // 消息内容
}

// Attachment 表示多模态输入中的附件
type Attachment struct {
	Type     string // 附件类型 (image, audio, etc)
	Data     []byte // 附件数据
	MimeType string // MIME类型
	FileName string // 文件名
}

// ProcessedAttachment 表示经过处理的附件
type ProcessedAttachment struct {
	Type       string      // 附件类型
	Data       interface{} // 处理后的数据
	SourceName string      // 源文件名
}

// ModelInfo 包含LLM模型的详细信息
type ModelInfo struct {
	Name                  string  // 模型名称
	ContextWindowSize     int     // 上下文窗口大小（token数）
	MaxOutputTokens       int     // 最大输出token数
	SupportsImageInput    bool    // 是否支持图像输入
	SupportsAudioInput    bool    // 是否支持音频输入
	SupportsVisionOutput  bool    // 是否支持视觉输出
	PricingPerInputToken  float64 // 输入token的定价
	PricingPerOutputToken float64 // 输出token的定价
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

	// GetEmbedModel 获取嵌入模型
	GetEmbedModel() string
}
