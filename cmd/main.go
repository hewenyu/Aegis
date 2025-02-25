package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hewenyu/Aegis/internal/agent"
	"github.com/hewenyu/Aegis/internal/knowledge"
	"github.com/hewenyu/Aegis/internal/memory"
	"github.com/hewenyu/Aegis/internal/tool"
)

func main() {
	// 创建上下文
	ctx := context.Background()

	// 初始化组件
	toolManager := tool.NewManager()
	memoryManager := memory.NewManager()

	// 创建向量存储和知识库
	embedder := knowledge.NewMockEmbedder(128)
	vectorStore := knowledge.NewInMemoryVectorStore(128, embedder)
	knowledgeBase := knowledge.NewBase(vectorStore)

	// 创建Agent管理器
	agentManager := agent.NewManager(toolManager, memoryManager, knowledgeBase)

	// 注册一些示例工具
	registerExampleTools(ctx, toolManager)

	// 创建Agent
	agentConfig := agent.AgentConfig{
		Name:         "ExampleAgent",
		Description:  "一个示例AI Agent",
		Capabilities: []string{"conversation", "research"},
		Model: agent.ModelConfig{
			Type:        "gpt-4",
			Temperature: 0.7,
			MaxTokens:   1000,
		},
		Tools: []agent.ToolConfig{
			{ID: "calculator"},
			{ID: "weather"},
		},
		Memory: agent.MemoryConfig{
			Type: "default",
			Size: 100,
		},
		Knowledge: agent.KnowledgeConfig{
			Type: "default",
		},
	}

	myAgent, err := agentManager.CreateAgent(ctx, agentConfig)
	if err != nil {
		log.Fatalf("创建Agent失败: %v", err)
	}

	// 订阅Agent事件
	eventCh, err := agentManager.SubscribeToEvents(ctx, myAgent.Status().ID)
	if err != nil {
		log.Fatalf("订阅事件失败: %v", err)
	}

	// 启动事件监听
	go func() {
		for event := range eventCh {
			fmt.Printf("事件: [%s] %s - %v\n", event.Type, event.ID, event.Timestamp)
		}
	}()

	// 创建并分配任务
	task := agent.Task{
		Type:        "conversation",
		Description: "回答用户问题",
		Parameters: map[string]interface{}{
			"input": "什么是AI Agent?",
		},
		Deadline: time.Now().Add(30 * time.Second),
	}

	fmt.Printf("分配任务给Agent: %s\n", myAgent.Status().ID)
	err = agentManager.AssignTask(ctx, myAgent.Status().ID, task)
	if err != nil {
		log.Fatalf("分配任务失败: %v", err)
	}

	// 等待任务完成
	time.Sleep(2 * time.Second)

	// 获取任务状态
	taskStatus, err := agentManager.GetTaskStatus(ctx, task.ID)
	if err != nil {
		log.Fatalf("获取任务状态失败: %v", err)
	}

	fmt.Printf("任务状态: %s\n", taskStatus.Status)
	if taskStatus.Status == "completed" {
		result := taskStatus.Result.(agent.Result)
		fmt.Printf("任务结果: %v\n", result.Data)
	}

	// 销毁Agent
	fmt.Printf("销毁Agent: %s\n", myAgent.Status().ID)
	err = agentManager.DestroyAgent(ctx, myAgent.Status().ID)
	if err != nil {
		log.Fatalf("销毁Agent失败: %v", err)
	}

	fmt.Println("示例完成")
}

// 注册示例工具
func registerExampleTools(ctx context.Context, manager tool.Manager) {
	// 注册计算器工具
	calcTool := &calculatorTool{
		id:          "calculator",
		name:        "Calculator",
		description: "执行基本数学计算",
		version:     "1.0",
	}
	manager.RegisterTool(ctx, calcTool)

	// 注册天气工具
	weatherTool := &weatherTool{
		id:          "weather",
		name:        "Weather",
		description: "获取天气信息",
		version:     "1.0",
	}
	manager.RegisterTool(ctx, weatherTool)
}

// 计算器工具实现
type calculatorTool struct {
	id          string
	name        string
	description string
	version     string
}

func (t *calculatorTool) ID() string          { return t.id }
func (t *calculatorTool) Name() string        { return t.name }
func (t *calculatorTool) Description() string { return t.description }
func (t *calculatorTool) Version() string     { return t.version }

func (t *calculatorTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	operation, ok := params["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("missing operation parameter")
	}

	a, ok := params["a"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid parameter a")
	}

	b, ok := params["b"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid parameter b")
	}

	var result float64
	switch operation {
	case "add":
		result = a + b
	case "subtract":
		result = a - b
	case "multiply":
		result = a * b
	case "divide":
		if b == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		result = a / b
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}

	return map[string]interface{}{
		"result": result,
	}, nil
}

func (t *calculatorTool) Validate(params map[string]interface{}) error {
	if _, ok := params["operation"].(string); !ok {
		return fmt.Errorf("missing operation parameter")
	}
	if _, ok := params["a"].(float64); !ok {
		return fmt.Errorf("missing or invalid parameter a")
	}
	if _, ok := params["b"].(float64); !ok {
		return fmt.Errorf("missing or invalid parameter b")
	}
	return nil
}

// 天气工具实现
type weatherTool struct {
	id          string
	name        string
	description string
	version     string
}

func (t *weatherTool) ID() string          { return t.id }
func (t *weatherTool) Name() string        { return t.name }
func (t *weatherTool) Description() string { return t.description }
func (t *weatherTool) Version() string     { return t.version }

func (t *weatherTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	location, ok := params["location"].(string)
	if !ok {
		return nil, fmt.Errorf("missing location parameter")
	}

	// 这里只是模拟，实际应用中应该调用天气API
	weatherData := map[string]interface{}{
		"location":    location,
		"temperature": 22.5,
		"condition":   "晴天",
		"humidity":    65,
		"wind":        "东北风 3级",
	}

	return weatherData, nil
}

func (t *weatherTool) Validate(params map[string]interface{}) error {
	if _, ok := params["location"].(string); !ok {
		return fmt.Errorf("missing location parameter")
	}
	return nil
}
