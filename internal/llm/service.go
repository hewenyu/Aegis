package llm

import (
	"context"
	"fmt"
	"sync"

	"github.com/hewenyu/Aegis/internal/types"
)

// service 是Service接口的实现
type service struct {
	providers map[string]types.Provider
	mu        sync.RWMutex
}

// NewService 创建一个新的LLM服务
func NewService() Service {
	return &service{
		providers: make(map[string]types.Provider),
	}
}

// RegisterProvider 注册一个LLM提供者
func (s *service) RegisterProvider(provider types.Provider) error {
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	name := provider.Name()
	if name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.providers[name]; exists {
		return fmt.Errorf("provider %s already registered", name)
	}

	s.providers[name] = provider
	return nil
}

// GetProvider 获取指定名称的LLM提供者
func (s *service) GetProvider(name string) (types.Provider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	provider, exists := s.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	return provider, nil
}

// ListProviders 列出所有已注册的LLM提供者
func (s *service) ListProviders() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	providers := make([]string, 0, len(s.providers))
	for name := range s.providers {
		providers = append(providers, name)
	}

	return providers
}

// ListModels 获取所有可用模型
func (s *service) ListModels(ctx context.Context) (map[string][]types.ModelInfo, error) {
	s.mu.RLock()
	providers := make(map[string]types.Provider, len(s.providers))
	for name, provider := range s.providers {
		providers[name] = provider
	}
	s.mu.RUnlock()

	result := make(map[string][]types.ModelInfo)
	for name, provider := range providers {
		models, err := provider.ListModels(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list models for provider %s: %w", name, err)
		}
		result[name] = models
	}

	return result, nil
}

// GetModel 获取模型信息
func (s *service) GetModel(ctx context.Context, providerName, modelID string) (types.ModelInfo, error) {
	provider, err := s.GetProvider(providerName)
	if err != nil {
		return types.ModelInfo{}, err
	}

	return provider.GetModel(ctx, modelID)
}

// Complete 执行文本补全
func (s *service) Complete(ctx context.Context, providerName, modelID string, request types.CompletionRequest) (types.CompletionResponse, error) {
	provider, err := s.GetProvider(providerName)
	if err != nil {
		return types.CompletionResponse{}, err
	}

	return provider.Complete(ctx, modelID, request)
}

// Chat 执行聊天补全
func (s *service) Chat(ctx context.Context, providerName, modelID string, request types.ChatRequest) (types.ChatResponse, error) {
	provider, err := s.GetProvider(providerName)
	if err != nil {
		return types.ChatResponse{}, err
	}

	return provider.Chat(ctx, modelID, request)
}

// Embed 执行文本嵌入
func (s *service) Embed(ctx context.Context, providerName, modelID string, request types.EmbeddingRequest) (types.EmbeddingResponse, error) {
	provider, err := s.GetProvider(providerName)
	if err != nil {
		return types.EmbeddingResponse{}, err
	}

	return provider.Embed(ctx, modelID, request)
}
