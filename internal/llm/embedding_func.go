package llm

import (
	"context"

	"github.com/philippgille/chromem-go"
)

// NewEmbeddingFunc 返回一个用于生成嵌入向量的函数
func NewEmbeddingFunc(provider Provider) chromem.EmbeddingFunc {
	return func(ctx context.Context, text string) ([]float32, error) {
		response, err := provider.Embed(ctx, provider.GetEmbedModel(), EmbeddingRequest{Input: text})
		if err != nil {
			return nil, err
		}
		// 将[]float64转换为[]float32
		embedding := make([]float32, len(response.Embedding))
		for i, v := range response.Embedding {
			embedding[i] = float32(v)
		}
		return embedding, nil
	}
}
