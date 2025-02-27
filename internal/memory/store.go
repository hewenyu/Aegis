package memory

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryIndex 提供记忆的索引功能
type MemoryIndex struct {
	byType     map[MemoryType]map[string]struct{}
	byContext  map[string]map[string]map[string]struct{} // context key -> value -> memory ID
	byTimespan map[string]time.Time                      // memory ID -> timestamp
	mu         sync.RWMutex
}

// NewMemoryIndex 创建一个新的记忆索引
func NewMemoryIndex() *MemoryIndex {
	return &MemoryIndex{
		byType:     make(map[MemoryType]map[string]struct{}),
		byContext:  make(map[string]map[string]map[string]struct{}),
		byTimespan: make(map[string]time.Time),
	}
}

// AddMemory 将记忆添加到索引
func (idx *MemoryIndex) AddMemory(m Memory) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// 按类型索引
	if _, ok := idx.byType[m.Type]; !ok {
		idx.byType[m.Type] = make(map[string]struct{})
	}
	idx.byType[m.Type][m.ID] = struct{}{}

	// 按上下文索引
	for key, value := range m.Context {
		if _, ok := idx.byContext[key]; !ok {
			idx.byContext[key] = make(map[string]map[string]struct{})
		}

		strValue := toString(value)
		if _, ok := idx.byContext[key][strValue]; !ok {
			idx.byContext[key][strValue] = make(map[string]struct{})
		}

		idx.byContext[key][strValue][m.ID] = struct{}{}
	}

	// 按时间索引
	idx.byTimespan[m.ID] = m.Timestamp
}

// RemoveMemory 从索引中移除记忆
func (idx *MemoryIndex) RemoveMemory(m Memory) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// 从类型索引中移除
	if typeIndex, ok := idx.byType[m.Type]; ok {
		delete(typeIndex, m.ID)
		if len(typeIndex) == 0 {
			delete(idx.byType, m.Type)
		}
	}

	// 从上下文索引中移除
	for key, value := range m.Context {
		strValue := toString(value)
		if valueMap, ok := idx.byContext[key]; ok {
			if idMap, ok := valueMap[strValue]; ok {
				delete(idMap, m.ID)
				if len(idMap) == 0 {
					delete(valueMap, strValue)
					if len(valueMap) == 0 {
						delete(idx.byContext, key)
					}
				}
			}
		}
	}

	// 从时间索引中移除
	delete(idx.byTimespan, m.ID)
}

// FindByType 按类型查找记忆ID
func (idx *MemoryIndex) FindByType(memType MemoryType) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if typeIndex, ok := idx.byType[memType]; ok {
		result := make([]string, 0, len(typeIndex))
		for id := range typeIndex {
			result = append(result, id)
		}
		return result
	}
	return []string{}
}

// FindByContext 按上下文查找记忆ID
func (idx *MemoryIndex) FindByContext(key string, value interface{}) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	strValue := toString(value)
	if valueMap, ok := idx.byContext[key]; ok {
		if idMap, ok := valueMap[strValue]; ok {
			result := make([]string, 0, len(idMap))
			for id := range idMap {
				result = append(result, id)
			}
			return result
		}
	}
	return []string{}
}

// FindByTimeRange 按时间范围查找记忆ID
func (idx *MemoryIndex) FindByTimeRange(start, end time.Time) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var result []string
	for id, timestamp := range idx.byTimespan {
		if (start.IsZero() || !timestamp.Before(start)) &&
			(end.IsZero() || !timestamp.After(end)) {
			result = append(result, id)
		}
	}
	return result
}

// 辅助函数

// toString 将值转换为字符串
func toString(value interface{}) string {
	if value == nil {
		return "nil"
	}
	switch v := value.(type) {
	case string:
		return v
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%T:%p", v, v)
	}
}

// MemoryRetriever 提供高级记忆检索功能
type MemoryRetriever struct {
	store Store
}

// NewMemoryRetriever 创建一个新的记忆检索器
func NewMemoryRetriever(store Store) *MemoryRetriever {
	return &MemoryRetriever{
		store: store,
	}
}

// GetRecentMemories 获取最近的记忆
func (r *MemoryRetriever) GetRecentMemories(ctx context.Context, limit int) ([]Memory, error) {
	query := MemoryQuery{
		Limit: limit,
	}
	return r.store.Recall(ctx, query)
}

// GetRelevantMemories 获取与上下文相关的记忆
func (r *MemoryRetriever) GetRelevantMemories(ctx context.Context, contextKey string, contextValue interface{}, limit int) ([]Memory, error) {
	query := MemoryQuery{
		Context: map[string]interface{}{
			contextKey: contextValue,
		},
		Limit: limit,
	}
	return r.store.Recall(ctx, query)
}

// GetImportantMemories 获取重要的记忆
func (r *MemoryRetriever) GetImportantMemories(ctx context.Context, minImportance float64, limit int) ([]Memory, error) {
	query := MemoryQuery{
		Importance: minImportance,
		Limit:      limit,
	}
	return r.store.Recall(ctx, query)
}

// GetMemoriesByType 按类型获取记忆
func (r *MemoryRetriever) GetMemoriesByType(ctx context.Context, memType MemoryType, limit int) ([]Memory, error) {
	query := MemoryQuery{
		Type:  memType,
		Limit: limit,
	}
	return r.store.Recall(ctx, query)
}

// GetMemoriesByTimeRange 按时间范围获取记忆
func (r *MemoryRetriever) GetMemoriesByTimeRange(ctx context.Context, start, end time.Time, limit int) ([]Memory, error) {
	query := MemoryQuery{
		TimeRange: TimeRange{
			Start: start,
			End:   end,
		},
		Limit: limit,
	}
	return r.store.Recall(ctx, query)
}
