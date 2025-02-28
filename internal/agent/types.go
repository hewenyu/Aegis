package agent

import (
	"context"
	"errors"
	"time"

	"github.com/hewenyu/Aegis/internal/types"
)

// AgentConfig 定义了创建Agent所需的配置
type AgentConfig struct {
	ID           string
	Name         string
	Description  string
	Capabilities []string
	Model        ModelConfig
	Tools        []ToolConfig
	Memory       types.MemoryConfig
	Knowledge    KnowledgeConfig
}

// ModelConfig 定义了AI模型的配置
type ModelConfig struct {
	Type        string
	Temperature float64
	MaxTokens   int
}

// ToolConfig 定义了Agent要使用的工具配置
type ToolConfig struct {
	ID     string
	Config map[string]interface{}
}

// KnowledgeConfig 定义了Agent知识库的配置
type KnowledgeConfig struct {
	Type    string
	Sources []string
}

// Manager 接口定义了Agent管理器的操作
type Manager interface {
	// Agent 生命周期管理
	CreateAgent(ctx context.Context, config AgentConfig) (types.Agent, error)
	DestroyAgent(ctx context.Context, agentID string) error
	PauseAgent(ctx context.Context, agentID string) error
	ResumeAgent(ctx context.Context, agentID string) error

	// 任务管理
	AssignTask(ctx context.Context, agentID string, task types.Task) error
	CancelTask(ctx context.Context, taskID string) error
	GetTaskStatus(ctx context.Context, taskID string) (types.TaskStatus, error)

	// 状态监控
	GetAgentStatus(ctx context.Context, agentID string) (types.AgentStatus, error)
	SubscribeToEvents(ctx context.Context, agentID string) (<-chan Event, error)
}

// Event 代表Agent产生的事件
type Event struct {
	ID        string
	Type      string
	Data      interface{}
	Timestamp time.Time
}

func NewEvent(id string, t string, data interface{}) *Event {
	return &Event{
		ID:        id,
		Type:      t,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// 错误定义
var (
	ErrAgentNotFound = errors.New("agent not found")
	ErrTaskNotFound  = errors.New("task not found")
	ErrInvalidConfig = errors.New("invalid configuration")
	ErrTaskFailed    = errors.New("task execution failed")
)
