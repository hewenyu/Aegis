package memory

import (
	"context"
	"errors"
	"sync"

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
	m.mu.RLock()
	defer m.mu.RUnlock()
	storeI, ok := m.stores.Load(storeID)
	if !ok {
		return nil, types.ErrStoreNotFound
	}
	return storeI.(types.Store), nil
}

// DeleteStore 删除指定ID的记忆存储
func (m *manager) DeleteStore(ctx context.Context, storeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.stores.Load(storeID); !ok {
		return types.ErrStoreNotFound
	}
	m.stores.Delete(storeID)
	return nil
}

// ListStores 列出所有记忆存储
func (m *manager) ListStores(ctx context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var storeIDs []string
	m.stores.Range(func(key, value interface{}) bool {
		storeIDs = append(storeIDs, key.(string))
		return true
	})
	return storeIDs, nil
}
