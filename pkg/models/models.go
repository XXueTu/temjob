package models

import (
	"time"

	"gorm.io/gorm"
)

type WorkflowModel struct {
	ID        string    `gorm:"type:varchar(36);primary_key" json:"id"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	Input     string    `gorm:"type:json" json:"input"`
	Output    string    `gorm:"type:json" json:"output"`
	State     string    `gorm:"type:varchar(50);not null;index" json:"state"`
	CreatedAt time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`
	StartedAt *time.Time `gorm:"type:datetime;null" json:"started_at"`
	EndedAt   *time.Time `gorm:"type:datetime;null" json:"ended_at"`
	Tasks     []TaskModel `gorm:"foreignKey:WorkflowID" json:"tasks,omitempty"`
}

func (WorkflowModel) TableName() string {
	return "workflows"
}

type TaskModel struct {
	ID          string     `gorm:"type:varchar(36);primary_key" json:"id"`
	WorkflowID  string     `gorm:"type:varchar(36);not null;index" json:"workflow_id"`
	Type        string     `gorm:"type:varchar(255);not null" json:"type"`
	Input       string     `gorm:"type:json" json:"input"`
	Output      string     `gorm:"type:json" json:"output"`
	State       string     `gorm:"type:varchar(50);not null;index" json:"state"`
	Error       string     `gorm:"type:text" json:"error"`
	RetryCount  int        `gorm:"type:int;default:0" json:"retry_count"`
	MaxRetries  int        `gorm:"type:int;default:3" json:"max_retries"`
	CreatedAt   time.Time  `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`
	StartedAt   *time.Time `gorm:"type:datetime;null" json:"started_at"`
	CompletedAt *time.Time `gorm:"type:datetime;null" json:"completed_at"`
	WorkerID    string     `gorm:"type:varchar(255)" json:"worker_id"`
	Workflow    WorkflowModel `gorm:"foreignKey:WorkflowID" json:"workflow,omitempty"`
}

func (TaskModel) TableName() string {
	return "tasks"
}

type WorkflowExecutionLog struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	WorkflowID string    `gorm:"type:varchar(36);not null;index" json:"workflow_id"`
	TaskID     string    `gorm:"type:varchar(36);index" json:"task_id"`
	Level      string    `gorm:"type:varchar(20);not null" json:"level"`
	Message    string    `gorm:"type:text;not null" json:"message"`
	Metadata   string    `gorm:"type:json" json:"metadata"`
	CreatedAt  time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (WorkflowExecutionLog) TableName() string {
	return "workflow_execution_logs"
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&WorkflowModel{},
		&TaskModel{},
		&WorkflowExecutionLog{},
	)
}