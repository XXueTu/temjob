package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"temjob/pkg"
)

type Worker struct {
	id           string
	taskQueue    pkg.TaskQueue
	stateManager pkg.StateManager
	logger       *zap.Logger
	handlers     map[string]pkg.TaskHandler
	mu           sync.RWMutex
	running      bool
	stopCh       chan struct{}
}

func NewWorker(taskQueue pkg.TaskQueue, stateManager pkg.StateManager, logger *zap.Logger) *Worker {
	return &Worker{
		id:           uuid.New().String(),
		taskQueue:    taskQueue,
		stateManager: stateManager,
		logger:       logger,
		handlers:     make(map[string]pkg.TaskHandler),
		stopCh:       make(chan struct{}),
	}
}

func (w *Worker) GetID() string {
	return w.id
}

func (w *Worker) RegisterTaskHandler(taskType string, handler pkg.TaskHandler) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.handlers[taskType] = handler
	w.logger.Info("Task handler registered", zap.String("task_type", taskType), zap.String("worker_id", w.id))
}

func (w *Worker) Start(ctx context.Context) error {
	w.running = true
	w.logger.Info("Worker started", zap.String("worker_id", w.id))

	for w.running {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-w.stopCh:
			return nil
		default:
			if err := w.processNextTask(ctx); err != nil {
				w.logger.Error("Error processing task", zap.Error(err))
				time.Sleep(1 * time.Second)
			}
		}
	}

	return nil
}

func (w *Worker) Stop() error {
	w.running = false
	close(w.stopCh)
	w.logger.Info("Worker stopped", zap.String("worker_id", w.id))
	return nil
}

func (w *Worker) processNextTask(ctx context.Context) error {
	task, err := w.taskQueue.Dequeue(ctx, w.id)
	if err != nil {
		return fmt.Errorf("failed to dequeue task: %w", err)
	}

	if task == nil {
		return nil
	}

	w.logger.Info("Processing task", zap.String("task_id", task.ID), zap.String("task_type", task.Type))

	w.mu.RLock()
	handler, exists := w.handlers[task.Type]
	w.mu.RUnlock()

	if !exists {
		errMsg := fmt.Sprintf("no handler found for task type: %s", task.Type)
		w.logger.Error(errMsg)
		return w.taskQueue.UpdateTaskState(ctx, task.ID, pkg.TaskStateFailed, nil, errMsg)
	}

	taskCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	output, err := handler(taskCtx, task.Input)
	if err != nil {
		w.logger.Error("Task execution failed", zap.String("task_id", task.ID), zap.Error(err))
		return w.taskQueue.UpdateTaskState(ctx, task.ID, pkg.TaskStateFailed, nil, err.Error())
	}

	w.logger.Info("Task completed successfully", zap.String("task_id", task.ID))
	return w.taskQueue.UpdateTaskState(ctx, task.ID, pkg.TaskStateCompleted, output, "")
}

func (w *Worker) GetStats(ctx context.Context) (*WorkerStats, error) {
	w.mu.RLock()
	handlerCount := len(w.handlers)
	w.mu.RUnlock()

	return &WorkerStats{
		WorkerID:     w.id,
		Running:      w.running,
		HandlerCount: handlerCount,
	}, nil
}

type WorkerStats struct {
	WorkerID     string `json:"worker_id"`
	Running      bool   `json:"running"`
	HandlerCount int    `json:"handler_count"`
}