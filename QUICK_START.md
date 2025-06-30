# TemJob 快速入门指南

本指南将帮助您在 5 分钟内快速集成 TemJob 到您的项目中。

## 🚀 5分钟快速集成

### 第一步：环境准备

确保您的系统已安装：
- Go 1.23+
- Redis
- MySQL

### 第二步：创建新项目

```bash
mkdir my-temjob-app
cd my-temjob-app
go mod init my-temjob-app
```

### 第三步：安装 TemJob

```bash
go get github.com/XXueTu/temjob
```

### 第四步：编写第一个工作流

创建 `main.go` 文件：

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "github.com/XXueTu/temjob/pkg/sdk"
)

func main() {
    // 1. 创建客户端
    client, err := sdk.NewClient(sdk.ClientConfig{
        RedisAddr:     "localhost:6379",
        RedisPassword: "",
        RedisDB:       0,
    })
    if err != nil {
        log.Fatal("Failed to create client:", err)
    }
    defer client.Close()

    // 2. 注册任务处理器
    client.RegisterTaskHandler("greet_user", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
        name := input["name"].(string)
        message := fmt.Sprintf("Hello, %s! Welcome to TemJob!", name)
        
        fmt.Println(message)
        
        return map[string]interface{}{
            "greeting": message,
            "timestamp": time.Now().Format(time.RFC3339),
        }, nil
    }))

    client.RegisterTaskHandler("send_welcome_email", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
        greeting := input["greeting"].(string)
        
        // 模拟发送邮件
        fmt.Printf("📧 Sending email: %s\n", greeting)
        time.Sleep(1 * time.Second)
        
        return map[string]interface{}{
            "email_sent": true,
            "sent_at": time.Now().Format(time.RFC3339),
        }, nil
    }))

    // 3. 定义工作流
    workflowDef := sdk.NewWorkflowBuilder("welcome_workflow").
        AddTask("greet_user", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
            return input, nil
        }), 3).
        AddTask("send_welcome_email", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
            return input, nil
        }), 3).
        AddStep("greet_user").Then().
        AddStep("send_welcome_email").DependsOn("greet_user").Then().
        Build()

    // 4. 注册工作流
    client.RegisterWorkflow(workflowDef)

    // 5. 启动引擎和工作器
    ctx := context.Background()
    go client.StartEngine(ctx)
    go client.StartWorker(ctx)

    // 等待服务启动
    time.Sleep(2 * time.Second)

    // 6. 提交工作流
    workflowID, err := client.SubmitWorkflow(ctx, "welcome_workflow", map[string]interface{}{
        "name": "Alice",
    })
    if err != nil {
        log.Fatal("Failed to submit workflow:", err)
    }

    fmt.Printf("✅ Workflow submitted: %s\n", workflowID)

    // 7. 监控工作流执行
    for {
        workflow, err := client.GetWorkflow(ctx, workflowID)
        if err != nil {
            log.Printf("Failed to get workflow: %v", err)
            break
        }

        fmt.Printf("📊 Workflow %s state: %s\n", workflowID, workflow.State)

        if workflow.State == "completed" {
            fmt.Printf("🎉 Workflow completed successfully!\n")
            if workflow.Output != nil {
                fmt.Printf("📤 Output: %+v\n", workflow.Output)
            }
            break
        } else if workflow.State == "failed" {
            fmt.Printf("❌ Workflow failed!\n")
            break
        }

        time.Sleep(1 * time.Second)
    }
}
```

### 第五步：运行

```bash
go run main.go
```

期望输出：
```
✅ Workflow submitted: 12345-abcde-67890
Hello, Alice! Welcome to TemJob!
📧 Sending email: Hello, Alice! Welcome to TemJob!
📊 Workflow 12345-abcde-67890 state: running
📊 Workflow 12345-abcde-67890 state: completed
🎉 Workflow completed successfully!
📤 Output: map[email_sent:true greeting:Hello, Alice! Welcome to TemJob! ...]
```

## 🏗️ 实际业务场景示例

### 电商订单处理工作流

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "github.com/XXueTu/temjob/pkg/sdk"
)

// 订单处理工作流
func CreateOrderWorkflow() {
    client, _ := sdk.NewClient(sdk.ClientConfig{
        RedisAddr: "localhost:6379",
    })
    defer client.Close()

    // 验证库存
    client.RegisterTaskHandler("validate_inventory", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
        productID := input["product_id"].(string)
        quantity := int(input["quantity"].(float64))
        
        // 模拟库存检查
        if quantity > 100 {
            return nil, fmt.Errorf("insufficient inventory for product %s", productID)
        }
        
        return map[string]interface{}{
            "inventory_valid": true,
            "reserved_qty": quantity,
        }, nil
    }))

    // 处理支付
    client.RegisterTaskHandler("process_payment", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
        amount := input["amount"].(float64)
        
        // 模拟支付处理
        time.Sleep(2 * time.Second)
        
        return map[string]interface{}{
            "payment_processed": true,
            "transaction_id": fmt.Sprintf("txn_%d", time.Now().Unix()),
            "charged_amount": amount,
        }, nil
    }))

    // 创建订单
    client.RegisterTaskHandler("create_order", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
        transactionID := input["transaction_id"].(string)
        
        orderID := fmt.Sprintf("order_%d", time.Now().Unix())
        
        return map[string]interface{}{
            "order_created": true,
            "order_id": orderID,
            "status": "confirmed",
        }, nil
    }))

    // 发送确认邮件
    client.RegisterTaskHandler("send_confirmation", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
        orderID := input["order_id"].(string)
        
        fmt.Printf("📧 Sending order confirmation for %s\n", orderID)
        
        return map[string]interface{}{
            "confirmation_sent": true,
            "email_status": "delivered",
        }, nil
    }))

    // 定义工作流
    workflowDef := sdk.NewWorkflowBuilder("order_processing").
        AddTask("validate_inventory", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
            return input, nil
        }), 3).
        AddTask("process_payment", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
            return input, nil
        }), 3).
        AddTask("create_order", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
            return input, nil
        }), 3).
        AddTask("send_confirmation", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
            return input, nil
        }), 2).
        AddStep("validate_inventory").Then().
        AddStep("process_payment").DependsOn("validate_inventory").Then().
        AddStep("create_order").DependsOn("process_payment").Then().
        AddStep("send_confirmation").DependsOn("create_order").Then().
        Build()

    client.RegisterWorkflow(workflowDef)

    ctx := context.Background()
    go client.StartEngine(ctx)
    go client.StartWorker(ctx)

    time.Sleep(2 * time.Second)

    // 提交订单处理请求
    workflowID, err := client.SubmitWorkflow(ctx, "order_processing", map[string]interface{}{
        "product_id": "PHONE_123",
        "quantity":   2,
        "amount":     1999.99,
        "customer_email": "customer@example.com",
    })
    
    if err != nil {
        log.Fatal("Failed to submit workflow:", err)
    }

    fmt.Printf("🛒 Order processing started: %s\n", workflowID)
    
    // 监控处理进度
    for {
        workflow, err := client.GetWorkflow(ctx, workflowID)
        if err != nil {
            break
        }

        fmt.Printf("📊 Order %s status: %s\n", workflowID, workflow.State)

        if workflow.State == "completed" || workflow.State == "failed" {
            break
        }

        time.Sleep(2 * time.Second)
    }
}
```

## 🔧 与现有项目集成

### 方式一：独立服务模式

在现有项目中启动 TemJob 作为独立的微服务：

```go
// temjob-service/main.go
package main

import (
    "github.com/XXueTu/temjob/pkg/config"
    "github.com/XXueTu/temjob/pkg/sdk"
)

func main() {
    // 启动 TemJob 服务
    cfg, _ := config.LoadConfig("config.yaml")
    
    client, _ := sdk.NewClient(sdk.ClientConfig{
        RedisAddr: cfg.Redis.Addr(),
        RedisPassword: cfg.Redis.Password,
        RedisDB: cfg.Redis.DB,
    })
    
    // 注册您的业务工作流
    registerBusinessWorkflows(client)
    
    // 启动服务
    ctx := context.Background()
    go client.StartEngine(ctx)
    go client.StartWorker(ctx)
    
    // 启动 HTTP API 服务
    startHTTPServer(client)
}

func registerBusinessWorkflows(client *sdk.Client) {
    // 注册您的任务处理器和工作流
}
```

### 方式二：嵌入式集成

将 TemJob 直接嵌入到现有应用中：

```go
// your-app/services/workflow_service.go
package services

import (
    "context"
    "github.com/XXueTu/temjob/pkg/sdk"
)

type WorkflowService struct {
    client *sdk.Client
}

func NewWorkflowService() *WorkflowService {
    client, _ := sdk.NewClient(sdk.ClientConfig{
        RedisAddr: "localhost:6379",
    })
    
    ws := &WorkflowService{client: client}
    ws.registerWorkflows()
    
    return ws
}

func (ws *WorkflowService) Start(ctx context.Context) {
    go ws.client.StartEngine(ctx)
    go ws.client.StartWorker(ctx)
}

func (ws *WorkflowService) SubmitUserRegistration(userID string, email string) (string, error) {
    return ws.client.SubmitWorkflow(context.Background(), "user_registration", map[string]interface{}{
        "user_id": userID,
        "email": email,
    })
}

func (ws *WorkflowService) registerWorkflows() {
    // 注册用户注册工作流
    ws.client.RegisterTaskHandler("send_welcome_email", ...)
    ws.client.RegisterTaskHandler("create_user_profile", ...)
    // ... 更多任务处理器
}
```

然后在主应用中使用：

```go
// your-app/main.go
func main() {
    // 其他初始化代码...
    
    workflowService := services.NewWorkflowService()
    workflowService.Start(context.Background())
    
    // 在业务逻辑中使用
    workflowID, err := workflowService.SubmitUserRegistration("user123", "user@example.com")
    if err != nil {
        log.Printf("Failed to start user registration workflow: %v", err)
    }
}
```

## 🌐 HTTP API 集成

如果您的项目使用不同的编程语言，可以通过 HTTP API 集成：

### 启动 TemJob 服务

```bash
go run main.go --config config.yaml
```

### 通过 HTTP API 提交工作流

```bash
# 提交工作流
curl -X POST http://localhost:8088/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "data_processing",
    "input": {
      "input_file": "data.csv",
      "output_dir": "/tmp/output"
    }
  }'

# 查询工作流状态
curl http://localhost:8088/api/v1/workflows/{workflow_id}

# 获取工作流任务列表
curl http://localhost:8088/api/v1/workflows/{workflow_id}/tasks
```

### Python 客户端示例

```python
import requests
import time

class TemJobClient:
    def __init__(self, base_url="http://localhost:8088"):
        self.base_url = base_url
    
    def submit_workflow(self, name, input_data):
        response = requests.post(f"{self.base_url}/api/v1/workflows", json={
            "name": name,
            "input": input_data
        })
        return response.json()["workflow_id"]
    
    def get_workflow(self, workflow_id):
        response = requests.get(f"{self.base_url}/api/v1/workflows/{workflow_id}")
        return response.json()
    
    def wait_for_completion(self, workflow_id):
        while True:
            workflow = self.get_workflow(workflow_id)
            if workflow["state"] in ["completed", "failed"]:
                return workflow
            time.sleep(1)

# 使用示例
client = TemJobClient()
workflow_id = client.submit_workflow("data_processing", {
    "input_file": "data.csv",
    "output_dir": "/tmp/output"
})

result = client.wait_for_completion(workflow_id)
print(f"Workflow completed with state: {result['state']}")
```

## 📊 监控和调试

### 使用 Web UI

访问 http://localhost:8088 查看：
- 工作流执行状态
- 任务执行详情
- 实时监控面板
- 执行历史记录

### 日志配置

在 `config.yaml` 中配置详细日志：

```yaml
logging:
  level: debug
  format: json
  output: stdout
```

### 性能监控

```go
// 添加执行时间监控
client.RegisterTaskHandler("monitored_task", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        log.Printf("Task executed in %v", duration)
    }()
    
    // 您的业务逻辑
    return map[string]interface{}{
        "result": "success",
    }, nil
}))
```

## 🚀 下一步

恭喜！您已经成功集成了 TemJob。接下来您可以：

1. 📖 阅读完整的 [集成指南](INTEGRATION_GUIDE.md)
2. 🏗️ 了解 [架构设计](docs/ARCHITECTURE.md)
3. 🔧 配置 [高级特性](docs/ADVANCED.md)
4. 🐛 查看 [故障排查](docs/TROUBLESHOOTING.md)
5. 💻 浏览更多 [示例代码](examples/)

有问题？查看我们的 [FAQ](docs/FAQ.md) 或提交 [Issue](https://github.com/XXueTu/temjob/issues)！