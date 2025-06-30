package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"temjob/pkg"
	cstate "temjob/pkg/state"
)

const (
	TaskQueueKey       = "temjob:queue:tasks"
	ProcessingQueueKey = "temjob:queue:processing"
	QueueTaskPrefix    = "temjob:queue:task:"
)

type RedisTaskQueue struct {
	client       *redis.Client
	logger       *zap.Logger
	stateManager pkg.StateManager
}

func NewRedisTaskQueue(client *redis.Client, logger *zap.Logger, stateManager pkg.StateManager) *RedisTaskQueue {
	return &RedisTaskQueue{
		client:       client,
		logger:       logger,
		stateManager: stateManager,
	}
}

func (q *RedisTaskQueue) Enqueue(ctx context.Context, task *pkg.Task) error {
	taskData, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	pipe := q.client.Pipeline()
	pipe.HSet(ctx, QueueTaskPrefix+task.ID, "data", taskData)
	pipe.LPush(ctx, TaskQueueKey, task.ID)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	q.logger.Info("Task enqueued", zap.String("task_id", task.ID))
	return nil
}

func (q *RedisTaskQueue) Dequeue(ctx context.Context, workerID string) (*pkg.Task, error) {
	result, err := q.client.BRPopLPush(ctx, TaskQueueKey, ProcessingQueueKey+":"+workerID, 30*time.Second).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to dequeue task: %w", err)
	}

	taskID := result
	taskData, err := q.client.HGet(ctx, QueueTaskPrefix+taskID, "data").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get task data: %w", err)
	}

	var task pkg.Task
	if err := json.Unmarshal([]byte(taskData), &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	task.State = pkg.TaskStateRunning
	task.WorkerID = workerID
	now := time.Now()
	task.StartedAt = &now

	if err := q.updateTaskData(ctx, &task); err != nil {
		return nil, fmt.Errorf("failed to update task state: %w", err)
	}

	q.logger.Info("Task dequeued", zap.String("task_id", task.ID), zap.String("worker_id", workerID))
	return &task, nil
}

func (q *RedisTaskQueue) UpdateTaskState(ctx context.Context, taskID string, state pkg.TaskState, output map[string]interface{}, errMsg string) error {
	taskData, err := q.client.HGet(ctx, QueueTaskPrefix+taskID, "data").Result()
	if err != nil {
		return fmt.Errorf("failed to get task data: %w", err)
	}

	var task pkg.Task
	if err := json.Unmarshal([]byte(taskData), &task); err != nil {
		return fmt.Errorf("failed to unmarshal task: %w", err)
	}

	task.State = state
	if output != nil {
		task.Output = output
	}
	if errMsg != "" {
		task.Error = errMsg
	}

	if state == pkg.TaskStateCompleted || state == pkg.TaskStateFailed {
		now := time.Now()
		task.CompletedAt = &now
	}

	if err := q.updateTaskData(ctx, &task); err != nil {
		return fmt.Errorf("failed to update task data: %w", err)
	}

	// Also update the state manager (MySQL) with the task state
	if q.stateManager != nil {
		if err := q.stateManager.SaveTask(ctx, &task); err != nil {
			q.logger.Warn("Failed to sync task state to state manager", zap.Error(err))
		} else {
			// Clear cache to ensure fresh data is retrieved
			if mysqlState, ok := q.stateManager.(*cstate.MySQLStateManager); ok {
				mysqlState.InvalidateCache(ctx, task.WorkflowID)
			}
		}
	}

	if task.WorkerID != "" && (state == pkg.TaskStateCompleted || state == pkg.TaskStateFailed) {
		q.client.LRem(ctx, ProcessingQueueKey+":"+task.WorkerID, 1, taskID)
	}

	if state == pkg.TaskStateFailed && task.RetryCount < task.MaxRetries {
		task.RetryCount++
		task.State = pkg.TaskStateRetrying
		task.WorkerID = ""
		task.StartedAt = nil
		task.CompletedAt = nil

		if err := q.updateTaskData(ctx, &task); err != nil {
			return fmt.Errorf("failed to update retry task: %w", err)
		}

		if err := q.Enqueue(ctx, &task); err != nil {
			return fmt.Errorf("failed to requeue task for retry: %w", err)
		}

		q.logger.Info("Task requeued for retry", zap.String("task_id", taskID), zap.Int("retry_count", task.RetryCount))
	}

	q.logger.Info("Task state updated", zap.String("task_id", taskID), zap.String("state", string(state)))
	return nil
}

func (q *RedisTaskQueue) updateTaskData(ctx context.Context, task *pkg.Task) error {
	taskData, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	return q.client.HSet(ctx, QueueTaskPrefix+task.ID, "data", taskData).Err()
}

func (q *RedisTaskQueue) GetQueueLength(ctx context.Context) (int64, error) {
	return q.client.LLen(ctx, TaskQueueKey).Result()
}

func (q *RedisTaskQueue) GetProcessingTasks(ctx context.Context, workerID string) ([]string, error) {
	return q.client.LRange(ctx, ProcessingQueueKey+":"+workerID, 0, -1).Result()
}
