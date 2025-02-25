package memory

import (
	"context"
	"errors"
	"time"
)

// Memory 代表一个记忆单元
type Memory struct {
	ID         string
	Type       MemoryType
	Content    interface{}
	Timestamp  time.Time
	Importance float64
	Context    map[string]interface{}
}

// MemoryType 定义了记忆类型
type MemoryType string

// 预定义记忆类型
const (
	ShortTerm MemoryType = "short_term"
	LongTerm  MemoryType = "long_term"
	Working   MemoryType = "working"
)

// Manager 接口定义了记忆管理器的操作
type Manager interface {
	// CreateStore 创建一个新的记忆存储
	CreateStore(ctx context.Context, config MemoryConfig) (Store, error)
	// GetStore 获取指定ID的记忆存储
	GetStore(ctx context.Context, storeID string) (Store, error)
	// DeleteStore 删除指定ID的记忆存储
	DeleteStore(ctx context.Context, storeID string) error
	// ListStores 列出所有记忆存储
	ListStores(ctx context.Context) ([]string, error)
}

// Store 接口定义了记忆存储的操作
type Store interface {
	// Store 存储记忆
	Store(ctx context.Context, m Memory) error
	// Recall 检索记忆
	Recall(ctx context.Context, query MemoryQuery) ([]Memory, error)
	// Forget 删除记忆
	Forget(ctx context.Context, filter MemoryFilter) error
	// Consolidate 整合记忆
	Consolidate(ctx context.Context) error
	// GetStats 获取记忆统计信息
	GetStats(ctx context.Context) (MemoryStats, error)
}

// MemoryQuery 定义了记忆查询条件
type MemoryQuery struct {
	Type       MemoryType
	TimeRange  TimeRange
	Importance float64
	Context    map[string]interface{}
	Limit      int
}

// MemoryFilter 定义了记忆过滤条件
type MemoryFilter struct {
	IDs        []string
	Type       MemoryType
	TimeRange  TimeRange
	Importance float64
	Context    map[string]interface{}
}

// TimeRange 定义了时间范围
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// MemoryConfig 定义了记忆存储配置
type MemoryConfig struct {
	Type string
	Size int
}

// MemoryStats 定义了记忆统计信息
type MemoryStats struct {
	TotalItems   int
	ShortTerm    int
	LongTerm     int
	WorkingItems int
}

// 错误定义
var (
	ErrMemoryNotFound = errors.New("memory not found")
	ErrStoreNotFound  = errors.New("memory store not found")
	ErrInvalidMemory  = errors.New("invalid memory")
	ErrInvalidQuery   = errors.New("invalid query")
)
