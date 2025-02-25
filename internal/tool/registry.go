package tool

import (
	"context"
	"sync"
)

// Registry 提供工具注册和发现功能
type Registry struct {
	metadata   sync.Map // 存储工具元数据
	categories map[ToolCategory]map[string]struct{}
	tags       map[string]map[string]struct{}
	mu         sync.RWMutex
}

// NewRegistry 创建一个新的工具注册表
func NewRegistry() *Registry {
	return &Registry{
		categories: make(map[ToolCategory]map[string]struct{}),
		tags:       make(map[string]map[string]struct{}),
	}
}

// RegisterMetadata 注册工具元数据
func (r *Registry) RegisterMetadata(ctx context.Context, metadata ToolMetadata) error {
	if metadata.ID == "" {
		return ErrInvalidTool
	}

	// 存储元数据
	r.metadata.Store(metadata.ID, metadata)

	// 更新类别索引
	r.mu.Lock()
	for _, category := range metadata.Categories {
		if _, ok := r.categories[category]; !ok {
			r.categories[category] = make(map[string]struct{})
		}
		r.categories[category][metadata.ID] = struct{}{}
	}

	// 更新标签索引
	for _, tag := range metadata.Tags {
		if _, ok := r.tags[tag]; !ok {
			r.tags[tag] = make(map[string]struct{})
		}
		r.tags[tag][metadata.ID] = struct{}{}
	}
	r.mu.Unlock()

	return nil
}

// UnregisterMetadata 注销工具元数据
func (r *Registry) UnregisterMetadata(ctx context.Context, toolID string) error {
	metadataI, ok := r.metadata.Load(toolID)
	if !ok {
		return ErrToolNotFound
	}

	metadata := metadataI.(ToolMetadata)
	r.metadata.Delete(toolID)

	// 更新类别索引
	r.mu.Lock()
	for _, category := range metadata.Categories {
		if tools, ok := r.categories[category]; ok {
			delete(tools, toolID)
			if len(tools) == 0 {
				delete(r.categories, category)
			}
		}
	}

	// 更新标签索引
	for _, tag := range metadata.Tags {
		if tools, ok := r.tags[tag]; ok {
			delete(tools, toolID)
			if len(tools) == 0 {
				delete(r.tags, tag)
			}
		}
	}
	r.mu.Unlock()

	return nil
}

// GetMetadata 获取工具元数据
func (r *Registry) GetMetadata(ctx context.Context, toolID string) (ToolMetadata, error) {
	metadataI, ok := r.metadata.Load(toolID)
	if !ok {
		return ToolMetadata{}, ErrToolNotFound
	}

	return metadataI.(ToolMetadata), nil
}

// FindByCategory 按类别查找工具
func (r *Registry) FindByCategory(ctx context.Context, category ToolCategory) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools, ok := r.categories[category]
	if !ok {
		return []string{}
	}

	result := make([]string, 0, len(tools))
	for toolID := range tools {
		result = append(result, toolID)
	}

	return result
}

// FindByTag 按标签查找工具
func (r *Registry) FindByTag(ctx context.Context, tag string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools, ok := r.tags[tag]
	if !ok {
		return []string{}
	}

	result := make([]string, 0, len(tools))
	for toolID := range tools {
		result = append(result, toolID)
	}

	return result
}

// ListCategories 列出所有类别
func (r *Registry) ListCategories(ctx context.Context) []ToolCategory {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ToolCategory, 0, len(r.categories))
	for category := range r.categories {
		result = append(result, category)
	}

	return result
}

// ListTags 列出所有标签
func (r *Registry) ListTags(ctx context.Context) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]string, 0, len(r.tags))
	for tag := range r.tags {
		result = append(result, tag)
	}

	return result
}

// ListAllTools 列出所有工具ID
func (r *Registry) ListAllTools(ctx context.Context) []string {
	var result []string

	r.metadata.Range(func(key, value interface{}) bool {
		result = append(result, key.(string))
		return true
	})

	return result
}
