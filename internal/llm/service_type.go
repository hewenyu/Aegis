package llm

import "context"

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
