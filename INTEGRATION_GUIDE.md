# TemJob 分布式任务调度框架

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Redis](https://img.shields.io/badge/Redis-Required-red.svg)](https://redis.io)
[![MySQL](https://img.shields.io/badge/MySQL-Required-blue.svg)](https://mysql.com)

## 🚀 简介

TemJob 是一个基于 Redis 的高性能分布式任务调度框架，参考 Temporal 的设计理念，提供了完整的工作流编排能力。支持任务依赖、重试、监控等企业级特性，并配备现代化的 Web 管理界面。

### ✨ 核心特性

- 🔄 **工作流编排**: 支持复杂的任务依赖关系和工作流定义
- 📊 **实时监控**: 现代化 Web UI，实时展示任务执行状态
- 🚀 **高性能**: 基于 Redis 队列，支持分布式部署
- 🛡️ **可靠性**: MySQL 持久化存储，Redis 缓存加速
- 🔧 **易扩展**: 简洁的 SDK 设计，快速集成业务逻辑
- 📈 **可观测**: 完整的日志记录和执行链路追踪

## 📋 系统要求

- **Go**: 1.23+
- **Redis**: 5.0+
- **MySQL**: 8.0+
- **内存**: 推荐 2GB+
- **CPU**: 推荐 2 核+

## 🛠️ 快速开始

### 1. 环境准备

确保 Redis 和 MySQL 服务已启动：

```bash
# Redis (默认端口 6379)
redis-server

# MySQL (默认端口 3306)
mysql.server start
```

### 2. 获取代码

```bash
git clone https://github.com/XXueTu/temjob.git
cd temjob
```

### 3. 配置文件

创建 `config.yaml` 配置文件：

```yaml
database:
  mysql:
    host: localhost
    port: 3306
    user: root
    password: your_password
    database: temjob
    charset: utf8mb4
    parse_time: true
    loc: Local

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0
  pool_size: 10

server:
  port: 8088
  mode: debug  # debug, release

worker:
  concurrency: 10
  timeout: 30m
  retry_delay: 5s

engine:
  monitor_interval: 10s
  max_workflow_timeout: 24h

logging:
  level: info
  format: json
  output: stdout
```

### 4. 安装依赖

```bash
go mod tidy
```

### 5. 启动服务

```bash
go run main.go --config config.yaml
```

服务启动后访问：http://localhost:8088

## 📚 集成指南

### 1. 添加依赖

在您的项目中添加 TemJob 依赖：

```bash
go mod init your-project
go get github.com/XXueTu/temjob
```

### 2. 基础集成示例

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/XXueTu/temjob/pkg/sdk"
)

func main() {
    // 创建客户端
    client, err := sdk.NewClient(sdk.ClientConfig{
        RedisAddr:     "localhost:6379",
        RedisPassword: "",
        RedisDB:       0,
    })
    if err != nil {
        log.Fatal("Failed to create client:", err)
    }
    defer client.Close()

    // 注册任务处理器
    client.RegisterTaskHandler("send_email", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
        recipient := input["recipient"].(string)
        subject := input["subject"].(string)
        
        // 实现您的业务逻辑
        log.Printf("Sending email to %s: %s", recipient, subject)
        
        return map[string]interface{}{
            "email_sent": true,
            "sent_at":    time.Now().Format(time.RFC3339),
        }, nil
    }))

    // 定义工作流
    workflowDef := sdk.NewWorkflowBuilder("notification_workflow").
        AddTask("send_email", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
            return input, nil
        }), 3).
        AddStep("send_email").Then().
        Build()

    // 注册工作流
    client.RegisterWorkflow(workflowDef)

    // 启动引擎和工作器
    ctx := context.Background()
    go client.StartEngine(ctx)
    go client.StartWorker(ctx)

    // 提交工作流
    workflowID, err := client.SubmitWorkflow(ctx, "notification_workflow", map[string]interface{}{
        "recipient": "user@example.com",
        "subject":   "Welcome!",
    })
    if err != nil {
        log.Fatal("Failed to submit workflow:", err)
    }

    log.Printf("Workflow submitted: %s", workflowID)
}
```

### 3. 复杂工作流示例

```go
// 数据处理工作流示例
func CreateDataProcessingWorkflow() pkg.WorkflowDefinition {
    return sdk.NewWorkflowBuilder("data_processing").
        // 定义任务
        AddTask("validate_input", sdk.SimpleTaskHandler(validateInput), 3).
        AddTask("process_data", sdk.SimpleTaskHandler(processData), 3).
        AddTask("generate_report", sdk.SimpleTaskHandler(generateReport), 3).
        AddTask("send_notification", sdk.SimpleTaskHandler(sendNotification), 2).
        
        // 定义执行流程和依赖关系
        AddStep("validate_input").Then().
        AddStep("process_data").DependsOn("validate_input").Then().
        AddStep("generate_report").DependsOn("process_data").Then().
        AddStep("send_notification").DependsOn("generate_report").Then().
        Build()
}

func validateInput(input map[string]interface{}) (map[string]interface{}, error) {
    // 验证输入数据
    dataFile := input["data_file"].(string)
    if dataFile == "" {
        return nil, fmt.Errorf("data_file is required")
    }
    
    return map[string]interface{}{
        "validated": true,
        "file_size": 1024000,
        "data_file": dataFile,
    }, nil
}

func processData(input map[string]interface{}) (map[string]interface{}, error) {
    // 处理数据
    dataFile := input["data_file"].(string)
    
    // 模拟数据处理
    time.Sleep(5 * time.Second)
    
    return map[string]interface{}{
        "processed_records": 10000,
        "output_file":       "processed_" + dataFile,
        "processing_time":   "5.2s",
    }, nil
}
```

## 🏗️ 架构设计

### 核心组件

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web UI        │    │   Workflow      │    │   Task Queue    │
│   (管理界面)     │    │   Engine        │    │   (Redis)       │
│                 │    │   (编排引擎)     │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Workers       │    │   State         │    │   SDK           │
│   (任务执行器)   │    │   Manager       │    │   (客户端)       │
│                 │    │   (MySQL+Redis) │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### 数据流

1. **工作流提交**: 通过 SDK 提交工作流定义和输入数据
2. **任务调度**: Workflow Engine 根据依赖关系将任务加入 Redis 队列
3. **任务执行**: Workers 从队列中获取任务并执行
4. **状态管理**: 执行状态保存到 MySQL，并在 Redis 中缓存
5. **结果聚合**: Engine 收集任务结果，更新工作流状态

## 🔌 API 接口

### RESTful API

#### 工作流管理

```bash
# 获取工作流列表
GET /api/v1/workflows?limit=20&offset=0

# 获取工作流详情
GET /api/v1/workflows/{workflow_id}

# 获取工作流任务列表
GET /api/v1/workflows/{workflow_id}/tasks

# 取消工作流
POST /api/v1/workflows/{workflow_id}/cancel
```

#### 任务管理

```bash
# 获取任务详情
GET /api/v1/tasks/{task_id}

# 获取统计信息
GET /api/v1/stats
```

### SDK 接口

```go
// 客户端配置
type ClientConfig struct {
    RedisAddr     string
    RedisPassword string
    RedisDB       int
}

// 主要接口
type Client interface {
    // 注册任务处理器
    RegisterTaskHandler(taskType string, handler TaskHandler)
    
    // 注册工作流
    RegisterWorkflow(definition WorkflowDefinition)
    
    // 提交工作流
    SubmitWorkflow(ctx context.Context, workflowName string, input map[string]interface{}) (string, error)
    
    // 获取工作流状态
    GetWorkflow(ctx context.Context, workflowID string) (*Workflow, error)
    
    // 启动引擎和工作器
    StartEngine(ctx context.Context) error
    StartWorker(ctx context.Context) error
}
```

## 🔧 高级配置

### 性能调优

```yaml
# 工作器并发配置
worker:
  concurrency: 20        # 并发数
  timeout: 1h           # 任务超时
  retry_delay: 10s      # 重试间隔

# 引擎配置
engine:
  monitor_interval: 5s          # 监控间隔
  max_workflow_timeout: 48h     # 工作流最大执行时间

# Redis 连接池
redis:
  pool_size: 20         # 连接池大小
  max_retries: 3        # 最大重试次数
  min_idle_conns: 5     # 最小空闲连接
```

### 监控和日志

```yaml
logging:
  level: info           # debug, info, warn, error
  format: json          # json, console
  output: stdout        # stdout, stderr, file
  file_path: logs/temjob.log
  max_size: 100         # MB
  max_backups: 7
  max_age: 30           # days
```

## 🚀 部署指南

### Docker 部署

```dockerfile
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod tidy && go build -o temjob main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/temjob .
COPY --from=builder /app/config.yaml .

EXPOSE 8088
CMD ["./temjob", "--config", "config.yaml"]
```

### Docker Compose

```yaml
version: '3.8'

services:
  temjob:
    build: .
    ports:
      - "8088:8088"
    environment:
      - CONFIG_PATH=/app/config.yaml
    depends_on:
      - redis
      - mysql
    volumes:
      - ./config.yaml:/app/config.yaml

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    command: redis-server --appendonly yes

  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: root123
      MYSQL_DATABASE: temjob
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql

volumes:
  mysql_data:
```

### Kubernetes 部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: temjob
spec:
  replicas: 3
  selector:
    matchLabels:
      app: temjob
  template:
    metadata:
      labels:
        app: temjob
    spec:
      containers:
      - name: temjob
        image: temjob:latest
        ports:
        - containerPort: 8088
        env:
        - name: CONFIG_PATH
          value: "/app/config.yaml"
        volumeMounts:
        - name: config
          mountPath: /app/config.yaml
          subPath: config.yaml
      volumes:
      - name: config
        configMap:
          name: temjob-config
---
apiVersion: v1
kind: Service
metadata:
  name: temjob-service
spec:
  selector:
    app: temjob
  ports:
  - port: 8088
    targetPort: 8088
  type: LoadBalancer
```

## 📖 最佳实践

### 1. 任务设计原则

- **幂等性**: 任务应该支持重复执行而不产生副作用
- **原子性**: 每个任务应该是一个原子操作
- **错误处理**: 合理使用重试机制，记录详细错误信息

```go
func idempotentTask(input map[string]interface{}) (map[string]interface{}, error) {
    taskID := input["task_id"].(string)
    
    // 检查是否已执行
    if isAlreadyProcessed(taskID) {
        return getExistingResult(taskID), nil
    }
    
    // 执行业务逻辑
    result, err := doBusinessLogic(input)
    if err != nil {
        return nil, fmt.Errorf("business logic failed: %w", err)
    }
    
    // 保存结果
    saveResult(taskID, result)
    return result, nil
}
```

### 2. 工作流设计

- **模块化**: 将复杂流程拆分为独立的任务单元
- **依赖关系**: 合理设计任务依赖，避免循环依赖
- **错误恢复**: 设计错误处理和恢复策略

```go
// 良好的工作流设计示例
func CreateOrderProcessingWorkflow() pkg.WorkflowDefinition {
    return sdk.NewWorkflowBuilder("order_processing").
        // 并行执行的验证任务
        AddTask("validate_inventory", validateInventory, 3).
        AddTask("validate_payment", validatePayment, 3).
        
        // 依赖验证结果的处理任务
        AddTask("reserve_inventory", reserveInventory, 3).
        AddTask("charge_payment", chargePayment, 3).
        
        // 最终确认
        AddTask("confirm_order", confirmOrder, 2).
        
        // 定义执行流程
        AddStep("validate_inventory").Then().
        AddStep("validate_payment").Then().
        AddStep("reserve_inventory").DependsOn("validate_inventory").Then().
        AddStep("charge_payment").DependsOn("validate_payment").Then().
        AddStep("confirm_order").DependsOn("reserve_inventory", "charge_payment").Then().
        Build()
}
```

### 3. 监控和告警

```go
// 添加监控指标
func monitoredTaskHandler(next TaskHandler) TaskHandler {
    return sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
        start := time.Now()
        defer func() {
            duration := time.Since(start)
            // 记录执行时间
            metrics.RecordTaskDuration(duration)
        }()
        
        result, err := next(input)
        if err != nil {
            // 记录错误
            metrics.IncrementTaskErrors()
        }
        
        return result, err
    })
}
```

## 🐛 故障排查

### 常见问题

1. **任务卡住不执行**
   - 检查 Redis 连接
   - 确认 Worker 是否正常启动
   - 查看任务是否有未满足的依赖

2. **任务执行失败**
   - 查看任务错误日志
   - 检查重试配置
   - 验证任务处理器实现

3. **Web UI 无法访问**
   - 确认服务端口配置
   - 检查防火墙设置
   - 查看服务启动日志

### 日志分析

```bash
# 查看实时日志
tail -f logs/temjob.log | jq '.'

# 过滤错误日志
grep "ERROR" logs/temjob.log | jq '.'

# 查看特定工作流日志
grep "workflow_id=xxx" logs/temjob.log
```

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🙏 致谢

- 感谢 [Temporal](https://temporal.io/) 项目的设计启发
- 感谢 Go 社区的优秀开源项目

## 📞 联系方式

- **Issues**: [GitHub Issues](https://github.com/XXueTu/temjob/issues)
- **Discussions**: [GitHub Discussions](https://github.com/XXueTu/temjob/discussions)
- **Email**: support@temjob.dev

---

**⭐ 如果这个项目对您有帮助，请给我们一个 Star！**