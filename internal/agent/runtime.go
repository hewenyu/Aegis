package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hewenyu/Aegis/internal/tool"
	"github.com/hewenyu/Aegis/internal/types"
)

// Runtime 提供Agent的运行时环境
type Runtime struct {
	agent         *baseAgent
	tools         []tool.Tool
	memory        types.Store
	knowledge     types.Context
	context       map[string]interface{}
	executionMu   sync.Mutex
	stopCh        chan struct{}
	taskQueue     chan types.Task
	maxConcurrent int
}

// NewRuntime 创建新的Agent运行时
func NewRuntime(agent *baseAgent, tools []tool.Tool, memory types.Store, knowledge types.Context) *Runtime {
	return &Runtime{
		agent:         agent,
		tools:         tools,
		memory:        memory,
		knowledge:     knowledge,
		context:       make(map[string]interface{}),
		stopCh:        make(chan struct{}),
		taskQueue:     make(chan types.Task, 10), // 任务队列缓冲区大小可配置
		maxConcurrent: 1,                         // 默认单任务执行
	}
}

// Start 启动运行时
func (r *Runtime) Start(ctx context.Context) error {
	// 启动任务处理循环
	for i := 0; i < r.maxConcurrent; i++ {
		go r.taskWorker(ctx)
	}
	return nil
}

// Stop 停止运行时
func (r *Runtime) Stop(ctx context.Context) error {
	close(r.stopCh)
	// TODO: 等待所有任务完成或超时
	return nil
}

// EnqueueTask 将任务加入队列
func (r *Runtime) EnqueueTask(task types.Task) error {
	select {
	case r.taskQueue <- task:
		return nil
	default:
		return fmt.Errorf("task queue is full")
	}
}

// taskWorker 是处理任务的工作协程
func (r *Runtime) taskWorker(ctx context.Context) {
	for {
		select {
		case <-r.stopCh:
			return
		case task := <-r.taskQueue:
			r.processTask(ctx, task)
		}
	}
}

// processTask 处理单个任务
func (r *Runtime) processTask(ctx context.Context, task types.Task) {
	// 建立任务执行上下文
	taskCtx := r.createTaskContext(ctx, task)

	// 记录任务开始
	r.recordEvent(taskCtx, "task_started", task.ID)

	// 执行任务
	result, err := r.executeTask(taskCtx, task)

	// 记录任务结束
	if err != nil {
		r.recordEvent(taskCtx, "task_failed", map[string]interface{}{
			"task_id": task.ID,
			"error":   err.Error(),
		})
	} else {
		r.recordEvent(taskCtx, "task_completed", map[string]interface{}{
			"task_id": task.ID,
			"result":  result,
		})
	}
}

// createTaskContext 创建任务执行上下文
func (r *Runtime) createTaskContext(ctx context.Context, task types.Task) context.Context {
	// 添加任务相关信息到上下文
	taskCtx := context.WithValue(ctx, "task_id", task.ID)
	taskCtx = context.WithValue(taskCtx, "agent_id", r.agent.id)

	// 设置超时
	if !task.Deadline.IsZero() {
		var cancel context.CancelFunc
		taskCtx, cancel = context.WithDeadline(taskCtx, task.Deadline)
		go func() {
			<-taskCtx.Done()
			cancel()
		}()
	}

	return taskCtx
}

// executeTask 执行具体任务
func (r *Runtime) executeTask(ctx context.Context, task types.Task) (types.Result, error) {
	r.executionMu.Lock()
	defer r.executionMu.Unlock()

	// 任务类型路由
	switch task.Type {
	case "conversation":
		return r.handleConversation(ctx, task)
	case "research":
		return r.handleResearch(ctx, task)
	case "analysis":
		return r.handleAnalysis(ctx, task)
	default:
		return types.Result{}, fmt.Errorf("unknown task type: %s", task.Type)
	}
}

// 不同类型任务的处理函数

// handleConversation 处理对话类型任务
func (r *Runtime) handleConversation(ctx context.Context, task types.Task) (types.Result, error) {
	// 从任务参数中获取必要信息
	input, ok := task.Parameters["input"].(string)
	if !ok {
		return types.Result{}, fmt.Errorf("missing required parameter: input")
	}

	// TODO: 实现对话处理逻辑
	// 1. 获取对话历史
	// 2. 调用LLM生成回复
	// 3. 更新对话历史

	// 示例响应
	response := fmt.Sprintf("This is a response to: %s", input)

	return types.Result{
		Data: map[string]interface{}{
			"response": response,
		},
		Metadata: map[string]interface{}{
			"tokens_used": 100, // 假设值
			"model":       "gpt-4",
		},
		Timestamp: time.Now(),
	}, nil
}

// handleResearch 处理研究类型任务
func (r *Runtime) handleResearch(ctx context.Context, task types.Task) (types.Result, error) {
	// 从任务参数中获取必要信息
	topics, ok := task.Parameters["topics"].([]string)
	if !ok {
		return types.Result{}, fmt.Errorf("missing required parameter: topics")
	}

	// TODO: 实现研究处理逻辑
	// 1. 使用工具搜索信息
	// 2. 整合结果
	// 3. 生成报告

	// 示例响应
	results := make(map[string]interface{})
	for _, topic := range topics {
		results[topic] = fmt.Sprintf("Research results for %s", topic)
	}

	return types.Result{
		Data: map[string]interface{}{
			"research_results": results,
		},
		Metadata: map[string]interface{}{
			"topics_count":   len(topics),
			"search_time_ms": 1500, // 假设值
		},
		Timestamp: time.Now(),
	}, nil
}

// handleAnalysis 处理分析类型任务
func (r *Runtime) handleAnalysis(ctx context.Context, task types.Task) (types.Result, error) {
	// 从任务参数中获取必要信息
	data, ok := task.Parameters["data"]
	if !ok {
		return types.Result{}, fmt.Errorf("missing required parameter: data")
	}

	// TODO: 实现分析处理逻辑
	// 1. 数据预处理
	// 2. 应用分析模型
	// 3. 生成结果

	// 示例响应
	return types.Result{
		Data: map[string]interface{}{
			"analysis_result":    "Analysis completed successfully",
			"summary":            "This is a summary of the analysis",
			"analyzed_data_size": fmt.Sprintf("%T with %v elements", data, len(fmt.Sprintf("%v", data))),
		},
		Metadata: map[string]interface{}{
			"data_points":        100, // 假设值
			"processing_time_ms": 500,
		},
		Timestamp: time.Now(),
	}, nil
}

// 工具调用和辅助函数

// callTool 调用指定工具
func (r *Runtime) callTool(ctx context.Context, toolID string, params map[string]interface{}) (interface{}, error) {
	// 查找工具
	var tool tool.Tool
	for _, t := range r.tools {
		if t.ID() == toolID {
			tool = t
			break
		}
	}

	if tool == nil {
		return nil, fmt.Errorf("tool not found: %s", toolID)
	}

	// 验证参数
	if err := tool.Validate(params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// 记录工具调用事件
	r.recordEvent(ctx, "tool_call_started", map[string]interface{}{
		"tool_id": toolID,
		"params":  params,
	})

	// 执行工具
	startTime := time.Now()
	result, err := tool.Execute(ctx, params)
	duration := time.Since(startTime)

	// 记录工具调用结果
	if err != nil {
		r.recordEvent(ctx, "tool_call_failed", map[string]interface{}{
			"tool_id":  toolID,
			"duration": duration.Milliseconds(),
			"error":    err.Error(),
		})
		return nil, err
	}

	r.recordEvent(ctx, "tool_call_completed", map[string]interface{}{
		"tool_id":  toolID,
		"duration": duration.Milliseconds(),
	})

	// 存储工具调用记忆
	if r.memory != nil {
		mem := types.Memory{
			Type: types.MemoryType("short_term"),
			Content: map[string]interface{}{
				"tool_id": toolID,
				"params":  params,
				"result":  result,
			},
			Importance: 0.5, // 中等重要性
			Context: map[string]interface{}{
				"agent_id": r.agent.id,
				"tool_id":  toolID,
			},
			Timestamp: time.Now(),
		}

		go func(m types.Memory) {
			if err := r.memory.Store(context.Background(), m); err != nil {
				fmt.Printf("Failed to store tool call memory: %v\n", err)
			}
		}(mem)
	}

	return result, nil
}

// recordEvent 记录事件
func (r *Runtime) recordEvent(ctx context.Context, eventType string, data interface{}) {
	event := NewEvent(uuid.New().String(), eventType, data)

	// TODO: 将事件记录到事件流
	// 临时打印事件，避免未使用变量错误
	if r.agent != nil && r.agent.id != "" {
		fmt.Printf("Agent %s event: %s - %v\n", r.agent.id, event.Type, event.Timestamp)
	}
}

// recordMemory 记录到记忆
func (r *Runtime) recordMemory(ctx context.Context, content interface{}, importance float64) error {
	if r.memory == nil {
		return fmt.Errorf("memory store not available")
	}

	mem := types.Memory{
		ID:         uuid.New().String(),
		Type:       types.MemoryType("short_term"),
		Content:    content,
		Importance: importance,
		Context: map[string]interface{}{
			"agent_id": r.agent.id,
		},
		Timestamp: time.Now(),
	}

	return r.memory.Store(ctx, mem)
}

// retrieveMemory 检索记忆
func (r *Runtime) retrieveMemory(ctx context.Context, query types.MemoryQuery) ([]types.Memory, error) {
	if r.memory == nil {
		return nil, fmt.Errorf("memory store not available")
	}

	return r.memory.Recall(ctx, query)
}

// retrieveRecentMemories 检索最近的记忆
func (r *Runtime) retrieveRecentMemories(ctx context.Context, limit int) ([]types.Memory, error) {
	query := types.MemoryQuery{
		Limit: limit,
	}
	return r.retrieveMemory(ctx, query)
}

// retrieveMemoriesByType 按类型检索记忆
func (r *Runtime) retrieveMemoriesByType(ctx context.Context, memType types.MemoryType, limit int) ([]types.Memory, error) {
	query := types.MemoryQuery{
		Type:  memType,
		Limit: limit,
	}
	return r.retrieveMemory(ctx, query)
}

// retrieveMemoriesByContext 按上下文检索记忆
func (r *Runtime) retrieveMemoriesByContext(ctx context.Context, contextKey string, contextValue interface{}, limit int) ([]types.Memory, error) {
	query := types.MemoryQuery{
		Context: map[string]interface{}{
			contextKey: contextValue,
		},
		Limit: limit,
	}
	return r.retrieveMemory(ctx, query)
}

// retrieveImportantMemories 检索重要的记忆
func (r *Runtime) retrieveImportantMemories(ctx context.Context, minImportance float64, limit int) ([]types.Memory, error) {
	query := types.MemoryQuery{
		Importance: minImportance,
		Limit:      limit,
	}
	return r.retrieveMemory(ctx, query)
}

// retrieveKnowledge 从知识库检索信息
func (r *Runtime) retrieveKnowledge(ctx context.Context, query string, limit int) ([]types.Knowledge, error) {
	if r.knowledge == nil {
		return nil, fmt.Errorf("knowledge context not available")
	}

	// 执行语义搜索
	results, err := r.knowledge.SemanticSearch(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("semantic search failed: %w", err)
	}

	return results, nil
}

// queryKnowledge 按条件查询知识
func (r *Runtime) queryKnowledge(ctx context.Context, knowledgeType string, filter map[string]interface{}, limit int) ([]types.Knowledge, error) {
	if r.knowledge == nil {
		return nil, fmt.Errorf("knowledge context not available")
	}

	query := types.Query{
		Type:   knowledgeType,
		Filter: filter,
		Limit:  limit,
	}

	results, err := r.knowledge.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("knowledge query failed: %w", err)
	}

	return results, nil
}

// addKnowledge 向知识库添加知识
func (r *Runtime) addKnowledge(ctx context.Context, content interface{}, knowledgeType string, metadata map[string]interface{}) error {
	if r.knowledge == nil {
		return fmt.Errorf("knowledge context not available")
	}

	k := types.Knowledge{
		ID:       uuid.New().String(),
		Type:     knowledgeType,
		Content:  content,
		Metadata: metadata,
	}

	return r.knowledge.AddKnowledge(ctx, k)
}

// getRelevantKnowledge 获取与文本相关的知识
func (r *Runtime) getRelevantKnowledge(ctx context.Context, text string, limit int) ([]types.Knowledge, error) {
	if r.knowledge == nil {
		return nil, fmt.Errorf("knowledge context not available")
	}

	return r.knowledge.GetRelevantKnowledge(ctx, text, limit)
}
