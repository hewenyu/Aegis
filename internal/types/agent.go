package types

import (
	"context"
	"time"
)

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

// AgentStatus 代表Agent的状态
type AgentStatus struct {
	ID          string
	Status      string
	CurrentTask string
	Memory      MemoryStats
	Resources   ResourceStats
}

// ResourceStats 代表Agent使用的资源统计
type ResourceStats struct {
	CPU        float64
	Memory     int64
	Tokens     int
	APILatency time.Duration
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
