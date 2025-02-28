package llm

import (
	"context"

	"github.com/hewenyu/Aegis/internal/types"
)

// Service 表示LLM服务接口
type Service interface {
	// 注册LLM提供者
	RegisterProvider(provider types.Provider) error

	// 获取LLM提供者
	GetProvider(name string) (types.Provider, error)

	// 列出所有可用的LLM提供者
	ListProviders() []string

	// 获取所有可用模型
	ListModels(ctx context.Context) (map[string][]types.ModelInfo, error)

	// 获取模型信息
	GetModel(ctx context.Context, providerName, modelID string) (types.ModelInfo, error)

	// 执行文本补全
	Complete(ctx context.Context, providerName, modelID string, request types.CompletionRequest) (types.CompletionResponse, error)

	// 执行聊天补全
	Chat(ctx context.Context, providerName, modelID string, request types.ChatRequest) (types.ChatResponse, error)

	// 执行文本嵌入
	Embed(ctx context.Context, providerName, modelID string, request types.EmbeddingRequest) (types.EmbeddingResponse, error)
}
