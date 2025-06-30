package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/XXueTu/temjob/pkg"
)

type Engine struct {
	stateManager pkg.StateManager
	taskQueue    pkg.TaskQueue
	logger       *zap.Logger
	definitions  map[string]pkg.WorkflowDefinition
	mu           sync.RWMutex
	running      bool
	stopCh       chan struct{}
}

func NewEngine(stateManager pkg.StateManager, taskQueue pkg.TaskQueue, logger *zap.Logger) *Engine {
	return &Engine{
		stateManager: stateManager,
		taskQueue:    taskQueue,
		logger:       logger,
		definitions:  make(map[string]pkg.WorkflowDefinition),
		stopCh:       make(chan struct{}),
	}
}

func (e *Engine) RegisterWorkflow(definition pkg.WorkflowDefinition) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.definitions[definition.Name] = definition
	e.logger.Info("Workflow registered", zap.String("name", definition.Name))
}

func (e *Engine) SubmitWorkflow(ctx context.Context, workflowName string, input map[string]interface{}) (string, error) {
	e.mu.RLock()
	definition, exists := e.definitions[workflowName]
	e.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("workflow definition not found: %s", workflowName)
	}

	workflowID := pkg.NewWorkflowID()
	workflow := &pkg.Workflow{
		ID:        workflowID,
		Name:      workflowName,
		Input:     input,
		State:     pkg.WorkflowStatePending,
		Tasks:     []string{},
		CreatedAt: time.Now(),
	}

	if err := e.stateManager.SaveWorkflow(ctx, workflow); err != nil {
		return "", fmt.Errorf("failed to save workflow: %w", err)
	}

	go e.executeWorkflow(context.Background(), workflowID, definition)

	e.logger.Info("Workflow submitted", zap.String("workflow_id", workflowID), zap.String("name", workflowName))
	return workflowID, nil
}

func (e *Engine) GetWorkflow(ctx context.Context, workflowID string) (*pkg.Workflow, error) {
	return e.stateManager.GetWorkflow(ctx, workflowID)
}

func (e *Engine) CancelWorkflow(ctx context.Context, workflowID string) error {
	workflow, err := e.stateManager.GetWorkflow(ctx, workflowID)
	if err != nil {
		return err
	}

	if workflow.State == pkg.WorkflowStateCompleted || workflow.State == pkg.WorkflowStateFailed {
		return fmt.Errorf("cannot cancel workflow in state: %s", workflow.State)
	}

	workflow.State = pkg.WorkflowStateCanceled
	now := time.Now()
	workflow.EndedAt = &now

	return e.stateManager.SaveWorkflow(ctx, workflow)
}

func (e *Engine) Start(ctx context.Context) error {
	e.running = true
	go e.monitorWorkflows(ctx)
	e.logger.Info("Workflow engine started")
	return nil
}

func (e *Engine) Stop() error {
	e.running = false
	close(e.stopCh)
	e.logger.Info("Workflow engine stopped")
	return nil
}

func (e *Engine) executeWorkflow(ctx context.Context, workflowID string, definition pkg.WorkflowDefinition) {
	workflow, err := e.stateManager.GetWorkflow(ctx, workflowID)
	if err != nil {
		e.logger.Error("Failed to get workflow", zap.Error(err))
		return
	}

	workflow.State = pkg.WorkflowStateRunning
	now := time.Now()
	workflow.StartedAt = &now

	if err := e.stateManager.SaveWorkflow(ctx, workflow); err != nil {
		e.logger.Error("Failed to update workflow state", zap.Error(err))
		return
	}

	// Execute tasks sequentially according to dependencies
	e.executeWorkflowTasks(ctx, workflowID, definition)
}

func (e *Engine) executeWorkflowTasks(ctx context.Context, workflowID string, definition pkg.WorkflowDefinition) {
	completedTasks := make(map[string]*pkg.Task)
	workflowContext := make(map[string]interface{})

	// Get initial workflow input
	workflow, err := e.stateManager.GetWorkflow(ctx, workflowID)
	if err != nil {
		e.failWorkflow(ctx, workflowID, fmt.Sprintf("failed to get workflow: %v", err))
		return
	}

	// Copy initial input to context
	for k, v := range workflow.Input {
		workflowContext[k] = v
	}

	for {
		readyTasks := e.getReadyTasks(definition.Flow, completedTasks, workflowContext)
		if len(readyTasks) == 0 {
			break
		}

		// Submit ready tasks
		for _, step := range readyTasks {
			taskDef, exists := definition.Tasks[step.TaskType]
			if !exists {
				e.failWorkflow(ctx, workflowID, fmt.Sprintf("task definition not found: %s", step.TaskType))
				return
			}

			taskID := pkg.NewTaskID()
			task := &pkg.Task{
				ID:         taskID,
				WorkflowID: workflowID,
				Type:       step.TaskType,
				Input:      workflowContext,
				State:      pkg.TaskStatePending,
				MaxRetries: taskDef.MaxRetries,
				CreatedAt:  time.Now(),
			}

			if err := e.stateManager.SaveTask(ctx, task); err != nil {
				e.failWorkflow(ctx, workflowID, fmt.Sprintf("failed to save task: %v", err))
				return
			}

			workflow.Tasks = append(workflow.Tasks, taskID)
			if err := e.stateManager.SaveWorkflow(ctx, workflow); err != nil {
				e.failWorkflow(ctx, workflowID, fmt.Sprintf("failed to update workflow: %v", err))
				return
			}

			if err := e.taskQueue.Enqueue(ctx, task); err != nil {
				e.failWorkflow(ctx, workflowID, fmt.Sprintf("failed to enqueue task: %v", err))
				return
			}

			e.logger.Info("Task submitted", zap.String("task_id", taskID), zap.String("type", step.TaskType))
		}

		// Wait for submitted tasks to complete
		for _, step := range readyTasks {
			if err := e.waitForTaskCompletion(ctx, workflowID, step.TaskType, completedTasks, workflowContext); err != nil {
				e.failWorkflow(ctx, workflowID, fmt.Sprintf("task %s failed: %v", step.TaskType, err))
				return
			}
		}

		if len(completedTasks) == len(definition.Flow) {
			break
		}
	}

	// Complete workflow
	workflow.State = pkg.WorkflowStateCompleted
	now := time.Now()
	workflow.EndedAt = &now
	workflow.Output = workflowContext

	if err := e.stateManager.SaveWorkflow(ctx, workflow); err != nil {
		e.logger.Error("Failed to complete workflow", zap.Error(err))
		return
	}

	e.logger.Info("Workflow completed", zap.String("workflow_id", workflowID))
}

func (e *Engine) waitForTaskCompletion(ctx context.Context, workflowID, taskType string, completedTasks map[string]*pkg.Task, workflowContext map[string]interface{}) error {
	// Find the task by type in the workflow
	workflow, err := e.stateManager.GetWorkflow(ctx, workflowID)
	if err != nil {
		return err
	}

	var targetTask *pkg.Task
	for _, taskID := range workflow.Tasks {
		task, err := e.stateManager.GetTask(ctx, taskID)
		if err != nil {
			continue
		}
		if task.Type == taskType && task.State != pkg.TaskStateCompleted && task.State != pkg.TaskStateFailed {
			targetTask = task
			break
		}
	}

	if targetTask == nil {
		return fmt.Errorf("task %s not found", taskType)
	}

	// Poll for task completion
	for {
		task, err := e.stateManager.GetTask(ctx, targetTask.ID)
		if err != nil {
			return err
		}

		if task.State == pkg.TaskStateCompleted {
			completedTasks[taskType] = task
			// Merge task output into workflow context
			if task.Output != nil {
				for k, v := range task.Output {
					workflowContext[k] = v
				}
			}
			e.logger.Info("Task completed", zap.String("task_id", task.ID), zap.String("type", taskType))
			return nil
		} else if task.State == pkg.TaskStateFailed {
			return fmt.Errorf("task failed: %s", task.Error)
		}

		time.Sleep(1 * time.Second)
	}
}

func (e *Engine) getReadyTasks(flow []pkg.WorkflowStep, completed map[string]*pkg.Task, context map[string]interface{}) []pkg.WorkflowStep {
	var ready []pkg.WorkflowStep

	for _, step := range flow {
		if completed[step.TaskType] != nil {
			continue
		}

		allDepsCompleted := true
		for _, dep := range step.DependsOn {
			if completed[dep] == nil {
				allDepsCompleted = false
				break
			}
		}

		if allDepsCompleted && (step.Condition == nil || step.Condition(context)) {
			ready = append(ready, step)
		}
	}

	return ready
}

func (e *Engine) allTasksCompleted(flow []pkg.WorkflowStep, completed map[string]*pkg.Task) bool {
	for _, step := range flow {
		if completed[step.TaskType] == nil {
			return false
		}
	}
	return true
}

func (e *Engine) failWorkflow(ctx context.Context, workflowID string, reason string) {
	workflow, err := e.stateManager.GetWorkflow(ctx, workflowID)
	if err != nil {
		e.logger.Error("Failed to get workflow for failure", zap.Error(err))
		return
	}

	workflow.State = pkg.WorkflowStateFailed
	now := time.Now()
	workflow.EndedAt = &now

	if err := e.stateManager.SaveWorkflow(ctx, workflow); err != nil {
		e.logger.Error("Failed to save failed workflow", zap.Error(err))
		return
	}

	e.logger.Error("Workflow failed", zap.String("workflow_id", workflowID), zap.String("reason", reason))
}

func (e *Engine) monitorWorkflows(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.checkWorkflowProgress(ctx)
		}
	}
}

func (e *Engine) checkWorkflowProgress(ctx context.Context) {
	workflows, err := e.stateManager.ListWorkflows(ctx, 100, 0)
	if err != nil {
		e.logger.Error("Failed to list workflows for monitoring", zap.Error(err))
		return
	}

	for _, workflow := range workflows {
		if workflow.State == pkg.WorkflowStateRunning {
			e.checkWorkflowTasks(ctx, workflow)
		}
	}
}

func (e *Engine) checkWorkflowTasks(ctx context.Context, workflow *pkg.Workflow) {
	tasks, err := e.stateManager.GetWorkflowTasks(ctx, workflow.ID)
	if err != nil {
		e.logger.Error("Failed to get workflow tasks", zap.Error(err))
		return
	}

	allCompleted := true
	hasFailures := false

	for _, task := range tasks {
		if task.State == pkg.TaskStateRunning || task.State == pkg.TaskStatePending || task.State == pkg.TaskStateRetrying {
			allCompleted = false
		}
		if task.State == pkg.TaskStateFailed && task.RetryCount >= task.MaxRetries {
			hasFailures = true
		}
	}

	if hasFailures {
		e.failWorkflow(ctx, workflow.ID, "workflow has failed tasks")
	} else if allCompleted {
		workflow.State = pkg.WorkflowStateCompleted
		now := time.Now()
		workflow.EndedAt = &now
		e.stateManager.SaveWorkflow(ctx, workflow)
	}
}
