package state

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/XXueTu/temjob/pkg"
	"github.com/XXueTu/temjob/pkg/models"
)

type MySQLStateManager struct {
	db       *gorm.DB
	redis    *redis.Client
	logger   *zap.Logger
	cacheTTL time.Duration
}

func NewMySQLStateManager(db *gorm.DB, redis *redis.Client, logger *zap.Logger) *MySQLStateManager {
	return &MySQLStateManager{
		db:       db,
		redis:    redis,
		logger:   logger,
		cacheTTL: 5 * time.Minute,
	}
}

func (s *MySQLStateManager) SaveWorkflow(ctx context.Context, workflow *pkg.Workflow) error {
	inputJSON, _ := json.Marshal(workflow.Input)
	outputJSON, _ := json.Marshal(workflow.Output)

	workflowModel := &models.WorkflowModel{
		ID:        workflow.ID,
		Name:      workflow.Name,
		Input:     string(inputJSON),
		Output:    string(outputJSON),
		State:     string(workflow.State),
		CreatedAt: workflow.CreatedAt,
		StartedAt: workflow.StartedAt,
		EndedAt:   workflow.EndedAt,
	}

	err := s.db.WithContext(ctx).Save(workflowModel).Error
	if err != nil {
		return fmt.Errorf("failed to save workflow to MySQL: %w", err)
	}

	cacheKey := "workflow:" + workflow.ID
	workflowJSON, _ := json.Marshal(workflow)
	s.redis.Set(ctx, cacheKey, workflowJSON, s.cacheTTL)

	s.logger.Info("Workflow saved", zap.String("workflow_id", workflow.ID))
	return nil
}

func (s *MySQLStateManager) GetWorkflow(ctx context.Context, workflowID string) (*pkg.Workflow, error) {
	cacheKey := "workflow:" + workflowID

	if cached, err := s.redis.Get(ctx, cacheKey).Result(); err == nil {
		var workflow pkg.Workflow
		if json.Unmarshal([]byte(cached), &workflow) == nil {
			s.logger.Debug("Workflow retrieved from cache", zap.String("workflow_id", workflowID))
			return &workflow, nil
		}
	}

	var workflowModel models.WorkflowModel
	err := s.db.WithContext(ctx).First(&workflowModel, "id = ?", workflowID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("workflow not found: %s", workflowID)
		}
		return nil, fmt.Errorf("failed to get workflow from MySQL: %w", err)
	}

	workflow := s.modelToWorkflow(&workflowModel)

	workflowJSON, _ := json.Marshal(workflow)
	s.redis.Set(ctx, cacheKey, workflowJSON, s.cacheTTL)

	return workflow, nil
}

func (s *MySQLStateManager) SaveTask(ctx context.Context, task *pkg.Task) error {
	inputJSON, _ := json.Marshal(task.Input)
	outputJSON, _ := json.Marshal(task.Output)

	taskModel := &models.TaskModel{
		ID:          task.ID,
		WorkflowID:  task.WorkflowID,
		Type:        task.Type,
		Input:       string(inputJSON),
		Output:      string(outputJSON),
		State:       string(task.State),
		Error:       task.Error,
		RetryCount:  task.RetryCount,
		MaxRetries:  task.MaxRetries,
		CreatedAt:   task.CreatedAt,
		StartedAt:   task.StartedAt,
		CompletedAt: task.CompletedAt,
		WorkerID:    task.WorkerID,
	}

	err := s.db.WithContext(ctx).Save(taskModel).Error
	if err != nil {
		return fmt.Errorf("failed to save task to MySQL: %w", err)
	}

	cacheKey := "task:" + task.ID
	taskJSON, _ := json.Marshal(task)
	s.redis.Set(ctx, cacheKey, taskJSON, s.cacheTTL)

	s.logger.Info("Task saved", zap.String("task_id", task.ID))
	return nil
}

func (s *MySQLStateManager) GetTask(ctx context.Context, taskID string) (*pkg.Task, error) {
	cacheKey := "task:" + taskID

	if cached, err := s.redis.Get(ctx, cacheKey).Result(); err == nil {
		var task pkg.Task
		if json.Unmarshal([]byte(cached), &task) == nil {
			s.logger.Debug("Task retrieved from cache", zap.String("task_id", taskID))
			return &task, nil
		}
	}

	var taskModel models.TaskModel
	err := s.db.WithContext(ctx).First(&taskModel, "id = ?", taskID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("task not found: %s", taskID)
		}
		return nil, fmt.Errorf("failed to get task from MySQL: %w", err)
	}

	task := s.modelToTask(&taskModel)

	taskJSON, _ := json.Marshal(task)
	s.redis.Set(ctx, cacheKey, taskJSON, s.cacheTTL)

	return task, nil
}

func (s *MySQLStateManager) GetWorkflowTasks(ctx context.Context, workflowID string) ([]*pkg.Task, error) {
	cacheKey := "workflow_tasks:" + workflowID

	if cached, err := s.redis.Get(ctx, cacheKey).Result(); err == nil {
		var tasks []*pkg.Task
		if json.Unmarshal([]byte(cached), &tasks) == nil {
			s.logger.Debug("Workflow tasks retrieved from cache", zap.String("workflow_id", workflowID))
			return tasks, nil
		}
	}

	var taskModels []models.TaskModel
	err := s.db.WithContext(ctx).Where("workflow_id = ?", workflowID).Order("created_at").Find(&taskModels).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow tasks from MySQL: %w", err)
	}

	tasks := make([]*pkg.Task, len(taskModels))
	for i, taskModel := range taskModels {
		tasks[i] = s.modelToTask(&taskModel)
	}

	tasksJSON, _ := json.Marshal(tasks)
	s.redis.Set(ctx, cacheKey, tasksJSON, time.Minute)

	return tasks, nil
}

func (s *MySQLStateManager) ListWorkflows(ctx context.Context, limit, offset int) ([]*pkg.Workflow, error) {
	var workflowModels []models.WorkflowModel
	err := s.db.WithContext(ctx).Order("created_at DESC").Limit(limit).Offset(offset).Find(&workflowModels).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows from MySQL: %w", err)
	}

	workflows := make([]*pkg.Workflow, len(workflowModels))
	for i, workflowModel := range workflowModels {
		workflows[i] = s.modelToWorkflow(&workflowModel)
	}

	return workflows, nil
}

func (s *MySQLStateManager) modelToWorkflow(model *models.WorkflowModel) *pkg.Workflow {
	var input, output map[string]interface{}
	json.Unmarshal([]byte(model.Input), &input)
	json.Unmarshal([]byte(model.Output), &output)

	var taskIDs []string
	for _, task := range model.Tasks {
		taskIDs = append(taskIDs, task.ID)
	}

	return &pkg.Workflow{
		ID:        model.ID,
		Name:      model.Name,
		Input:     input,
		Output:    output,
		State:     pkg.WorkflowState(model.State),
		Tasks:     taskIDs,
		CreatedAt: model.CreatedAt,
		StartedAt: model.StartedAt,
		EndedAt:   model.EndedAt,
	}
}

func (s *MySQLStateManager) modelToTask(model *models.TaskModel) *pkg.Task {
	var input, output map[string]interface{}
	json.Unmarshal([]byte(model.Input), &input)
	json.Unmarshal([]byte(model.Output), &output)

	return &pkg.Task{
		ID:          model.ID,
		WorkflowID:  model.WorkflowID,
		Type:        model.Type,
		Input:       input,
		Output:      output,
		State:       pkg.TaskState(model.State),
		Error:       model.Error,
		RetryCount:  model.RetryCount,
		MaxRetries:  model.MaxRetries,
		CreatedAt:   model.CreatedAt,
		StartedAt:   model.StartedAt,
		CompletedAt: model.CompletedAt,
		WorkerID:    model.WorkerID,
	}
}

func (s *MySQLStateManager) InvalidateTaskCache(ctx context.Context, taskID string) error {
	pipe := s.redis.Pipeline()
	pipe.Del(ctx, "task:"+taskID)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *MySQLStateManager) InvalidateCache(ctx context.Context, workflowID string) error {
	pipe := s.redis.Pipeline()
	pipe.Del(ctx, "workflow:"+workflowID)
	pipe.Del(ctx, "workflow_tasks:"+workflowID)

	tasks, err := s.GetWorkflowTasks(ctx, workflowID)
	if err == nil {
		for _, task := range tasks {
			pipe.Del(ctx, "task:"+task.ID)
		}
	}

	_, err = pipe.Exec(ctx)
	return err
}

func (s *MySQLStateManager) LogWorkflowExecution(ctx context.Context, workflowID, taskID, level, message string, metadata map[string]interface{}) error {
	metadataJSON, _ := json.Marshal(metadata)

	log := &models.WorkflowExecutionLog{
		WorkflowID: workflowID,
		TaskID:     taskID,
		Level:      level,
		Message:    message,
		Metadata:   string(metadataJSON),
		CreatedAt:  time.Now(),
	}

	return s.db.WithContext(ctx).Create(log).Error
}
