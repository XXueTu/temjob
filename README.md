# TemJob - 分布式任务调度框架

TemJob 是一个基于 Redis 和 MySQL 的分布式任务调度框架，参考了 Temporal 的设计理念，提供工作流编排、任务生命周期管理和 Web UI 管理界面。

## 特性

- 🚀 **分布式任务调度**: 基于 Redis 的任务队列，支持多 Worker 节点
- 🔄 **工作流编排**: 支持复杂的任务依赖关系和条件执行
- 💾 **数据持久化**: MySQL 存储任务数据，Redis 作为缓存层
- 🔁 **任务重试机制**: 自动重试失败的任务
- 📊 **Web 管理界面**: 实时监控任务执行状态
- 🛠 **SDK 支持**: 简单易用的 SDK 用于定义和执行工作流

## 架构设计

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web UI        │    │   Worker Node   │    │   Worker Node   │
│   (Port 8080)   │    │                 │    │                 │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          │              ┌───────▼───────┐              │
          │              │  Task Queue   │              │
          │              │   (Redis)     │◄─────────────┘
          │              └───────┬───────┘
          │                      │
    ┌─────▼─────┐       ┌────────▼────────┐
    │  Engine   │       │  State Manager  │
    │           │       │                 │
    └───────────┘       └─────────────────┘
                                 │
                        ┌────────▼────────┐
                        │     MySQL       │
                        │  (Persistence)  │
                        └─────────────────┘
```

## 快速开始

### 环境要求

- Go 1.19+
- Redis
- MySQL

### 安装依赖

```bash
go mod tidy
```

### 配置数据库

1. 创建 MySQL 数据库:
```sql
CREATE DATABASE temjob CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

2. 确保 Redis 服务运行在 `localhost:6379`

### 配置文件

修改 `config.yaml` 文件配置数据库连接信息：

```yaml
database:
  mysql:
    host: 10.99.51.9
    port: 3306
    user: root
    password: Root@123
    database: wk
    charset: utf8mb4
    parse_time: true
    loc: Local

redis:
  host: 10.99.51.6
  port: 6379
  password: "redis123"
  db: 2
  pool_size: 10

server:
  port: 8088
  mode: debug  # debug, release
```

### 运行框架

```bash
# 使用默认配置文件 config.yaml
go run main.go

# 或指定配置文件路径
go run main.go -config=/path/to/your/config.yaml
```

访问 Web UI: http://localhost:8088 (根据配置文件中的端口)

## 使用示例

### 定义工作流

```go
package main

import (
    "context"
    "log"
    "temjob/pkg/sdk"
)

func main() {
    // 创建客户端
    client, err := sdk.NewClient(sdk.ClientConfig{
        RedisAddr: "localhost:6379",
        RedisDB:   0,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 注册任务处理函数
    client.RegisterTaskHandler("send_email", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
        recipient := input["recipient"].(string)
        // 发送邮件逻辑
        return map[string]interface{}{
            "email_sent": true,
        }, nil
    }))

    // 定义工作流
    workflowDef := sdk.NewWorkflowBuilder("notification").
        AddTask("send_email", handler, 3).
        AddStep("send_email").
        Build()

    // 注册工作流
    client.RegisterWorkflow(workflowDef)

    // 启动服务
    ctx := context.Background()
    go client.StartEngine(ctx)
    go client.StartWorker(ctx)

    // 提交工作流
    workflowID, err := client.SubmitWorkflow(ctx, "notification", map[string]interface{}{
        "recipient": "user@example.com",
    })
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("工作流已提交: %s", workflowID)
}
```

### 复杂工作流示例

```go
// 数据处理工作流
workflowDef := sdk.NewWorkflowBuilder("data_processing").
    AddTask("validate_input", validateHandler, 3).
    AddTask("process_data", processHandler, 5).
    AddTask("generate_report", reportHandler, 3).
    AddStep("validate_input").Then().
    AddStep("process_data").DependsOn("validate_input").Then().
    AddStep("generate_report").DependsOn("process_data").Then().
    Build()
```

## API 接口

### REST API

- `GET /api/v1/workflows` - 获取工作流列表
- `GET /api/v1/workflows/{id}` - 获取工作流详情
- `POST /api/v1/workflows/{id}/cancel` - 取消工作流
- `GET /api/v1/workflows/{id}/tasks` - 获取工作流任务列表
- `GET /api/v1/tasks/{id}` - 获取任务详情
- `GET /api/v1/stats` - 获取统计信息

### Web UI 页面

- `/` - 仪表板
- `/workflows` - 工作流列表
- `/workflows/{id}` - 工作流详情

## 核心组件

### WorkflowEngine
负责工作流的编排和执行，管理任务的依赖关系。

### Worker
执行具体的任务，支持分布式部署多个 Worker 节点。

### TaskQueue
基于 Redis 的任务队列，支持任务的可靠投递和处理。

### StateManager
管理工作流和任务的状态，支持 MySQL 持久化和 Redis 缓存。

## 配置选项

### 数据库配置

```go
// MySQL DSN
dsn := "user:password@tcp(localhost:3306)/temjob?charset=utf8mb4&parseTime=True&loc=Local"

// Redis 配置
redisClient := redis.NewClient(&redis.Options{
    Addr:     "localhost:6379",
    Password: "",
    DB:       0,
})
```

### 任务重试配置

```go
// 任务定义时设置最大重试次数
AddTask("task_name", handler, 5) // 最大重试 5 次
```

## 监控和日志

框架使用 Zap 进行结构化日志记录，所有关键操作都有详细的日志输出。

Web UI 提供实时的任务执行状态监控，包括：
- 工作流状态统计
- 任务执行进度
- 错误信息展示

## 部署

### 单机部署
直接运行 `go run main.go` 即可启动完整的服务。

### 分布式部署
- 部署多个 Worker 节点处理任务
- 共享 Redis 和 MySQL 实例
- 每个节点可以注册不同的任务处理器

## 贡献

欢迎提交 Issue 和 Pull Request 来改进这个项目。

## 许可证

MIT License