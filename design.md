# AI Agent 组件设计文档 (Golang 实现)

## 1. 项目结构
```
├── cmd/
│   └── main.go                 # 应用程序入口
├── internal/
│   ├── agent/                  # Agent 核心实现
│   │   ├── manager.go         
│   │   ├── runtime.go
│   │   └── types.go
│   ├── tool/                   # 工具管理
│   │   ├── manager.go
│   │   ├── registry.go
│   │   └── types.go
│   ├── knowledge/              # 知识库
│   │   ├── base.go
│   │   ├── vector.go
│   │   └── types.go
│   └── memory/                 # 记忆系统
│       ├── manager.go
│       ├── store.go
│       └── types.go
├── pkg/
│   ├── config/                 # 配置
│   ├── logger/                 # 日志
│   └── utils/                  # 工具函数
└── api/                        # API 定义
    └── proto/                  # gRPC 定义
```

## 2. 核心组件实现

### 2.1 Agent Manager

#### 2.1.1 接口定义
```go
// internal/agent/types.go
package agent

import (
    "context"
    "time"
)

type AgentConfig struct {
    ID          string
    Name        string
    Description string
    Capabilities []string
    Model       ModelConfig
    Tools       []ToolConfig
    Memory      MemoryConfig
    Knowledge   KnowledgeConfig
}

type ModelConfig struct {
    Type        string
    Temperature float64
    MaxTokens   int
}

type Agent interface {
    Initialize(ctx context.Context) error
    Execute(ctx context.Context, task Task) (Result, error)
    Stop(ctx context.Context) error
    Status() AgentStatus
}

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

type Task struct {
    ID          string
    Type        string
    Description string
    Parameters  map[string]interface{}
    Deadline    time.Time
}

type TaskStatus struct {
    ID        string
    Status    string
    Progress  float64
    Result    interface{}
    Error     error
    StartTime time.Time
    EndTime   time.Time
}
```

#### 2.1.2 实现
```go
// internal/agent/manager.go
package agent

import (
    "context"
    "sync"

    "github.com/google/uuid"
    "github.com/hewenyu/Aegis/internal/tool"
    "github.com/hewenyu/Aegis/internal/memory"
    "github.com/hewenyu/Aegis/internal/knowledge"
)

type manager struct {
    agents    sync.Map
    tasks     sync.Map
    toolMgr   tool.Manager
    memoryMgr memory.Manager
    knowledge knowledge.Base
}

func NewManager(toolMgr tool.Manager, memoryMgr memory.Manager, kb knowledge.Base) Manager {
    return &manager{
        toolMgr:   toolMgr,
        memoryMgr: memoryMgr,
        knowledge: kb,
    }
}

func (m *manager) CreateAgent(ctx context.Context, config AgentConfig) (Agent, error) {
    if config.ID == "" {
        config.ID = uuid.New().String()
    }

    // 验证配置
    if err := m.validateConfig(config); err != nil {
        return nil, err
    }

    // 初始化依赖
    tools, err := m.toolMgr.GetTools(ctx, config.Tools)
    if err != nil {
        return nil, err
    }

    memory, err := m.memoryMgr.CreateStore(ctx, config.Memory)
    if err != nil {
        return nil, err
    }

    kb, err := m.knowledge.CreateContext(ctx, config.Knowledge)
    if err != nil {
        return nil, err
    }

    // 创建 agent
    agent := NewAgent(config, tools, memory, kb)
    if err := agent.Initialize(ctx); err != nil {
        return nil, err
    }

    m.agents.Store(config.ID, agent)
    return agent, nil
}

func (m *manager) AssignTask(ctx context.Context, agentID string, task Task) error {
    agent, ok := m.agents.Load(agentID)
    if !ok {
        return ErrAgentNotFound
    }

    // 存储任务
    m.tasks.Store(task.ID, TaskStatus{
        ID:        task.ID,
        Status:    "pending",
        StartTime: time.Now(),
    })

    // 异步执行任务
    go func() {
        result, err := agent.(Agent).Execute(ctx, task)
        
        // 更新任务状态
        status := TaskStatus{
            ID:      task.ID,
            EndTime: time.Now(),
        }
        
        if err != nil {
            status.Status = "failed"
            status.Error = err
        } else {
            status.Status = "completed"
            status.Result = result
        }
        
        m.tasks.Store(task.ID, status)
    }()

    return nil
}

// ... 其他方法实现
```

### 2.2 Tool Manager

#### 2.2.1 接口定义
```go
// internal/tool/types.go
package tool

import (
    "context"
)

type Tool interface {
    ID() string
    Name() string
    Description() string
    Version() string
    Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
    Validate(params map[string]interface{}) error
}

type Manager interface {
    RegisterTool(ctx context.Context, tool Tool) error
    UnregisterTool(ctx context.Context, toolID string) error
    GetTool(ctx context.Context, toolID string) (Tool, error)
    GetTools(ctx context.Context, filter ToolFilter) ([]Tool, error)
    ExecuteTool(ctx context.Context, toolID string, params map[string]interface{}) (interface{}, error)
}

type ToolFilter struct {
    Categories []string
    Tags      []string
    Version   string
}
```

#### 2.2.2 实现
```go
// internal/tool/manager.go
package tool

import (
    "context"
    "sync"
)

type manager struct {
    tools       sync.Map
    permissions sync.Map
}

func NewManager() Manager {
    return &manager{}
}

func (m *manager) RegisterTool(ctx context.Context, tool Tool) error {
    if tool == nil {
        return ErrInvalidTool
    }

    // 验证工具
    if err := m.validateTool(tool); err != nil {
        return err
    }

    // 存储工具
    m.tools.Store(tool.ID(), tool)
    return nil
}

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
```

### 2.3 Knowledge Base

#### 2.3.1 接口定义
```go
// internal/knowledge/types.go
package knowledge

import (
    "context"
)

type Knowledge struct {
    ID       string
    Type     string
    Content  interface{}
    Metadata map[string]interface{}
    Vector   []float32
}

type Base interface {
    AddKnowledge(ctx context.Context, k Knowledge) error
    UpdateKnowledge(ctx context.Context, id string, k Knowledge) error
    DeleteKnowledge(ctx context.Context, id string) error
    Query(ctx context.Context, q Query) ([]Knowledge, error)
    SemanticSearch(ctx context.Context, text string) ([]Knowledge, error)
}

type Query struct {
    Type   string
    Filter map[string]interface{}
    Sort   []SortField
    Limit  int
}

type SortField struct {
    Field     string
    Ascending bool
}
```

### 2.4 Memory Manager

#### 2.4.1 接口定义
```go
// internal/memory/types.go
package memory

import (
    "context"
    "time"
)

type Memory struct {
    ID         string
    Type       MemoryType
    Content    interface{}
    Timestamp  time.Time
    Importance float64
    Context    map[string]interface{}
}

type MemoryType string

const (
    ShortTerm  MemoryType = "short_term"
    LongTerm   MemoryType = "long_term"
    Working    MemoryType = "working"
)

type Manager interface {
    Store(ctx context.Context, m Memory) error
    Recall(ctx context.Context, query MemoryQuery) ([]Memory, error)
    Forget(ctx context.Context, filter MemoryFilter) error
    Consolidate(ctx context.Context) error
}

type MemoryQuery struct {
    Type       MemoryType
    TimeRange  TimeRange
    Importance float64
    Limit      int
}

type TimeRange struct {
    Start time.Time
    End   time.Time
}
```

## 3. 使用示例

### 3.1 创建和使用 Agent
```go
package main

import (
    "context"
    "log"

    "github.com/hewenyu/Aegis/internal/agent"
    "github.com/hewenyu/Aegis/internal/tool"
    "github.com/hewenyu/Aegis/internal/memory"
    "github.com/hewenyu/Aegis/internal/knowledge"
)

func main() {
    ctx := context.Background()

    // 初始化管理器
    toolMgr := tool.NewManager()
    memoryMgr := memory.NewManager()
    kb := knowledge.NewBase()
    agentMgr := agent.NewManager(toolMgr, memoryMgr, kb)

    // 创建 Agent 配置
    config := agent.AgentConfig{
        Name:        "ResearchAssistant",
        Capabilities: []string{"web_search", "document_analysis"},
        Model: agent.ModelConfig{
            Type:        "gpt-4",
            Temperature: 0.7,
        },
        Tools: []agent.ToolConfig{
            {ID: "web_search"},
            {ID: "document_reader"},
        },
    }

    // 创建 Agent
    agent, err := agentMgr.CreateAgent(ctx, config)
    if err != nil {
        log.Fatal(err)
    }

    // 创建任务
    task := agent.Task{
        ID:          "task-1",
        Type:        "research",
        Description: "Research AI agents",
        Parameters: map[string]interface{}{
            "topics":    []string{"AI agents", "multi-agent systems"},
            "timeframe": "last 6 months",
        },
    }

    // 分配任务
    if err := agentMgr.AssignTask(ctx, agent.ID(), task); err != nil {
        log.Fatal(err)
    }

    // 监控任务状态
    statusChan, err := agentMgr.SubscribeToEvents(ctx, agent.ID())
    if err != nil {
        log.Fatal(err)
    }

    for event := range statusChan {
        log.Printf("Event: %+v\n", event)
    }
}
```

### 3.2 创建自定义工具
```go
package main

import (
    "context"

    "github.com/hewenyu/Aegis/internal/tool"
)

type AnalysisTool struct {
    id          string
    name        string
    description string
    version     string
}

func NewAnalysisTool() tool.Tool {
    return &AnalysisTool{
        id:          "custom_analysis_tool",
        name:        "Custom Analysis Tool",
        description: "Performs specialized data analysis",
        version:     "1.0.0",
    }
}

func (t *AnalysisTool) ID() string          { return t.id }
func (t *AnalysisTool) Name() string        { return t.name }
func (t *AnalysisTool) Description() string { return t.description }
func (t *AnalysisTool) Version() string     { return t.version }

func (t *AnalysisTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    // 实现分析逻辑
    data := params["data"].([]interface{})
    method := params["method"].(string)

    // 执行分析
    result := analyzeData(data, method)
    return result, nil
}

func (t *AnalysisTool) Validate(params map[string]interface{}) error {
    // 参数验证
    if _, ok := params["data"].([]interface{}); !ok {
        return tool.ErrInvalidParameter
    }
    if _, ok := params["method"].(string); !ok {
        return tool.ErrInvalidParameter
    }
    return nil
}

func main() {
    ctx := context.Background()
    
    // 创建工具管理器
    toolMgr := tool.NewManager()

    // 注册自定义工具
    customTool := NewAnalysisTool()
    if err := toolMgr.RegisterTool(ctx, customTool); err != nil {
        panic(err)
    }

    // 使用工具
    params := map[string]interface{}{
        "data":   []interface{}{1, 2, 3, 4, 5},
        "method": "statistical",
    }

    result, err := toolMgr.ExecuteTool(ctx, customTool.ID(), params)
    if err != nil {
        panic(err)
    }

    // 处理结果
    // ...
}
```

## 4. 错误处理

```go
// pkg/errors/errors.go
package errors

import (
    "errors"
    "fmt"
)

var (
    ErrAgentNotFound     = errors.New("agent not found")
    ErrToolNotFound      = errors.New("tool not found")
    ErrInvalidConfig     = errors.New("invalid configuration")
    ErrInvalidParameter  = errors.New("invalid parameter")
    ErrOperationTimeout  = errors.New("operation timeout")
    ErrNotImplemented    = errors.New("not implemented")
)

type AgentError struct {
    AgentID string
    Op      string
    Err     error
}

func (e *AgentError) Error() string {
    return fmt.Sprintf("agent %s: %s: %v", e.AgentID, e.Op, e.Err)
}
```

## 5. 配置管理

```go
// pkg/config/config.go
package config

import (
    "github.com/spf13/viper"
)

type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Vector   VectorConfig
    LLM      LLMConfig
    Logging  LogConfig
}

type ServerConfig struct {
    Port     int
    Host     string
    TimeoutS int
}

type DatabaseConfig struct {
    Type     string
    Host     string
    Port     int
    User     string
    Password string
    Database string
}

func Load() (*Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath(".")
    viper.AddConfigPath("./config")

    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }

    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, err
    }

    return &config, nil
}
```

## 6. 日志处理

```go
// pkg/logger/logger.go
package logger

import (
    "go.uber.org/zap"
)

var (
    log *zap.Logger
)

func Init(config Config) error {
    var err error
    if config.Development {
        log, err = zap.NewDevelopment()
    } else {
        log, err = zap.NewProduction()
    }
    return err
}

func Info(msg string, fields ...zap.Field) {
    log.Info(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
    log.Error(msg, fields...)
}

func Debug(msg string, fields ...zap.Field) {
    log.Debug(msg, fields...)
}
```

## 7. 性能优化

1. 使用连接池
```go
// pkg/database/pool.go
package database

import (
    "context"
    "time"

    "github.com/jackc/pgx/v4/pgxpool"
)

type Pool struct {
    *pgxpool.Pool
}

func NewPool(ctx context.Context, config Config) (*Pool, error) {
    poolConfig, err := pgxpool.ParseConfig(config.URL)
    if err != nil {
        return nil, err
    }

    poolConfig.MaxConns = 10
    poolConfig.MinConns = 2
    poolConfig.MaxConnLifetime = time.Hour
    poolConfig.MaxConnIdleTime = 30 * time.Minute

    pool, err := pgxpool.ConnectConfig(ctx, poolConfig)
    if err != nil {
        return nil, err
    }

    return &Pool{pool}, nil
}
```

2. 使用缓存
```go
// pkg/cache/cache.go
package cache

import (
    "context"
    "time"

    "github.com/go-redis/redis/v8"
)

type Cache struct {
    client *redis.Client
}

func NewCache(config Config) *Cache {
    client := redis.NewClient(&redis.Options{
        Addr:     config.Addr,
        Password: config.Password,
        DB:       config.DB,
    })

    return &Cache{client: client}
}

func (c *Cache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
    return c.client.Set(ctx, key, value, expiration).Err()
}

func (c *Cache) Get(ctx context.Context, key string) (string, error) {
    return c.client.Get(ctx, key).Result()
}
```

## 8. 测试

```go
// internal/agent/manager_test.go
package agent

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

type MockTool struct {
    mock.Mock
}

func TestCreateAgent(t *testing.T) {
    ctx := context.Background()
    
    // 创建 mock
    mockTool := new(MockTool)
    mockMemory := new(MockMemory)
    mockKB := new(MockKnowledgeBase)

    // 设置期望
    mockTool.On("GetTools", mock.Anything, mock.Anything).Return([]Tool{}, nil)
    mockMemory.On("CreateStore", mock.Anything, mock.Anything).Return(nil, nil)
    mockKB.On("CreateContext", mock.Anything, mock.Anything).Return(nil, nil)

    // 创建管理器
    mgr := NewManager(mockTool, mockMemory, mockKB)

    // 测试创建 agent
    config := AgentConfig{
        Name: "TestAgent",
    }

    agent, err := mgr.CreateAgent(ctx, config)
    
    // 验证结果
    assert.NoError(t, err)
    assert.NotNil(t, agent)
    mockTool.AssertExpectations(t)
    mockMemory.AssertExpectations(t)
    mockKB.AssertExpectations(t)
}
```
