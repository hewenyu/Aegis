package llm

import (
	"context"
	"fmt"

	"github.com/hewenyu/Aegis/internal/types"
)

// LLMAdapter 实现了 text.LLM 和 text.Embedder 接口
type LLMAdapter struct {
	provider types.Provider
	model    string
}

// NewOllamaAdapter 创建新的 Ollama 适配器
func NewLLMAdapter(provider types.Provider, model string) *LLMAdapter {
	return &LLMAdapter{
		provider: provider,
		model:    model,
	}
}

// Complete 实现 text.LLM 接口
func (a *LLMAdapter) Complete(ctx context.Context, prompt string) (string, error) {

	request := types.CompletionRequest{
		Prompt:      prompt,
		MaxTokens:   2000,
		Temperature: 0.7,
	}

	response, err := a.provider.Complete(ctx, a.model, request)
	if err != nil {
		return "", fmt.Errorf("failed to complete text: %w", err)
	}

	return response.Text, nil
}

// Embed 实现 text.Embedder 接口
func (a *LLMAdapter) Embed(ctx context.Context, text string) ([]float32, error) {
	request := types.EmbeddingRequest{
		Input: text,
	}

	response, err := a.provider.Embed(ctx, a.model, request)

	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// 将 float64 转换为 float32
	embedding := make([]float32, len(response.Embedding))
	for i, v := range response.Embedding {
		embedding[i] = float32(v)
	}

	return embedding, nil
}
