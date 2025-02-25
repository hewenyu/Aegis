package tool

import (
	"context"
	"sync"
)

// manager 实现了Manager接口
type manager struct {
	tools       sync.Map
	permissions sync.Map
}

// NewManager 创建一个新的工具管理器
func NewManager() Manager {
	return &manager{}
}

// RegisterTool 注册一个工具
func (m *manager) RegisterTool(ctx context.Context, tool Tool) error {
	if tool == nil {
		return ErrInvalidTool
	}

	// 验证工具
	if err := m.validateTool(tool); err != nil {
		return err
	}

	// 检查是否已存在
	if _, loaded := m.tools.LoadOrStore(tool.ID(), tool); loaded {
		return ErrToolAlreadyExists
	}

	return nil
}

// UnregisterTool 注销一个工具
func (m *manager) UnregisterTool(ctx context.Context, toolID string) error {
	if _, ok := m.tools.Load(toolID); !ok {
		return ErrToolNotFound
	}

	m.tools.Delete(toolID)
	return nil
}

// GetTool 获取指定ID的工具
func (m *manager) GetTool(ctx context.Context, toolID string) (Tool, error) {
	toolI, ok := m.tools.Load(toolID)
	if !ok {
		return nil, ErrToolNotFound
	}

	return toolI.(Tool), nil
}

// GetTools 获取符合过滤条件的工具列表
func (m *manager) GetTools(ctx context.Context, filter ToolFilter) ([]Tool, error) {
	var result []Tool

	// 如果没有过滤条件，返回所有工具
	if len(filter.Categories) == 0 && len(filter.Tags) == 0 && filter.Version == "" {
		m.tools.Range(func(key, value interface{}) bool {
			result = append(result, value.(Tool))
			return true
		})
		return result, nil
	}

	// 应用过滤条件
	m.tools.Range(func(key, value interface{}) bool {
		tool := value.(Tool)

		// 如果指定了版本，但不匹配，则跳过
		if filter.Version != "" && tool.Version() != filter.Version {
			return true
		}

		// TODO: 实现类别和标签过滤
		// 这需要工具实现提供类别和标签信息的方法

		result = append(result, tool)
		return true
	})

	return result, nil
}

// ExecuteTool 执行指定工具
func (m *manager) ExecuteTool(ctx context.Context, toolID string, params map[string]interface{}) (interface{}, error) {
	toolI, ok := m.tools.Load(toolID)
	if !ok {
		return nil, ErrToolNotFound
	}
	tool := toolI.(Tool)

	// 参数验证
	if err := tool.Validate(params); err != nil {
		return nil, err
	}

	// 执行工具
	result, err := tool.Execute(ctx, params)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// validateTool 验证工具是否有效
func (m *manager) validateTool(tool Tool) error {
	if tool.ID() == "" {
		return ErrInvalidTool
	}
	if tool.Name() == "" {
		return ErrInvalidTool
	}
	if tool.Version() == "" {
		return ErrInvalidTool
	}
	return nil
}
