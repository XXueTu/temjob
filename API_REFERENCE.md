# TemJob API 参考文档

本文档详细说明了 TemJob 的 RESTful API 接口和 Go SDK 的使用方法。

## 📡 RESTful API

### 基础信息

- **Base URL**: `http://localhost:8088`
- **Content-Type**: `application/json`
- **响应格式**: JSON

### 认证

当前版本暂不支持认证，后续版本将添加 JWT 认证支持。

---

## 🔄 工作流 API

### 1. 获取工作流列表

```http
GET /api/v1/workflows
```

#### 查询参数

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| limit | int | 否 | 20 | 返回数量限制 |
| offset | int | 否 | 0 | 偏移量 |

#### 响应示例

```json
{
  "workflows": [
    {
      "id": "wf_12345",
      "name": "data_processing",
      "state": "completed",
      "created_at": "2023-12-01T10:00:00Z",
      "started_at": "2023-12-01T10:00:01Z",
      "ended_at": "2023-12-01T10:02:30Z",
      "tasks": ["task_1", "task_2", "task_3"]
    }
  ]
}
```

### 2. 获取工作流详情

```http
GET /api/v1/workflows/{workflow_id}
```

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| workflow_id | string | 是 | 工作流 ID |

#### 响应示例

```json
{
  "id": "wf_12345",
  "name": "data_processing",
  "input": {
    "input_file": "data.csv",
    "output_dir": "/tmp/output"
  },
  "output": {
    "processed_records": 1000,
    "output_file": "processed_data.csv",
    "summary": "Processing completed successfully"
  },
  "state": "completed",
  "tasks": ["task_1", "task_2", "task_3"],
  "created_at": "2023-12-01T10:00:00Z",
  "started_at": "2023-12-01T10:00:01Z",
  "ended_at": "2023-12-01T10:02:30Z"
}
```

### 3. 取消工作流

```http
POST /api/v1/workflows/{workflow_id}/cancel
```

#### 响应示例

```json
{
  "message": "Workflow canceled successfully"
}
```

### 4. 获取工作流任务列表

```http
GET /api/v1/workflows/{workflow_id}/tasks
```

#### 响应示例

```json
{
  "tasks": [
    {
      "id": "task_1",
      "workflow_id": "wf_12345",
      "type": "validate_input",
      "state": "completed",
      "input": {
        "input_file": "data.csv"
      },
      "output": {
        "validated": true,
        "file_size": 1024
      },
      "created_at": "2023-12-01T10:00:00Z",
      "started_at": "2023-12-01T10:00:01Z",
      "completed_at": "2023-12-01T10:00:05Z",
      "retry_count": 0,
      "max_retries": 3
    }
  ]
}
```

---

## 📋 任务 API

### 1. 获取任务详情

```http
GET /api/v1/tasks/{task_id}
```

#### 响应示例

```json
{
  "id": "task_1",
  "workflow_id": "wf_12345",
  "type": "validate_input",
  "input": {
    "input_file": "data.csv"
  },
  "output": {
    "validated": true,
    "file_size": 1024
  },
  "state": "completed",
  "error": "",
  "retry_count": 0,
  "max_retries": 3,
  "created_at": "2023-12-01T10:00:00Z",
  "started_at": "2023-12-01T10:00:01Z",
  "completed_at": "2023-12-01T10:00:05Z",
  "worker_id": "worker_abc123"
}
```

---

## 📊 统计 API

### 1. 获取系统统计

```http
GET /api/v1/stats
```

#### 响应示例

```json
{
  "total_workflows": 150,
  "running": 5,
  "completed": 140,
  "failed": 3,
  "canceled": 2,
  "pending": 0
}
```

---

## 🚨 错误响应

### 错误格式

```json
{
  "error": "workflow not found",
  "code": "WORKFLOW_NOT_FOUND",
  "message": "Workflow with ID 'wf_invalid' was not found"
}
```

### 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 请求成功 |
| 400 | 请求参数错误 |
| 404 | 资源未找到 |
| 500 | 服务器内部错误 |

---

## 🔧 Go SDK 参考

### 客户端配置

```go
type ClientConfig struct {
    RedisAddr     string  // Redis 地址，如 "localhost:6379"
    RedisPassword string  // Redis 密码
    RedisDB       int     // Redis 数据库编号
}
```

### 创建客户端

```go
client, err := sdk.NewClient(sdk.ClientConfig{
    RedisAddr:     "localhost:6379",
    RedisPassword: "",
    RedisDB:       0,
})
if err != nil {
    log.Fatal("Failed to create client:", err)
}
defer client.Close()
```

---

## 📝 任务处理器

### TaskHandler 接口

```go
type TaskHandler interface {
    Handle(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
}
```

### SimpleTaskHandler

```go
func SimpleTaskHandler(fn func(input map[string]interface{}) (map[string]interface{}, error)) TaskHandler
```

#### 使用示例

```go
client.RegisterTaskHandler("send_email", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
    recipient := input["recipient"].(string)
    subject := input["subject"].(string)
    
    // 发送邮件的业务逻辑
    err := sendEmail(recipient, subject)
    if err != nil {
        return nil, fmt.Errorf("failed to send email: %w", err)
    }
    
    return map[string]interface{}{
        "email_sent": true,
        "sent_at": time.Now().Format(time.RFC3339),
    }, nil
}))
```

---

## 🏗️ 工作流构建器

### WorkflowBuilder

```go
type WorkflowBuilder struct {
    name  string
    tasks map[string]TaskDefinition
    flow  []WorkflowStep
}
```

### 方法

#### NewWorkflowBuilder

```go
func NewWorkflowBuilder(name string) *WorkflowBuilder
```

创建新的工作流构建器。

#### AddTask

```go
func (wb *WorkflowBuilder) AddTask(taskType string, handler TaskHandler, maxRetries int) *WorkflowBuilder
```

添加任务定义。

**参数说明：**
- `taskType`: 任务类型标识符
- `handler`: 任务处理器
- `maxRetries`: 最大重试次数

#### AddStep

```go
func (wb *WorkflowBuilder) AddStep(taskType string) *StepBuilder
```

添加执行步骤。

### StepBuilder

#### DependsOn

```go
func (sb *StepBuilder) DependsOn(dependencies ...string) *StepBuilder
```

设置任务依赖关系。

#### Then

```go
func (sb *StepBuilder) Then() *WorkflowBuilder
```

完成步骤构建，返回工作流构建器。

#### Build

```go
func (wb *WorkflowBuilder) Build() WorkflowDefinition
```

构建最终的工作流定义。

### 完整示例

```go
workflowDef := sdk.NewWorkflowBuilder("order_processing").
    // 添加任务定义
    AddTask("validate_order", sdk.SimpleTaskHandler(validateOrder), 3).
    AddTask("process_payment", sdk.SimpleTaskHandler(processPayment), 3).
    AddTask("ship_order", sdk.SimpleTaskHandler(shipOrder), 2).
    AddTask("send_notification", sdk.SimpleTaskHandler(sendNotification), 2).
    
    // 定义执行流程
    AddStep("validate_order").Then().
    AddStep("process_payment").DependsOn("validate_order").Then().
    AddStep("ship_order").DependsOn("process_payment").Then().
    AddStep("send_notification").DependsOn("ship_order").Then().
    Build()

client.RegisterWorkflow(workflowDef)
```

---

## 🎯 Client 接口

### RegisterTaskHandler

```go
func (c *Client) RegisterTaskHandler(taskType string, handler TaskHandler)
```

注册任务处理器。

### RegisterWorkflow

```go
func (c *Client) RegisterWorkflow(definition WorkflowDefinition)
```

注册工作流定义。

### SubmitWorkflow

```go
func (c *Client) SubmitWorkflow(ctx context.Context, workflowName string, input map[string]interface{}) (string, error)
```

提交工作流执行请求。

**返回值：**
- `string`: 工作流 ID
- `error`: 错误信息

### GetWorkflow

```go
func (c *Client) GetWorkflow(ctx context.Context, workflowID string) (*Workflow, error)
```

获取工作流状态信息。

### StartEngine

```go
func (c *Client) StartEngine(ctx context.Context) error
```

启动工作流引擎。

### StartWorker

```go
func (c *Client) StartWorker(ctx context.Context) error
```

启动任务工作器。

---

## 📊 数据结构

### Workflow

```go
type Workflow struct {
    ID        string                 `json:"id"`
    Name      string                 `json:"name"`
    Input     map[string]interface{} `json:"input"`
    Output    map[string]interface{} `json:"output"`
    State     WorkflowState          `json:"state"`
    Tasks     []string               `json:"tasks"`
    CreatedAt time.Time              `json:"created_at"`
    StartedAt *time.Time             `json:"started_at"`
    EndedAt   *time.Time             `json:"ended_at"`
}
```

### Task

```go
type Task struct {
    ID          string                 `json:"id"`
    WorkflowID  string                 `json:"workflow_id"`
    Type        string                 `json:"type"`
    Input       map[string]interface{} `json:"input"`
    Output      map[string]interface{} `json:"output"`
    State       TaskState              `json:"state"`
    Error       string                 `json:"error"`
    RetryCount  int                    `json:"retry_count"`
    MaxRetries  int                    `json:"max_retries"`
    CreatedAt   time.Time              `json:"created_at"`
    StartedAt   *time.Time             `json:"started_at"`
    CompletedAt *time.Time             `json:"completed_at"`
    WorkerID    string                 `json:"worker_id"`
}
```

### 状态枚举

#### WorkflowState

```go
type WorkflowState string

const (
    WorkflowStatePending   WorkflowState = "pending"
    WorkflowStateRunning   WorkflowState = "running"
    WorkflowStateCompleted WorkflowState = "completed"
    WorkflowStateFailed    WorkflowState = "failed"
    WorkflowStateCanceled  WorkflowState = "canceled"
)
```

#### TaskState

```go
type TaskState string

const (
    TaskStatePending   TaskState = "pending"
    TaskStateRunning   TaskState = "running"
    TaskStateCompleted TaskState = "completed"
    TaskStateFailed    TaskState = "failed"
    TaskStateRetrying  TaskState = "retrying"
    TaskStateCanceled  TaskState = "canceled"
)
```

---

## 🔄 高级用法

### 条件执行

```go
client.RegisterTaskHandler("conditional_task", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
    condition := input["condition"].(bool)
    
    if !condition {
        // 跳过执行
        return map[string]interface{}{
            "skipped": true,
            "reason": "condition not met",
        }, nil
    }
    
    // 正常执行
    return map[string]interface{}{
        "executed": true,
    }, nil
}))
```

### 并行任务

```go
workflowDef := sdk.NewWorkflowBuilder("parallel_processing").
    AddTask("task_a", handlerA, 3).
    AddTask("task_b", handlerB, 3).
    AddTask("task_c", handlerC, 3).
    AddTask("merge_results", mergeHandler, 3).
    
    // task_a 和 task_b 可以并行执行
    AddStep("task_a").Then().
    AddStep("task_b").Then().
    
    // task_c 依赖 task_a
    AddStep("task_c").DependsOn("task_a").Then().
    
    // merge_results 依赖所有前置任务
    AddStep("merge_results").DependsOn("task_a", "task_b", "task_c").Then().
    Build()
```

### 错误处理

```go
client.RegisterTaskHandler("robust_task", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Task panic recovered: %v", r)
        }
    }()
    
    // 业务逻辑
    result, err := doSomething(input)
    if err != nil {
        // 返回结构化错误信息
        return nil, fmt.Errorf("business logic failed: %w", err)
    }
    
    return result, nil
}))
```

### 上下文传递

```go
client.RegisterTaskHandler("context_aware_task", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
    // 从前置任务获取数据
    previousResult := input["previous_result"]
    userID := input["user_id"].(string)
    
    // 处理业务逻辑
    result := processWithContext(userID, previousResult)
    
    // 传递给后续任务
    return map[string]interface{}{
        "user_id": userID,
        "current_result": result,
        "timestamp": time.Now().Unix(),
    }, nil
}))
```

---

## 📚 示例集合

### 数据处理流水线

```go
func DataProcessingPipeline() {
    client, _ := sdk.NewClient(sdk.ClientConfig{RedisAddr: "localhost:6379"})
    
    // ETL 流水线
    workflowDef := sdk.NewWorkflowBuilder("data_pipeline").
        AddTask("extract", sdk.SimpleTaskHandler(extractData), 3).
        AddTask("transform", sdk.SimpleTaskHandler(transformData), 3).
        AddTask("validate", sdk.SimpleTaskHandler(validateData), 3).
        AddTask("load", sdk.SimpleTaskHandler(loadData), 3).
        AddTask("notify", sdk.SimpleTaskHandler(sendNotification), 2).
        
        AddStep("extract").Then().
        AddStep("transform").DependsOn("extract").Then().
        AddStep("validate").DependsOn("transform").Then().
        AddStep("load").DependsOn("validate").Then().
        AddStep("notify").DependsOn("load").Then().
        Build()
        
    client.RegisterWorkflow(workflowDef)
}
```

### 用户注册流程

```go
func UserRegistrationFlow() {
    client, _ := sdk.NewClient(sdk.ClientConfig{RedisAddr: "localhost:6379"})
    
    workflowDef := sdk.NewWorkflowBuilder("user_registration").
        AddTask("validate_email", sdk.SimpleTaskHandler(validateEmail), 3).
        AddTask("create_account", sdk.SimpleTaskHandler(createAccount), 3).
        AddTask("send_welcome_email", sdk.SimpleTaskHandler(sendWelcomeEmail), 2).
        AddTask("setup_profile", sdk.SimpleTaskHandler(setupDefaultProfile), 3).
        AddTask("assign_permissions", sdk.SimpleTaskHandler(assignDefaultPermissions), 3).
        
        AddStep("validate_email").Then().
        AddStep("create_account").DependsOn("validate_email").Then().
        AddStep("send_welcome_email").DependsOn("create_account").Then().
        AddStep("setup_profile").DependsOn("create_account").Then().
        AddStep("assign_permissions").DependsOn("setup_profile").Then().
        Build()
        
    client.RegisterWorkflow(workflowDef)
}
```

---

## 🐛 错误代码参考

| 错误代码 | HTTP状态码 | 说明 |
|----------|------------|------|
| WORKFLOW_NOT_FOUND | 404 | 工作流不存在 |
| TASK_NOT_FOUND | 404 | 任务不存在 |
| INVALID_INPUT | 400 | 输入参数无效 |
| WORKFLOW_ALREADY_CANCELED | 400 | 工作流已被取消 |
| WORKFLOW_ALREADY_COMPLETED | 400 | 工作流已完成 |
| INTERNAL_ERROR | 500 | 内部服务器错误 |
| REDIS_CONNECTION_ERROR | 500 | Redis 连接错误 |
| DATABASE_ERROR | 500 | 数据库错误 |

---

## 📞 技术支持

- **文档**: [完整文档](INTEGRATION_GUIDE.md)
- **示例**: [示例代码](examples/)
- **Issues**: [GitHub Issues](https://github.com/XXueTu/temjob/issues)
- **讨论**: [GitHub Discussions](https://github.com/XXueTu/temjob/discussions)