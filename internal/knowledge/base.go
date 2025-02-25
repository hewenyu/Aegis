package knowledge

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
)

// baseKnowledge 实现了Base接口
type baseKnowledge struct {
	items    sync.Map
	vector   VectorStore
	mu       sync.RWMutex
	contexts map[string]Context
}

// NewBase 创建一个新的知识库
func NewBase(vectorStore VectorStore) Base {
	return &baseKnowledge{
		vector:   vectorStore,
		contexts: make(map[string]Context),
	}
}

// AddKnowledge 添加知识到知识库
func (b *baseKnowledge) AddKnowledge(ctx context.Context, k Knowledge) error {
	if k.ID == "" {
		k.ID = uuid.New().String()
	}

	// 验证知识
	if err := validateKnowledge(k); err != nil {
		return err
	}

	// 如果没有向量，生成向量
	if len(k.Vector) == 0 && b.vector != nil {
		var err error
		k.Vector, err = b.vector.Embed(ctx, k.Content)
		if err != nil {
			return err
		}
	}

	// 存储知识
	b.items.Store(k.ID, k)

	// 如果有向量存储，添加到向量索引
	if b.vector != nil {
		if err := b.vector.Add(ctx, k.ID, k.Vector); err != nil {
			// 如果向量存储失败，回滚
			b.items.Delete(k.ID)
			return err
		}
	}

	return nil
}

// UpdateKnowledge 更新知识库中的知识
func (b *baseKnowledge) UpdateKnowledge(ctx context.Context, id string, k Knowledge) error {
	// 检查知识是否存在
	if _, ok := b.items.Load(id); !ok {
		return ErrKnowledgeNotFound
	}

	// 确保ID一致
	k.ID = id

	// 验证知识
	if err := validateKnowledge(k); err != nil {
		return err
	}

	// 如果没有向量，生成向量
	if len(k.Vector) == 0 && b.vector != nil {
		var err error
		k.Vector, err = b.vector.Embed(ctx, k.Content)
		if err != nil {
			return err
		}
	}

	// 存储知识
	b.items.Store(id, k)

	// 如果有向量存储，更新向量索引
	if b.vector != nil {
		if err := b.vector.Update(ctx, id, k.Vector); err != nil {
			return err
		}
	}

	return nil
}

// DeleteKnowledge 从知识库中删除知识
func (b *baseKnowledge) DeleteKnowledge(ctx context.Context, id string) error {
	// 检查知识是否存在
	if _, ok := b.items.Load(id); !ok {
		return ErrKnowledgeNotFound
	}

	// 删除知识
	b.items.Delete(id)

	// 如果有向量存储，从向量索引中删除
	if b.vector != nil {
		if err := b.vector.Delete(ctx, id); err != nil {
			return err
		}
	}

	return nil
}

// Query 查询知识库
func (b *baseKnowledge) Query(ctx context.Context, q Query) ([]Knowledge, error) {
	if q.Limit <= 0 {
		q.Limit = 10 // 默认限制
	}

	var result []Knowledge
	count := 0

	// 遍历所有知识
	b.items.Range(func(key, value interface{}) bool {
		k := value.(Knowledge)

		// 应用类型过滤
		if q.Type != "" && k.Type != q.Type {
			return true
		}

		// 应用自定义过滤
		if !matchesFilter(k, q.Filter) {
			return true
		}

		result = append(result, k)
		count++

		// 达到限制时停止
		return count < q.Limit
	})

	// 应用排序
	if len(q.Sort) > 0 {
		// TODO: 实现排序逻辑
	}

	return result, nil
}

// SemanticSearch 语义搜索
func (b *baseKnowledge) SemanticSearch(ctx context.Context, text string, limit int) ([]Knowledge, error) {
	if limit <= 0 {
		limit = 10 // 默认限制
	}

	// 如果没有向量存储，返回错误
	if b.vector == nil {
		return nil, errors.New("vector store not available")
	}

	// 生成查询向量
	queryVector, err := b.vector.Embed(ctx, text)
	if err != nil {
		return nil, err
	}

	// 执行向量搜索
	ids, scores, err := b.vector.Search(ctx, queryVector, limit)
	if err != nil {
		return nil, err
	}

	// 获取知识
	result := make([]Knowledge, 0, len(ids))
	for i, id := range ids {
		if itemI, ok := b.items.Load(id); ok {
			k := itemI.(Knowledge)
			// 添加相似度分数到元数据
			if k.Metadata == nil {
				k.Metadata = make(map[string]interface{})
			}
			k.Metadata["similarity_score"] = scores[i]
			result = append(result, k)
		}
	}

	return result, nil
}

// CreateContext 创建知识上下文
func (b *baseKnowledge) CreateContext(ctx context.Context, config KnowledgeConfig) (Context, error) {
	contextID := uuid.New().String()

	// 创建上下文
	knowledgeCtx := &knowledgeContext{
		id:     contextID,
		base:   b,
		config: config,
	}

	// 存储上下文
	b.mu.Lock()
	b.contexts[contextID] = knowledgeCtx
	b.mu.Unlock()

	return knowledgeCtx, nil
}

// 辅助函数

// validateKnowledge 验证知识是否有效
func validateKnowledge(k Knowledge) error {
	if k.Content == nil {
		return ErrInvalidKnowledge
	}
	return nil
}

// matchesFilter 检查知识是否匹配过滤条件
func matchesFilter(k Knowledge, filter map[string]interface{}) bool {
	if filter == nil {
		return true
	}

	// 检查元数据是否匹配过滤条件
	for key, value := range filter {
		if metaValue, ok := k.Metadata[key]; !ok || metaValue != value {
			return false
		}
	}

	return true
}

// VectorStore 接口定义了向量存储的操作
type VectorStore interface {
	// Embed 将内容转换为向量
	Embed(ctx context.Context, content interface{}) ([]float32, error)
	// Add 添加向量到存储
	Add(ctx context.Context, id string, vector []float32) error
	// Update 更新存储中的向量
	Update(ctx context.Context, id string, vector []float32) error
	// Delete 从存储中删除向量
	Delete(ctx context.Context, id string) error
	// Search 搜索相似向量
	Search(ctx context.Context, vector []float32, limit int) ([]string, []float32, error)
}

// knowledgeContext 实现了Context接口
type knowledgeContext struct {
	id     string
	base   *baseKnowledge
	config KnowledgeConfig
	items  sync.Map
}

// Query 在上下文中查询知识
func (c *knowledgeContext) Query(ctx context.Context, q Query) ([]Knowledge, error) {
	// 应用上下文过滤条件
	if q.Filter == nil {
		q.Filter = make(map[string]interface{})
	}

	// 合并上下文过滤条件
	for k, v := range c.config.Filters {
		if _, exists := q.Filter[k]; !exists {
			q.Filter[k] = v
		}
	}

	return c.base.Query(ctx, q)
}

// SemanticSearch 在上下文中进行语义搜索
func (c *knowledgeContext) SemanticSearch(ctx context.Context, text string, limit int) ([]Knowledge, error) {
	// 执行基础语义搜索
	results, err := c.base.SemanticSearch(ctx, text, limit*2) // 获取更多结果，然后过滤
	if err != nil {
		return nil, err
	}

	// 应用上下文过滤
	filtered := make([]Knowledge, 0, limit)
	for _, k := range results {
		if matchesFilter(k, c.config.Filters) {
			filtered = append(filtered, k)
			if len(filtered) >= limit {
				break
			}
		}
	}

	return filtered, nil
}

// AddKnowledge 向上下文添加知识
func (c *knowledgeContext) AddKnowledge(ctx context.Context, k Knowledge) error {
	// 确保知识符合上下文过滤条件
	if k.Metadata == nil {
		k.Metadata = make(map[string]interface{})
	}

	// 添加上下文标记
	k.Metadata["context_id"] = c.id

	// 添加到基础知识库
	if err := c.base.AddKnowledge(ctx, k); err != nil {
		return err
	}

	// 存储在上下文中
	c.items.Store(k.ID, k)

	return nil
}

// GetRelevantKnowledge 获取与给定文本相关的知识
func (c *knowledgeContext) GetRelevantKnowledge(ctx context.Context, text string, limit int) ([]Knowledge, error) {
	return c.SemanticSearch(ctx, text, limit)
}
