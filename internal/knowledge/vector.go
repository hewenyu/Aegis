package knowledge

import (
	"context"
	"math"
)

// VectorStoreConfig 向量存储配置
type VectorStoreConfig struct {
	// 是否持久化存储
	Persistent bool `json:"persistent"`
	// 持久化存储路径
	StoragePath string `json:"storage_path"`
	// 是否压缩存储
	Compress bool `json:"compress"`
	// 默认集合名称
	DefaultCollection string `json:"default_collection"`
	// 嵌入模型配置
	EmbeddingModel EmbeddingModelConfig `json:"embedding_model"`
}

// EmbeddingModelConfig 嵌入模型配置
type EmbeddingModelConfig struct {
	// 提供商类型: "ollama", "openai", "cohere" 等
	Provider string `json:"provider"`
	// 模型名称
	ModelID string `json:"model_id"`
	// API Key (如果需要)
	APIKey string `json:"api_key"`
	// 基础URL (如果需要)
	BaseURL string `json:"base_url"`
}

// Embedder 接口定义了嵌入器的行为
type Embedder interface {
	// Embed 将内容转换为向量
	Embed(ctx context.Context, content interface{}) ([]float64, error)
}

// cosineSimilarity 计算两个向量之间的余弦相似度
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
