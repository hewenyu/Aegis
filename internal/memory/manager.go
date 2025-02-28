package memory

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hewenyu/Aegis/internal/types"
)

// manager 实现了Manager接口
type manager struct {
	stores sync.Map
	mu     sync.RWMutex
}

// NewManager 创建一个新的记忆管理器
func NewManager() types.Manager {
	return &manager{}
}

// CreateStore 创建一个新的记忆存储
func (m *manager) CreateStore(ctx context.Context, config types.MemoryConfig) (types.Store, error) {
	storeID := uuid.New().String()

	var store types.Store
	switch config.Type {
	case "default", "":
		store = NewInMemoryStore(storeID, config.Size)
	default:
		return nil, errors.New("unsupported memory store type")
	}

	m.stores.Store(storeID, store)
	return store, nil
}

// GetStore 获取指定ID的记忆存储
func (m *manager) GetStore(ctx context.Context, storeID string) (types.Store, error) {
	storeI, ok := m.stores.Load(storeID)
	if !ok {
		return nil, types.ErrStoreNotFound
	}
	return storeI.(types.Store), nil
}

// DeleteStore 删除指定ID的记忆存储
func (m *manager) DeleteStore(ctx context.Context, storeID string) error {
	if _, ok := m.stores.Load(storeID); !ok {
		return types.ErrStoreNotFound
	}
	m.stores.Delete(storeID)
	return nil
}

// ListStores 列出所有记忆存储
func (m *manager) ListStores(ctx context.Context) ([]string, error) {
	var storeIDs []string
	m.stores.Range(func(key, value interface{}) bool {
		storeIDs = append(storeIDs, key.(string))
		return true
	})
	return storeIDs, nil
}

// inMemoryStore 是Store接口的内存实现
type inMemoryStore struct {
	id       string
	memories sync.Map
	maxSize  int
	stats    types.MemoryStats
	mu       sync.RWMutex
}

// NewInMemoryStore 创建一个新的内存记忆存储
func NewInMemoryStore(id string, maxSize int) types.Store {
	if maxSize <= 0 {
		maxSize = 1000 // 默认大小
	}
	return &inMemoryStore{
		id:      id,
		maxSize: maxSize,
	}
}

// Store 存储记忆
func (s *inMemoryStore) Store(ctx context.Context, m types.Memory) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}

	if m.Timestamp.IsZero() {
		m.Timestamp = time.Now()
	}

	// 验证记忆
	if err := validateMemory(m); err != nil {
		return err
	}

	// 存储记忆
	s.memories.Store(m.ID, m)

	// 更新统计信息
	s.mu.Lock()
	s.stats.TotalItems++
	switch m.Type {
	case types.ShortTerm:
		s.stats.ShortTerm++
	case types.LongTerm:
		s.stats.LongTerm++
	case types.Working:
		s.stats.WorkingItems++
	}
	s.mu.Unlock()

	// 如果超过最大大小，进行整合
	if s.stats.TotalItems > s.maxSize {
		go s.Consolidate(context.Background())
	}

	return nil
}

// Recall 检索记忆
func (s *inMemoryStore) Recall(ctx context.Context, query types.MemoryQuery) ([]types.Memory, error) {
	if query.Limit <= 0 {
		query.Limit = 10 // 默认限制
	}

	var result []types.Memory
	count := 0

	// 遍历所有记忆
	s.memories.Range(func(key, value interface{}) bool {
		m := value.(types.Memory)

		// 应用类型过滤
		if query.Type != "" && m.Type != query.Type {
			return true
		}

		// 应用时间范围过滤
		if !query.TimeRange.Start.IsZero() && m.Timestamp.Before(query.TimeRange.Start) {
			return true
		}
		if !query.TimeRange.End.IsZero() && m.Timestamp.After(query.TimeRange.End) {
			return true
		}

		// 应用重要性过滤
		if query.Importance > 0 && m.Importance < query.Importance {
			return true
		}

		// 应用上下文过滤
		if !matchesContext(m, query.Context) {
			return true
		}

		result = append(result, m)
		count++

		// 达到限制时停止
		return count < query.Limit
	})

	// 按时间排序（最新的优先）
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.After(result[j].Timestamp)
	})

	return result, nil
}

// Forget 删除记忆
func (s *inMemoryStore) Forget(ctx context.Context, filter types.MemoryFilter) error {
	var toDelete []string

	// 找出要删除的记忆
	s.memories.Range(func(key, value interface{}) bool {
		id := key.(string)
		m := value.(types.Memory)

		// 如果指定了ID列表，只检查这些ID
		if len(filter.IDs) > 0 {
			found := false
			for _, filterID := range filter.IDs {
				if id == filterID {
					found = true
					break
				}
			}
			if !found {
				return true
			}
		}

		// 应用类型过滤
		if filter.Type != "" && m.Type != filter.Type {
			return true
		}

		// 应用时间范围过滤
		if !filter.TimeRange.Start.IsZero() && m.Timestamp.Before(filter.TimeRange.Start) {
			return true
		}
		if !filter.TimeRange.End.IsZero() && m.Timestamp.After(filter.TimeRange.End) {
			return true
		}

		// 应用重要性过滤
		if filter.Importance > 0 && m.Importance < filter.Importance {
			return true
		}

		// 应用上下文过滤
		if !matchesContext(m, filter.Context) {
			return true
		}

		toDelete = append(toDelete, id)
		return true
	})

	// 删除记忆
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range toDelete {
		if memoryI, ok := s.memories.Load(id); ok {
			memory := memoryI.(types.Memory)
			s.memories.Delete(id)
			s.stats.TotalItems--
			switch memory.Type {
			case types.ShortTerm:
				s.stats.ShortTerm--
			case types.LongTerm:
				s.stats.LongTerm--
			case types.Working:
				s.stats.WorkingItems--
			}
		}
	}

	return nil
}

// Consolidate 整合记忆
func (s *inMemoryStore) Consolidate(ctx context.Context) error {
	// 简单的整合策略：删除最旧的短期记忆，直到总数低于最大大小的80%
	targetSize := int(float64(s.maxSize) * 0.8)

	if s.stats.TotalItems <= targetSize {
		return nil
	}

	// 获取所有短期记忆
	var shortTermMemories []types.Memory
	s.memories.Range(func(key, value interface{}) bool {
		m := value.(types.Memory)
		if m.Type == types.ShortTerm {
			shortTermMemories = append(shortTermMemories, m)
		}
		return true
	})

	// 按时间排序（最旧的优先）
	sort.Slice(shortTermMemories, func(i, j int) bool {
		return shortTermMemories[i].Timestamp.Before(shortTermMemories[j].Timestamp)
	})

	// 计算需要删除的数量
	toDeleteCount := s.stats.TotalItems - targetSize
	if toDeleteCount > len(shortTermMemories) {
		toDeleteCount = len(shortTermMemories)
	}

	// 删除最旧的短期记忆
	for i := 0; i < toDeleteCount; i++ {
		s.memories.Delete(shortTermMemories[i].ID)
	}

	// 更新统计信息
	s.mu.Lock()
	s.stats.TotalItems -= toDeleteCount
	s.stats.ShortTerm -= toDeleteCount
	s.mu.Unlock()

	return nil
}

// GetStats 获取记忆统计信息
func (s *inMemoryStore) GetStats(ctx context.Context) (types.MemoryStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats, nil
}

// 辅助函数

// validateMemory 验证记忆是否有效
func validateMemory(m types.Memory) error {
	if m.Content == nil {
		return types.ErrInvalidMemory
	}
	return nil
}

// matchesContext 检查记忆是否匹配上下文
func matchesContext(m types.Memory, context map[string]interface{}) bool {
	if context == nil {
		return true
	}

	for key, value := range context {
		if ctxValue, ok := m.Context[key]; !ok || ctxValue != value {
			return false
		}
	}

	return true
}
