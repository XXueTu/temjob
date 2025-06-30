package pkg

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type TaskState string

const (
	TaskStatePending   TaskState = "pending"
	TaskStateRunning   TaskState = "running"
	TaskStateCompleted TaskState = "completed"
	TaskStateFailed    TaskState = "failed"
	TaskStateRetrying  TaskState = "retrying"
	TaskStateCanceled  TaskState = "canceled"
)

type WorkflowState string

const (
	WorkflowStatePending   WorkflowState = "pending"
	WorkflowStateRunning   WorkflowState = "running"
	WorkflowStateCompleted WorkflowState = "completed"
	WorkflowStateFailed    WorkflowState = "failed"
	WorkflowStateCanceled  WorkflowState = "canceled"
)

type Task struct {
	ID          string                 `json:"id"`
	WorkflowID  string                 `json:"workflow_id"`
	Type        string                 `json:"type"`
	Input       map[string]interface{} `json:"input"`
	Output      map[string]interface{} `json:"output,omitempty"`
	State       TaskState              `json:"state"`
	Error       string                 `json:"error,omitempty"`
	RetryCount  int                    `json:"retry_count"`
	MaxRetries  int                    `json:"max_retries"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	WorkerID    string                 `json:"worker_id,omitempty"`
}

type Workflow struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Input     map[string]interface{} `json:"input"`
	Output    map[string]interface{} `json:"output,omitempty"`
	State     WorkflowState          `json:"state"`
	Tasks     []string               `json:"tasks"`
	CreatedAt time.Time              `json:"created_at"`
	StartedAt *time.Time             `json:"started_at,omitempty"`
	EndedAt   *time.Time             `json:"ended_at,omitempty"`
}

type TaskHandler func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)

type WorkflowDefinition struct {
	Name  string
	Tasks map[string]TaskDefinition
	Flow  []WorkflowStep
}

type TaskDefinition struct {
	Type       string
	Handler    TaskHandler
	MaxRetries int
}

type WorkflowStep struct {
	TaskType     string
	DependsOn    []string
	Condition    func(map[string]interface{}) bool
	OnError      string
}

type Worker interface {
	Start(ctx context.Context) error
	Stop() error
	RegisterTaskHandler(taskType string, handler TaskHandler)
	GetID() string
}

type WorkflowEngine interface {
	RegisterWorkflow(definition WorkflowDefinition)
	SubmitWorkflow(ctx context.Context, workflowName string, input map[string]interface{}) (string, error)
	GetWorkflow(ctx context.Context, workflowID string) (*Workflow, error)
	CancelWorkflow(ctx context.Context, workflowID string) error
	Start(ctx context.Context) error
	Stop() error
}

type TaskQueue interface {
	Enqueue(ctx context.Context, task *Task) error
	Dequeue(ctx context.Context, workerID string) (*Task, error)
	UpdateTaskState(ctx context.Context, taskID string, state TaskState, output map[string]interface{}, err string) error
}

type StateManager interface {
	SaveWorkflow(ctx context.Context, workflow *Workflow) error
	GetWorkflow(ctx context.Context, workflowID string) (*Workflow, error)
	SaveTask(ctx context.Context, task *Task) error
	GetTask(ctx context.Context, taskID string) (*Task, error)
	GetWorkflowTasks(ctx context.Context, workflowID string) ([]*Task, error)
	ListWorkflows(ctx context.Context, limit, offset int) ([]*Workflow, error)
}

type CacheInvalidator interface {
	InvalidateCache(ctx context.Context, workflowID string) error
	InvalidateTaskCache(ctx context.Context, taskID string) error
}

func NewTaskID() string {
	return uuid.New().String()
}

func NewWorkflowID() string {
	return uuid.New().String()
}