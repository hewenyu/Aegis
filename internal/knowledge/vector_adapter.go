package knowledge

import (
	"context"
	"fmt"

	"github.com/hewenyu/Aegis/internal/tool/text"
	"github.com/hewenyu/Aegis/internal/types"
)

// VectorAdapter 实现了 text.VectorStore 接口
type VectorAdapter struct {
	store      types.VectorStore
	collection string
}

// NewVectorAdapter 创建新的向量存储适配器
func NewVectorAdapter(store types.VectorStore, collection string) *VectorAdapter {
	return &VectorAdapter{
		store:      store,
		collection: collection,
	}
}

// Store 实现 text.VectorStore 接口
func (a *VectorAdapter) Store(ctx context.Context, id string, vector []float32, metadata map[string]interface{}) error {
	doc := types.Document{
		ID:       id,
		Vector:   make([]float64, len(vector)),
		Metadata: metadata,
	}

	// 将 float32 转换为 float64
	for i, v := range vector {
		doc.Vector[i] = float64(v)
	}

	return a.store.Add(ctx, a.collection, []types.Document{doc})
}

// Search 实现 text.VectorStore 接口
func (a *VectorAdapter) Search(ctx context.Context, vector []float32, limit int) ([]text.SearchResult, error) {
	// 将向量转换为查询字符串（这里需要根据实际情况调整）
	query := fmt.Sprintf("vector_query:%v", vector)

	results, err := a.store.Search(ctx, a.collection, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search vectors: %w", err)
	}

	// 转换结果
	searchResults := make([]text.SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = text.SearchResult{
			ID:       r.DocumentID,
			Score:    float32(r.Similarity),
			Metadata: r.Metadata,
		}
	}

	return searchResults, nil
}
