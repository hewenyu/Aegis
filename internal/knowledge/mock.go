package knowledge

import (
	"context"
	"fmt"
	"math"
)

// MockEmbedder 是一个用于测试的模拟嵌入器
type MockEmbedder struct {
	dimensions int
}

// NewMockEmbedder 创建一个新的模拟嵌入器
func NewMockEmbedder(dimensions int) Embedder {
	return &MockEmbedder{dimensions: dimensions}
}

// Embed 生成一个随机向量，用于测试
func (e *MockEmbedder) Embed(ctx context.Context, content interface{}) ([]float64, error) {
	// 将内容字符串化并转换为简单向量
	var str string
	switch c := content.(type) {
	case string:
		str = c
	default:
		str = fmt.Sprintf("%v", c)
	}

	// 创建确定性但随机分布的向量
	vector := make([]float64, e.dimensions)
	for i := 0; i < e.dimensions; i++ {
		// 使用字符串和索引生成伪随机数
		hash := int(str[i%len(str)]) + i
		vector[i] = float64(hash%100) / 100.0
	}

	// 归一化向量
	var norm float64
	for _, v := range vector {
		norm += v * v
	}
	norm = math.Sqrt(norm)

	if norm > 0 {
		for i := range vector {
			vector[i] /= norm
		}
	}

	return vector, nil
}
