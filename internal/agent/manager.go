package agent

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// manager 实现了Manager接口
type manager struct {
	agents    sync.Map
	tasks     sync.Map
	events    map[string]chan Event
	eventsMu  sync.RWMutex
	toolMgr   interface{} // tool.Manager 接口，后续实现
	memoryMgr interface{} // memory.Manager 接口，后续实现
	knowledge interface{} // knowledge.Base 接口，后续实现
}

// NewManager 创建一个新的Agent管理器
func NewManager(toolMgr interface{}, memoryMgr interface{}, kb interface{}) Manager {
	return &manager{
		toolMgr:   toolMgr,
		memoryMgr: memoryMgr,
		knowledge: kb,
		events:    make(map[string]chan Event),
	}
}

// CreateAgent 创建一个新的Agent
func (m *manager) CreateAgent(ctx context.Context, config AgentConfig) (Agent, error) {
	if config.ID == "" {
		config.ID = uuid.New().String()
	}

	// 验证配置
	if err := m.validateConfig(config); err != nil {
		return nil, err
	}

	// 初始化必要的组件
	// 注意：这里只是示例，实际实现需要根据具体接口定义

	// 创建agent
	agent := &baseAgent{
		id:     config.ID,
		config: config,
		status: AgentStatus{
			ID:     config.ID,
			Status: "initialized",
		},
	}
	if err := agent.Initialize(ctx); err != nil {
		return nil, err
	}

	m.agents.Store(config.ID, agent)

	// 创建事件通道
	m.eventsMu.Lock()
	m.events[config.ID] = make(chan Event, 100) // 缓冲区大小可配置
	m.eventsMu.Unlock()

	return agent, nil
}

// DestroyAgent 销毁一个Agent
func (m *manager) DestroyAgent(ctx context.Context, agentID string) error {
	agentI, ok := m.agents.Load(agentID)
	if !ok {
		return ErrAgentNotFound
	}

	agent := agentI.(Agent)
	if err := agent.Stop(ctx); err != nil {
		return err
	}

	m.agents.Delete(agentID)

	// 关闭事件通道
	m.eventsMu.Lock()
	if ch, ok := m.events[agentID]; ok {
		close(ch)
		delete(m.events, agentID)
	}
	m.eventsMu.Unlock()

	return nil
}

// PauseAgent 暂停一个Agent
func (m *manager) PauseAgent(ctx context.Context, agentID string) error {
	agentI, ok := m.agents.Load(agentID)
	if !ok {
		return ErrAgentNotFound
	}

	agent := agentI.(*baseAgent)
	agent.status.Status = "paused"

	// 发送暂停事件
	m.emitEvent(agentID, Event{
		ID:        uuid.New().String(),
		Type:      "agent_paused",
		Data:      agent.id,
		Timestamp: time.Now(),
	})

	return nil
}

// ResumeAgent 恢复一个Agent
func (m *manager) ResumeAgent(ctx context.Context, agentID string) error {
	agentI, ok := m.agents.Load(agentID)
	if !ok {
		return ErrAgentNotFound
	}

	agent := agentI.(*baseAgent)
	agent.status.Status = "running"

	// 发送恢复事件
	m.emitEvent(agentID, Event{
		ID:        uuid.New().String(),
		Type:      "agent_resumed",
		Data:      agent.id,
		Timestamp: time.Now(),
	})

	return nil
}

// AssignTask 分配任务给Agent
func (m *manager) AssignTask(ctx context.Context, agentID string, task Task) error {
	agentI, ok := m.agents.Load(agentID)
	if !ok {
		return ErrAgentNotFound
	}

	if task.ID == "" {
		task.ID = uuid.New().String()
	}

	// 存储任务初始状态
	taskStatus := TaskStatus{
		ID:        task.ID,
		Status:    "pending",
		Progress:  0.0,
		StartTime: time.Now(),
	}
	m.tasks.Store(task.ID, taskStatus)

	// 发送任务分配事件
	m.emitEvent(agentID, Event{
		ID:        uuid.New().String(),
		Type:      "task_assigned",
		Data:      task.ID,
		Timestamp: time.Now(),
	})

	// 异步执行任务
	go func() {
		agent := agentI.(Agent)

		// 更新Agent状态
		if baseAgent, ok := agent.(*baseAgent); ok {
			baseAgent.status.Status = "working"
			baseAgent.status.CurrentTask = task.ID
		}

		// 更新任务状态
		taskStatus.Status = "running"
		m.tasks.Store(task.ID, taskStatus)

		// 执行任务
		result, err := agent.Execute(ctx, task)

		// 更新任务状态
		endTime := time.Now()
		taskStatus.EndTime = endTime

		if err != nil {
			taskStatus.Status = "failed"
			taskStatus.Error = err
			m.emitEvent(agentID, Event{
				ID:        uuid.New().String(),
				Type:      "task_failed",
				Data:      map[string]interface{}{"task_id": task.ID, "error": err.Error()},
				Timestamp: endTime,
			})
		} else {
			taskStatus.Status = "completed"
			taskStatus.Result = result
			taskStatus.Progress = 1.0
			m.emitEvent(agentID, Event{
				ID:        uuid.New().String(),
				Type:      "task_completed",
				Data:      map[string]interface{}{"task_id": task.ID},
				Timestamp: endTime,
			})
		}

		m.tasks.Store(task.ID, taskStatus)

		// 更新Agent状态
		if baseAgent, ok := agent.(*baseAgent); ok {
			baseAgent.status.Status = "idle"
			baseAgent.status.CurrentTask = ""
		}
	}()

	return nil
}

// CancelTask 取消任务
func (m *manager) CancelTask(ctx context.Context, taskID string) error {
	taskI, ok := m.tasks.Load(taskID)
	if !ok {
		return ErrTaskNotFound
	}

	taskStatus := taskI.(TaskStatus)
	if taskStatus.Status == "completed" || taskStatus.Status == "failed" {
		return nil // 任务已经完成或失败，无需取消
	}

	// 更新任务状态
	taskStatus.Status = "cancelled"
	taskStatus.EndTime = time.Now()
	m.tasks.Store(taskID, taskStatus)

	// TODO: 实际中需要通知Agent取消任务的执行

	return nil
}

// GetTaskStatus 获取任务状态
func (m *manager) GetTaskStatus(ctx context.Context, taskID string) (TaskStatus, error) {
	taskI, ok := m.tasks.Load(taskID)
	if !ok {
		return TaskStatus{}, ErrTaskNotFound
	}

	return taskI.(TaskStatus), nil
}

// GetAgentStatus 获取Agent状态
func (m *manager) GetAgentStatus(ctx context.Context, agentID string) (AgentStatus, error) {
	agentI, ok := m.agents.Load(agentID)
	if !ok {
		return AgentStatus{}, ErrAgentNotFound
	}

	agent := agentI.(Agent)
	return agent.Status(), nil
}

// SubscribeToEvents 订阅Agent事件
func (m *manager) SubscribeToEvents(ctx context.Context, agentID string) (<-chan Event, error) {
	m.eventsMu.RLock()
	ch, ok := m.events[agentID]
	m.eventsMu.RUnlock()

	if !ok {
		return nil, ErrAgentNotFound
	}

	return ch, nil
}

// 内部辅助方法

// validateConfig 验证Agent配置
func (m *manager) validateConfig(config AgentConfig) error {
	if config.Name == "" {
		return ErrInvalidConfig
	}

	// TODO: 添加更多验证逻辑

	return nil
}

// emitEvent 发送事件
func (m *manager) emitEvent(agentID string, event Event) {
	m.eventsMu.RLock()
	ch, ok := m.events[agentID]
	m.eventsMu.RUnlock()

	if ok {
		// 非阻塞发送，如果通道已满则丢弃事件
		select {
		case ch <- event:
		default:
			// TODO: 可以考虑记录日志
		}
	}
}

// baseAgent 是Agent接口的基本实现
type baseAgent struct {
	id     string
	config AgentConfig
	status AgentStatus
	mu     sync.RWMutex
}

func (a *baseAgent) Initialize(ctx context.Context) error {
	// 基本初始化逻辑
	return nil
}

func (a *baseAgent) Execute(ctx context.Context, task Task) (Result, error) {
	// 只是示例，实际实现需要根据Agent类型和任务类型
	return Result{
		Data:      map[string]interface{}{"message": "Task executed successfully"},
		Metadata:  map[string]interface{}{"task_type": task.Type},
		Timestamp: time.Now(),
	}, nil
}

func (a *baseAgent) Stop(ctx context.Context) error {
	// 停止逻辑
	return nil
}

func (a *baseAgent) Status() AgentStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}
