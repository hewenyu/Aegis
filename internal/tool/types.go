package tool

import (
	"context"
	"errors"
)

// Tool 接口定义了Agent可以使用的工具
type Tool interface {
	// ID 返回工具的唯一标识符
	ID() string
	// Name 返回工具的名称
	Name() string
	// Description 返回工具的描述
	Description() string
	// Version 返回工具的版本
	Version() string
	// Execute 执行工具操作
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
	// Validate 验证参数是否有效
	Validate(params map[string]interface{}) error
}

// Manager 接口定义了工具管理器的操作
type Manager interface {
	// RegisterTool 注册一个工具
	RegisterTool(ctx context.Context, tool Tool) error
	// UnregisterTool 注销一个工具
	UnregisterTool(ctx context.Context, toolID string) error
	// GetTool 获取指定ID的工具
	GetTool(ctx context.Context, toolID string) (Tool, error)
	// GetTools 获取符合过滤条件的工具列表
	GetTools(ctx context.Context, filter ToolFilter) ([]Tool, error)
	// ExecuteTool 执行指定工具
	ExecuteTool(ctx context.Context, toolID string, params map[string]interface{}) (interface{}, error)
}

// ToolFilter 定义了工具过滤条件
type ToolFilter struct {
	Categories []string
	Tags       []string
	Version    string
}

// ToolConfig 定义了工具配置
type ToolConfig struct {
	ID     string
	Config map[string]interface{}
}

// ToolCategory 定义了工具类别
type ToolCategory string

// 预定义工具类别
const (
	CategorySearch     ToolCategory = "search"
	CategoryAnalysis   ToolCategory = "analysis"
	CategoryGeneration ToolCategory = "generation"
	CategoryIO         ToolCategory = "io"
	CategorySystem     ToolCategory = "system"
)

// ToolMetadata 定义了工具元数据
type ToolMetadata struct {
	ID          string
	Name        string
	Description string
	Version     string
	Author      string
	Categories  []ToolCategory
	Tags        []string
	Parameters  []ParameterSpec
	Returns     []ReturnSpec
}

// ParameterSpec 定义了工具参数规格
type ParameterSpec struct {
	Name        string
	Type        string
	Description string
	Required    bool
	Default     interface{}
}

// ReturnSpec 定义了工具返回值规格
type ReturnSpec struct {
	Name        string
	Type        string
	Description string
}

// 错误定义
var (
	ErrToolNotFound      = errors.New("tool not found")
	ErrToolAlreadyExists = errors.New("tool already exists")
	ErrInvalidTool       = errors.New("invalid tool")
	ErrInvalidParameter  = errors.New("invalid parameter")
)
