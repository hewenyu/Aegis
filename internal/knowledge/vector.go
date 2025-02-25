package knowledge

import (
	"context"
	"errors"
	"math"
	"sort"
	"sync"
)

// inMemoryVectorStore 是VectorStore接口的内存实现
type inMemoryVectorStore struct {
	vectors    map[string][]float32
	dimensions int
	embedder   Embedder
	mu         sync.RWMutex
}

// NewInMemoryVectorStore 创建一个新的内存向量存储
func NewInMemoryVectorStore(dimensions int, embedder Embedder) VectorStore {
	return &inMemoryVectorStore{
		vectors:    make(map[string][]float32),
		dimensions: dimensions,
		embedder:   embedder,
	}
}

// Embed 将内容转换为向量
func (s *inMemoryVectorStore) Embed(ctx context.Context, content interface{}) ([]float32, error) {
	if s.embedder == nil {
		return nil, errors.New("embedder not available")
	}
	return s.embedder.Embed(ctx, content)
}

// Add 添加向量到存储
func (s *inMemoryVectorStore) Add(ctx context.Context, id string, vector []float32) error {
	if len(vector) != s.dimensions {
		return errors.New("vector dimensions mismatch")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 存储向量的副本
	vectorCopy := make([]float32, len(vector))
	copy(vectorCopy, vector)
	s.vectors[id] = vectorCopy

	return nil
}

// Update 更新存储中的向量
func (s *inMemoryVectorStore) Update(ctx context.Context, id string, vector []float32) error {
	if len(vector) != s.dimensions {
		return errors.New("vector dimensions mismatch")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.vectors[id]; !exists {
		return errors.New("vector not found")
	}

	// 存储向量的副本
	vectorCopy := make([]float32, len(vector))
	copy(vectorCopy, vector)
	s.vectors[id] = vectorCopy

	return nil
}

// Delete 从存储中删除向量
func (s *inMemoryVectorStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.vectors[id]; !exists {
		return errors.New("vector not found")
	}

	delete(s.vectors, id)
	return nil
}

// Search 搜索相似向量
func (s *inMemoryVectorStore) Search(ctx context.Context, vector []float32, limit int) ([]string, []float32, error) {
	if len(vector) != s.dimensions {
		return nil, nil, errors.New("vector dimensions mismatch")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.vectors) == 0 {
		return []string{}, []float32{}, nil
	}

	// 计算所有向量的相似度
	type result struct {
		id    string
		score float32
	}
	results := make([]result, 0, len(s.vectors))

	for id, v := range s.vectors {
		score := cosineSimilarity(vector, v)
		results = append(results, result{id: id, score: score})
	}

	// 按相似度排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// 限制结果数量
	if limit > len(results) {
		limit = len(results)
	}
	results = results[:limit]

	// 提取ID和分数
	ids := make([]string, limit)
	scores := make([]float32, limit)
	for i, r := range results {
		ids[i] = r.id
		scores[i] = r.score
	}

	return ids, scores, nil
}

// Embedder 接口定义了将内容转换为向量的操作
type Embedder interface {
	// Embed 将内容转换为向量
	Embed(ctx context.Context, content interface{}) ([]float32, error)
}

// 辅助函数

// cosineSimilarity 计算两个向量的余弦相似度
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float32
	var normA float32
	var normB float32

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / float32(math.Sqrt(float64(normA))*math.Sqrt(float64(normB)))
}

// mockEmbedder 是Embedder接口的模拟实现，用于测试
type mockEmbedder struct {
	dimensions int
}

// NewMockEmbedder 创建一个新的模拟嵌入器
func NewMockEmbedder(dimensions int) Embedder {
	return &mockEmbedder{dimensions: dimensions}
}

// Embed 将内容转换为向量
func (e *mockEmbedder) Embed(ctx context.Context, content interface{}) ([]float32, error) {
	// 简单地将内容转换为字符串，然后基于字符串生成向量
	var str string
	switch v := content.(type) {
	case string:
		str = v
	default:
		str = "default"
	}

	// 生成一个简单的向量
	vector := make([]float32, e.dimensions)
	for i := 0; i < e.dimensions; i++ {
		if i < len(str) {
			vector[i] = float32(str[i%len(str)]) / 255.0
		} else {
			vector[i] = 0
		}
	}

	// 归一化向量
	var sum float32
	for _, v := range vector {
		sum += v * v
	}
	norm := float32(math.Sqrt(float64(sum)))
	if norm > 0 {
		for i := range vector {
			vector[i] /= norm
		}
	}

	return vector, nil
}
