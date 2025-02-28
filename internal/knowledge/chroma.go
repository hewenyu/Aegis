package knowledge

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/hewenyu/Aegis/internal/llm"
	"github.com/hewenyu/Aegis/internal/types"
	"github.com/philippgille/chromem-go"
)

// chromaVectorStore 是VectorStore接口的ChromaDB实现
type chromaVectorStore struct {
	config      VectorStoreConfig
	db          *chromem.DB
	embedFunc   chromem.EmbeddingFunc
	collections map[string]*chromem.Collection
	mu          sync.RWMutex
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
			ModelID:  "mxbai-embed-large",
			BaseURL:  "http://localhost:11434",
		},
	}
}

// NewChromaVectorStore 创建新的ChromaDB向量存储
func NewChromaVectorStore(config VectorStoreConfig) (types.VectorStore, error) {
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
		baseURL := config.BaseURL
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		provider, err := llm.NewOllamaProvider(baseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to create ollama provider: %w", err)
		}
		return llm.NewEmbeddingFunc(provider), nil
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
		return chromem.NewEmbeddingFuncOllama("llama2:latest", "http://localhost:11434"), nil
	}
}

// Add 添加文档到向量存储
func (v *chromaVectorStore) Add(ctx context.Context, collectionName string, documents []types.Document) error {
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

		// 将float64转换为float32
		vector := make([]float32, len(doc.Vector))
		for i, v := range doc.Vector {
			vector[i] = float32(v)
		}

		if doc.Vector != nil {
			chromaDoc, docErr = chromem.NewDocument(ctx, doc.ID, metadata, vector, doc.Content, v.embedFunc)
		} else {
			// 否则让chromem计算向量
			chromaDoc, docErr = chromem.NewDocument(ctx, doc.ID, metadata, nil, doc.Content, v.embedFunc)
		}

		if docErr != nil {
			// 记录错误但继续处理其他文档
			fmt.Printf("添加文档失败: %v", docErr)
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
func (v *chromaVectorStore) Search(ctx context.Context, collectionName, query string, limit int) ([]types.SearchResult, error) {
	// 使用 getOrCreateCollection 替代 getCollection
	collection, err := v.getOrCreateCollection(collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create collection: %w", err)
	}

	if collection == nil {
		return nil, fmt.Errorf("collection is nil after initialization")
	}

	// 获取集合中的文档数量
	count := collection.Count()
	if count == 0 {
		return []types.SearchResult{}, nil
	}

	// 确保请求的结果数量不超过集合中的文档数量
	if limit > count {
		limit = count
	}

	// 执行查询
	results, err := collection.Query(ctx, query, limit, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query collection: %w", err)
	}

	// 转换结果
	searchResults := make([]types.SearchResult, len(results))
	for i, result := range results {
		metadata := make(map[string]interface{})
		for k, v := range result.Metadata {
			metadata[k] = v
		}

		searchResults[i] = types.SearchResult{
			DocumentID: result.ID,
			Content:    result.Content,
			Metadata:   metadata,
			Distance:   1.0 - float64(result.Similarity),
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
