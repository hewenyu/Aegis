package knowledge

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/hewenyu/Aegis/internal/types"
)

// inMemoryVectorStore 是一个使用内存存储向量的简单实现
type inMemoryVectorStore struct {
	vectors  map[string][]types.Document // 按集合名称组织的文档集合
	embedder Embedder
	mu       sync.RWMutex
}

// NewInMemoryVectorStore 创建一个内存向量存储
func NewInMemoryVectorStore(embedder Embedder) types.VectorStore {
	return &inMemoryVectorStore{
		vectors:  make(map[string][]types.Document),
		embedder: embedder,
		mu:       sync.RWMutex{},
	}
}

// Add 添加文档到向量存储
func (s *inMemoryVectorStore) Add(ctx context.Context, collectionName string, documents []types.Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 确保集合存在
	if _, exists := s.vectors[collectionName]; !exists {
		s.vectors[collectionName] = make([]types.Document, 0)
	}

	// 添加或更新文档
	for _, doc := range documents {
		// 如果文档没有向量，计算向量
		if doc.Vector == nil {
			vec, err := s.embedder.Embed(ctx, doc.Content)
			if err != nil {
				continue // 忽略错误，继续处理其他文档
			}
			doc.Vector = vec
		}

		// 寻找并替换现有文档，或者添加新文档
		found := false
		for i, existingDoc := range s.vectors[collectionName] {
			if existingDoc.ID == doc.ID {
				s.vectors[collectionName][i] = doc
				found = true
				break
			}
		}

		if !found {
			s.vectors[collectionName] = append(s.vectors[collectionName], doc)
		}
	}

	return nil
}

// Search 在向量存储中搜索相似文档
func (s *inMemoryVectorStore) Search(ctx context.Context, collectionName, query string, limit int) ([]types.SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 确保集合存在
	docs, exists := s.vectors[collectionName]
	if !exists {
		return nil, fmt.Errorf("collection %s not found", collectionName)
	}

	// 对查询进行向量化
	queryVector, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// 计算每个文档与查询的相似度
	type result struct {
		doc        types.Document
		similarity float64
	}

	results := make([]result, 0, len(docs))
	for _, doc := range docs {
		if doc.Vector == nil {
			continue
		}

		// 计算余弦相似度
		sim := cosineSimilarity(queryVector, doc.Vector)
		results = append(results, result{
			doc:        doc,
			similarity: float64(sim),
		})
	}

	// 按相似度排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].similarity > results[j].similarity
	})

	// 限制结果数量
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	// 构建响应
	searchResults := make([]types.SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = types.SearchResult{
			DocumentID: r.doc.ID,
			Content:    r.doc.Content,
			Metadata:   r.doc.Metadata,
			Distance:   1.0 - r.similarity,
			Similarity: r.similarity,
		}
	}

	return searchResults, nil
}

// Delete 从向量存储中删除文档
func (s *inMemoryVectorStore) Delete(ctx context.Context, collectionName string, documentIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 确保集合存在
	docs, exists := s.vectors[collectionName]
	if !exists {
		return fmt.Errorf("collection %s not found", collectionName)
	}

	// 创建ID集合用于快速查找
	idSet := make(map[string]struct{}, len(documentIDs))
	for _, id := range documentIDs {
		idSet[id] = struct{}{}
	}

	// 过滤掉要删除的文档
	newDocs := make([]types.Document, 0, len(docs)-len(documentIDs))
	for _, doc := range docs {
		if _, shouldDelete := idSet[doc.ID]; !shouldDelete {
			newDocs = append(newDocs, doc)
		}
	}

	s.vectors[collectionName] = newDocs
	return nil
}

// ListCollections 列出所有集合
func (s *inMemoryVectorStore) ListCollections(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	collections := make([]string, 0, len(s.vectors))
	for name := range s.vectors {
		collections = append(collections, name)
	}

	return collections, nil
}

// Close 关闭向量存储
func (s *inMemoryVectorStore) Close() error {
	// 内存实现无需特殊关闭操作
	return nil
}
