# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

TemJob is a distributed task scheduling and workflow orchestration framework inspired by Temporal. It provides Redis-based task queuing, MySQL persistence, task lifecycle management, and a modern web UI for monitoring workflows.

## Core Architecture

The system follows a layered architecture with these key components:

- **WorkflowEngine** (`pkg/workflow/engine.go`): Orchestrates workflow execution and manages task dependencies
- **Worker** (`pkg/worker/worker.go`): Executes individual tasks, supports distributed deployment
- **TaskQueue** (`pkg/queue/redis_queue.go`): Redis-based task queue with reliable delivery
- **StateManager** (`pkg/state/`): Dual-layer state management (MySQL persistence + Redis caching)
- **SDK** (`pkg/sdk/`): Client library for workflow definition and execution
- **Web Server** (`web/server.go`): REST API and web UI for monitoring

The framework uses a dual-state management approach:
- MySQL for persistent storage of workflows and tasks
- Redis for caching and task queuing

## Common Development Commands

### Build and Run
```bash
# Install dependencies
go mod tidy

# Run with default config
go run main.go

# Run with custom config
go run main.go -config=/path/to/config.yaml

# Build binary
go build -o temjob main.go
```

### Development Environment
```bash
# Start development environment (Go + Docker dependencies)
./scripts/start.sh dev

# Start full Docker environment
./scripts/start.sh docker

# Start production environment
./scripts/start.sh prod

# Stop all services
./scripts/start.sh stop

# Check service status
./scripts/start.sh status
```

### Docker Operations
```bash
# Start all services
docker-compose up -d

# Start only dependencies (Redis + MySQL)
docker-compose up -d redis mysql

# View logs
docker-compose logs -f temjob

# Scale workers
docker-compose up -d --scale temjob-worker=3
```

## Configuration

Configuration is managed through YAML files:
- Copy `config.yaml.example` to `config.yaml`
- Configure MySQL, Redis, server, worker, and logging settings
- Environment variables can be used in Docker deployments

Key configuration sections:
- `database.mysql`: MySQL connection settings
- `redis`: Redis connection and pooling
- `server`: HTTP server port and mode
- `worker`: Concurrency and timeout settings
- `engine`: Workflow monitoring intervals
- `logging`: Log level and format

## Workflow Definition Pattern

Workflows are defined using the SDK builder pattern:

```go
workflowDef := sdk.NewWorkflowBuilder("workflow_name").
    AddTask("task1", handler1, maxRetries).
    AddTask("task2", handler2, maxRetries).
    AddStep("task1").Then().
    AddStep("task2").DependsOn("task1").Then().
    Build()
```

Task handlers implement the `TaskHandler` interface or use `SimpleTaskHandler` wrapper.

## API Endpoints

RESTful API for workflow management:
- `GET /api/v1/workflows` - List workflows with pagination
- `GET /api/v1/workflows/{id}` - Get workflow details
- `POST /api/v1/workflows/{id}/cancel` - Cancel workflow
- `GET /api/v1/workflows/{id}/tasks` - Get workflow tasks
- `GET /api/v1/tasks/{id}` - Get task details
- `GET /api/v1/stats` - Get system statistics

## Web UI

The web interface provides:
- Dashboard with workflow analytics (`/`)
- Workflow list with status filtering (`/workflows`)
- Detailed workflow view with task orchestration diagrams (`/workflows/{id}`)

Web UI features light/dark theme switching with localStorage persistence.

## Task State Management

Tasks progress through these states:
- `pending` → `running` → `completed`/`failed`
- Failed tasks are automatically retried based on `max_retries`
- Workflow orchestration respects task dependencies defined in `DependsOn()`

## Distributed Deployment

The framework supports horizontal scaling:
- Multiple worker nodes can connect to the same Redis/MySQL
- Workers can register different task handlers
- Each worker processes tasks concurrently based on configuration
- Load balancing handled automatically through Redis queues

## Database Schema

Auto-migration is handled by GORM. Key tables:
- `workflows`: Workflow metadata and state
- `tasks`: Individual task data and execution details
- `workflow_executions`: Execution history and results

## Monitoring and Observability

- Structured logging with Zap (configurable JSON/console format)
- Web UI provides real-time status monitoring
- Optional Prometheus/Grafana integration via Docker Compose
- Health check endpoints for container orchestration

## Development Notes

- Use `gin.SetMode()` for environment-specific behavior
- All database operations use GORM for type safety
- Redis operations include connection pooling and retry logic
- Web templates use Bootstrap 5 with custom theming
- JavaScript uses modern ES6+ features with progressive enhancement