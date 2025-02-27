package agent

import (
	"context"
	"errors"
	"time"
)

// AgentConfig 定义了创建Agent所需的配置
type AgentConfig struct {
	ID           string
	Name         string
	Description  string
	Capabilities []string
	Model        ModelConfig
	Tools        []ToolConfig
	Memory       MemoryConfig
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

// MemoryConfig 定义了Agent记忆系统的配置
type MemoryConfig struct {
	Type string
	Size int
}

// KnowledgeConfig 定义了Agent知识库的配置
type KnowledgeConfig struct {
	Type    string
	Sources []string
}

// Agent 接口定义了AI Agent的基本操作
type Agent interface {
	// Initialize 初始化Agent
	Initialize(ctx context.Context) error
	// Execute 执行任务
	Execute(ctx context.Context, task Task) (Result, error)
	// Stop 停止Agent
	Stop(ctx context.Context) error
	// Status 获取Agent当前状态
	Status() AgentStatus
}

// Manager 接口定义了Agent管理器的操作
type Manager interface {
	// Agent 生命周期管理
	CreateAgent(ctx context.Context, config AgentConfig) (Agent, error)
	DestroyAgent(ctx context.Context, agentID string) error
	PauseAgent(ctx context.Context, agentID string) error
	ResumeAgent(ctx context.Context, agentID string) error

	// 任务管理
	AssignTask(ctx context.Context, agentID string, task Task) error
	CancelTask(ctx context.Context, taskID string) error
	GetTaskStatus(ctx context.Context, taskID string) (TaskStatus, error)

	// 状态监控
	GetAgentStatus(ctx context.Context, agentID string) (AgentStatus, error)
	SubscribeToEvents(ctx context.Context, agentID string) (<-chan Event, error)
}

// Task 代表一个需要Agent执行的任务
type Task struct {
	ID          string
	Type        string
	Description string
	Parameters  map[string]interface{}
	Deadline    time.Time
}

// Result 代表任务执行结果
type Result struct {
	Data      interface{}
	Metadata  map[string]interface{}
	Timestamp time.Time
}

// TaskStatus 代表任务的当前状态
type TaskStatus struct {
	ID        string
	Status    string
	Progress  float64
	Result    interface{}
	Error     error
	StartTime time.Time
	EndTime   time.Time
}

// AgentStatus 代表Agent的状态
type AgentStatus struct {
	ID          string
	Status      string
	CurrentTask string
	Memory      MemoryStats
	Resources   ResourceStats
}

// MemoryStats 代表Agent的内存统计
type MemoryStats struct {
	TotalItems   int
	ShortTerm    int
	LongTerm     int
	WorkingItems int
}

// ResourceStats 代表Agent使用的资源统计
type ResourceStats struct {
	CPU        float64
	Memory     int64
	Tokens     int
	APILatency time.Duration
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
