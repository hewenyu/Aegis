package knowledge

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"

	"github.com/philippgille/chromem-go"
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

// DefaultVectorStoreConfig 返回默认向量存储配置
func DefaultVectorStoreConfig() VectorStoreConfig {
	return VectorStoreConfig{
		Persistent:        false,
		StoragePath:       "./data/vector",
		Compress:          true,
		DefaultCollection: "default",
		EmbeddingModel: EmbeddingModelConfig{
			Provider: "ollama",
			ModelID:  "llama3",
			BaseURL:  "http://localhost:11434",
		},
	}
}

// chromaVectorStore 是VectorStore接口的ChromaDB实现
type chromaVectorStore struct {
	config      VectorStoreConfig
	db          *chromem.DB
	embedFunc   chromem.EmbeddingFunc
	collections map[string]*chromem.Collection
	mu          sync.RWMutex
}

// NewChromaVectorStore 创建新的ChromaDB向量存储
func NewChromaVectorStore(config VectorStoreConfig) (VectorStore, error) {
	var db *chromem.DB
	var err error

	// 初始化DB
	if config.Persistent {
		db, err = chromem.NewPersistentDB(config.StoragePath, config.Compress)
		if err != nil {
			return nil, fmt.Errorf("failed to create persistent vector DB: %w", err)
		}
	} else {
		db = chromem.NewDB()
	}

	// 创建嵌入函数
	embedFunc, err := createEmbeddingFunc(config.EmbeddingModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding function: %w", err)
	}

	store := &chromaVectorStore{
		config:      config,
		db:          db,
		embedFunc:   embedFunc,
		collections: make(map[string]*chromem.Collection),
		mu:          sync.RWMutex{},
	}

	// 创建默认集合
	if config.DefaultCollection != "" {
		_, err := store.getOrCreateCollection(config.DefaultCollection)
		if err != nil {
			return nil, fmt.Errorf("failed to create default collection: %w", err)
		}
	}

	return store, nil
}

// 创建嵌入函数
func createEmbeddingFunc(config EmbeddingModelConfig) (chromem.EmbeddingFunc, error) {
	switch strings.ToLower(config.Provider) {
	case "ollama":
		return chromem.NewEmbeddingFuncOllama(config.ModelID, config.BaseURL), nil
	case "openai":
		if config.APIKey == "" {
			return nil, fmt.Errorf("API key is required for OpenAI")
		}
		model := chromem.EmbeddingModelOpenAI(config.ModelID)
		if config.ModelID == "" {
			model = chromem.EmbeddingModelOpenAI3Small
		}
		return chromem.NewEmbeddingFuncOpenAI(config.APIKey, model), nil
	case "cohere":
		if config.APIKey == "" {
			return nil, fmt.Errorf("API key is required for Cohere")
		}
		model := chromem.EmbeddingModelCohere(config.ModelID)
		if config.ModelID == "" {
			model = chromem.EmbeddingModelCohereMultilingualV2
		}
		return chromem.NewEmbeddingFuncCohere(config.APIKey, model), nil
	case "mistral":
		if config.APIKey == "" {
			return nil, fmt.Errorf("API key is required for Mistral")
		}
		return chromem.NewEmbeddingFuncMistral(config.APIKey), nil
	case "local":
		if config.ModelID == "" {
			return nil, fmt.Errorf("model ID is required for LocalAI")
		}
		if config.BaseURL == "" {
			config.BaseURL = "http://localhost:8080"
		}
		return chromem.NewEmbeddingFuncOpenAICompat(config.BaseURL, config.APIKey, config.ModelID, nil), nil
	default:
		// 默认使用Ollama
		return chromem.NewEmbeddingFuncOllama("llama3", "http://localhost:11434"), nil
	}
}

// Add 添加文档到向量存储
func (v *chromaVectorStore) Add(ctx context.Context, collectionName string, documents []Document) error {
	collection, err := v.getOrCreateCollection(collectionName)
	if err != nil {
		return err
	}

	// 创建chromem文档数组
	chromaDocs := make([]chromem.Document, 0, len(documents))

	for _, doc := range documents {
		// 转换元数据
		metadata := make(map[string]string)
		for k, val := range doc.Metadata {
			// 将值转换为字符串
			switch v := val.(type) {
			case string:
				metadata[k] = v
			case int, int32, int64, float32, float64, bool:
				metadata[k] = fmt.Sprintf("%v", v)
			default:
				// 跳过无法转换的类型
				continue
			}
		}

		// 如果文档已经有向量，使用现有向量创建Document
		var chromaDoc chromem.Document
		var docErr error

		if doc.Vector != nil && len(doc.Vector) > 0 {
			chromaDoc, docErr = chromem.NewDocument(ctx, doc.ID, metadata, doc.Vector, doc.Content, v.embedFunc)
		} else {
			// 否则让chromem计算向量
			chromaDoc, docErr = chromem.NewDocument(ctx, doc.ID, metadata, nil, doc.Content, v.embedFunc)
		}

		if docErr != nil {
			// 记录错误但继续处理其他文档
			continue
		}

		chromaDocs = append(chromaDocs, chromaDoc)
	}

	// 批量添加文档
	if len(chromaDocs) > 0 {
		err = collection.AddDocuments(ctx, chromaDocs, 4) // 使用4个并发处理
		if err != nil {
			return fmt.Errorf("failed to add documents to collection: %w", err)
		}
	}

	return nil
}

// Search 在向量存储中搜索相似文档
func (v *chromaVectorStore) Search(ctx context.Context, collectionName, query string, limit int) ([]SearchResult, error) {
	collection, err := v.getCollection(collectionName)
	if err != nil {
		return nil, err
	}

	// 执行查询
	results, err := collection.Query(ctx, query, limit, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query collection: %w", err)
	}

	// 转换结果
	searchResults := make([]SearchResult, len(results))
	for i, result := range results {
		// 转换元数据
		metadata := make(map[string]interface{})
		for k, v := range result.Metadata {
			metadata[k] = v
		}

		searchResults[i] = SearchResult{
			DocumentID: result.ID,
			Content:    result.Content,
			Metadata:   metadata,
			Distance:   1.0 - float64(result.Similarity), // 从相似度转换为距离
			Similarity: float64(result.Similarity),
		}
	}

	return searchResults, nil
}

// Delete 从向量存储中删除文档
func (v *chromaVectorStore) Delete(ctx context.Context, collectionName string, documentIDs []string) error {
	collection, err := v.getCollection(collectionName)
	if err != nil {
		return err
	}

	// 删除文档
	err = collection.Delete(ctx, nil, nil, documentIDs...)
	if err != nil {
		return fmt.Errorf("failed to delete documents: %w", err)
	}

	return nil
}

// ListCollections 列出所有集合
func (v *chromaVectorStore) ListCollections(ctx context.Context) ([]string, error) {
	// Chromem的API没有context参数，忽略ctx
	collections := v.db.ListCollections()

	// 转换为字符串数组
	result := make([]string, 0, len(collections))
	for name := range collections {
		result = append(result, name)
	}

	return result, nil
}

// Close 关闭向量存储
func (v *chromaVectorStore) Close() error {
	// ChromaDB目前不需要显式关闭
	return nil
}

// 获取已存在的集合
func (v *chromaVectorStore) getCollection(name string) (*chromem.Collection, error) {
	v.mu.RLock()
	collection, exists := v.collections[name]
	v.mu.RUnlock()

	if exists {
		return collection, nil
	}

	// 尝试从数据库加载
	collections := v.db.ListCollections()
	if coll, exists := collections[name]; exists {
		v.mu.Lock()
		v.collections[name] = coll
		v.mu.Unlock()
		return coll, nil
	}

	return nil, fmt.Errorf("collection %s not found", name)
}

// 获取或创建集合
func (v *chromaVectorStore) getOrCreateCollection(name string) (*chromem.Collection, error) {
	// 先尝试获取
	collection, err := v.getCollection(name)
	if err == nil {
		return collection, nil
	}

	// 创建新集合
	v.mu.Lock()
	defer v.mu.Unlock()

	// 再次检查，避免并发问题
	collections := v.db.ListCollections()
	if coll, exists := collections[name]; exists {
		v.collections[name] = coll
		return coll, nil
	}

	// 创建集合
	coll, err := v.db.GetOrCreateCollection(name, nil, v.embedFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection %s: %w", name, err)
	}

	v.collections[name] = coll
	return coll, nil
}

// inMemoryVectorStore 是一个使用内存存储向量的简单实现
type inMemoryVectorStore struct {
	vectors  map[string][]Document // 按集合名称组织的文档集合
	embedder Embedder
	mu       sync.RWMutex
}

// NewInMemoryVectorStore 创建一个内存向量存储
func NewInMemoryVectorStore(embedder Embedder) VectorStore {
	return &inMemoryVectorStore{
		vectors:  make(map[string][]Document),
		embedder: embedder,
		mu:       sync.RWMutex{},
	}
}

// Add 添加文档到向量存储
func (s *inMemoryVectorStore) Add(ctx context.Context, collectionName string, documents []Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 确保集合存在
	if _, exists := s.vectors[collectionName]; !exists {
		s.vectors[collectionName] = make([]Document, 0)
	}

	// 添加或更新文档
	for _, doc := range documents {
		// 如果文档没有向量，计算向量
		if doc.Vector == nil || len(doc.Vector) == 0 {
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
func (s *inMemoryVectorStore) Search(ctx context.Context, collectionName, query string, limit int) ([]SearchResult, error) {
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
		doc        Document
		similarity float64
	}

	results := make([]result, 0, len(docs))
	for _, doc := range docs {
		if doc.Vector == nil || len(doc.Vector) == 0 {
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
	searchResults := make([]SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = SearchResult{
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
	newDocs := make([]Document, 0, len(docs)-len(documentIDs))
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

// MockEmbedder 是一个用于测试的模拟嵌入器
type MockEmbedder struct {
	dimensions int
}

// NewMockEmbedder 创建一个新的模拟嵌入器
func NewMockEmbedder(dimensions int) Embedder {
	return &MockEmbedder{dimensions: dimensions}
}

// Embed 生成一个随机向量，用于测试
func (e *MockEmbedder) Embed(ctx context.Context, content interface{}) ([]float32, error) {
	// 将内容字符串化并转换为简单向量
	var str string
	switch c := content.(type) {
	case string:
		str = c
	default:
		str = fmt.Sprintf("%v", c)
	}

	// 创建确定性但随机分布的向量
	vector := make([]float32, e.dimensions)
	for i := 0; i < e.dimensions; i++ {
		// 使用字符串和索引生成伪随机数
		hash := int(str[i%len(str)]) + i
		vector[i] = float32(hash%100) / 100.0
	}

	// 归一化向量
	var norm float32
	for _, v := range vector {
		norm += v * v
	}
	norm = float32(math.Sqrt(float64(norm)))

	if norm > 0 {
		for i := range vector {
			vector[i] /= norm
		}
	}

	return vector, nil
}

// Embedder 接口定义了嵌入器的行为
type Embedder interface {
	// Embed 将内容转换为向量
	Embed(ctx context.Context, content interface{}) ([]float32, error)
}

// cosineSimilarity 计算两个向量之间的余弦相似度
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}
