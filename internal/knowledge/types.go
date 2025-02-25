package knowledge

import (
	"context"
	"errors"
)

// Knowledge 代表一个知识单元
type Knowledge struct {
	ID       string
	Type     string
	Content  interface{}
	Metadata map[string]interface{}
	Vector   []float32
}

// Base 接口定义了知识库的基本操作
type Base interface {
	// AddKnowledge 添加知识到知识库
	AddKnowledge(ctx context.Context, k Knowledge) error
	// UpdateKnowledge 更新知识库中的知识
	UpdateKnowledge(ctx context.Context, id string, k Knowledge) error
	// DeleteKnowledge 从知识库中删除知识
	DeleteKnowledge(ctx context.Context, id string) error
	// Query 查询知识库
	Query(ctx context.Context, q Query) ([]Knowledge, error)
	// SemanticSearch 语义搜索
	SemanticSearch(ctx context.Context, text string, limit int) ([]Knowledge, error)
	// CreateContext 创建知识上下文
	CreateContext(ctx context.Context, config KnowledgeConfig) (Context, error)
}

// Context 接口定义了知识上下文的操作
type Context interface {
	// Query 在上下文中查询知识
	Query(ctx context.Context, q Query) ([]Knowledge, error)
	// SemanticSearch 在上下文中进行语义搜索
	SemanticSearch(ctx context.Context, text string, limit int) ([]Knowledge, error)
	// AddKnowledge 向上下文添加知识
	AddKnowledge(ctx context.Context, k Knowledge) error
	// GetRelevantKnowledge 获取与给定文本相关的知识
	GetRelevantKnowledge(ctx context.Context, text string, limit int) ([]Knowledge, error)
}

// Query 定义了知识查询条件
type Query struct {
	Type   string
	Filter map[string]interface{}
	Sort   []SortField
	Limit  int
}

// SortField 定义了排序字段
type SortField struct {
	Field     string
	Ascending bool
}

// KnowledgeConfig 定义了知识库配置
type KnowledgeConfig struct {
	Type    string
	Sources []string
	Filters map[string]interface{}
}

// VectorConfig 定义了向量存储配置
type VectorConfig struct {
	Type       string
	Dimensions int
	Metric     string
}

// 错误定义
var (
	ErrKnowledgeNotFound = errors.New("knowledge not found")
	ErrInvalidKnowledge  = errors.New("invalid knowledge")
	ErrInvalidQuery      = errors.New("invalid query")
)
