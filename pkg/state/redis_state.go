package state

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"

	"github.com/XXueTu/temjob/pkg"
)

const (
	WorkflowPrefix  = "temjob:workflow:"
	TaskPrefix      = "temjob:task:"
	WorkflowListKey = "temjob:workflows"
)

type RedisStateManager struct {
	client *redis.Client
}

func NewRedisStateManager(client *redis.Client) *RedisStateManager {
	return &RedisStateManager{
		client: client,
	}
}

func (s *RedisStateManager) SaveWorkflow(ctx context.Context, workflow *pkg.Workflow) error {
	data, err := json.Marshal(workflow)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}

	pipe := s.client.Pipeline()
	pipe.Set(ctx, WorkflowPrefix+workflow.ID, data, 0)
	pipe.ZAdd(ctx, WorkflowListKey, &redis.Z{
		Score:  float64(workflow.CreatedAt.Unix()),
		Member: workflow.ID,
	})

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to save workflow: %w", err)
	}

	return nil
}

func (s *RedisStateManager) GetWorkflow(ctx context.Context, workflowID string) (*pkg.Workflow, error) {
	data, err := s.client.Get(ctx, WorkflowPrefix+workflowID).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("workflow not found: %s", workflowID)
		}
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	var workflow pkg.Workflow
	if err := json.Unmarshal([]byte(data), &workflow); err != nil {
		return nil, fmt.Errorf("failed to unmarshal workflow: %w", err)
	}

	return &workflow, nil
}

func (s *RedisStateManager) SaveTask(ctx context.Context, task *pkg.Task) error {
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	err = s.client.Set(ctx, TaskPrefix+task.ID, data, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	return nil
}

func (s *RedisStateManager) GetTask(ctx context.Context, taskID string) (*pkg.Task, error) {
	data, err := s.client.Get(ctx, TaskPrefix+taskID).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("task not found: %s", taskID)
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	var task pkg.Task
	if err := json.Unmarshal([]byte(data), &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

func (s *RedisStateManager) GetWorkflowTasks(ctx context.Context, workflowID string) ([]*pkg.Task, error) {
	workflow, err := s.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	tasks := make([]*pkg.Task, 0, len(workflow.Tasks))
	for _, taskID := range workflow.Tasks {
		task, err := s.GetTask(ctx, taskID)
		if err != nil {
			return nil, fmt.Errorf("failed to get task %s: %w", taskID, err)
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (s *RedisStateManager) ListWorkflows(ctx context.Context, limit, offset int) ([]*pkg.Workflow, error) {
	workflowIDs, err := s.client.ZRevRange(ctx, WorkflowListKey, int64(offset), int64(offset+limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow IDs: %w", err)
	}

	workflows := make([]*pkg.Workflow, 0, len(workflowIDs))
	for _, workflowID := range workflowIDs {
		workflow, err := s.GetWorkflow(ctx, workflowID)
		if err != nil {
			continue
		}
		workflows = append(workflows, workflow)
	}

	return workflows, nil
}

func (s *RedisStateManager) UpdateWorkflowState(ctx context.Context, workflowID string, state pkg.WorkflowState) error {
	workflow, err := s.GetWorkflow(ctx, workflowID)
	if err != nil {
		return err
	}

	workflow.State = state
	return s.SaveWorkflow(ctx, workflow)
}

func (s *RedisStateManager) GetWorkflowStats(ctx context.Context) (map[string]int64, error) {
	workflows, err := s.ListWorkflows(ctx, 1000, 0)
	if err != nil {
		return nil, err
	}

	stats := make(map[string]int64)
	for _, workflow := range workflows {
		stats[string(workflow.State)]++
	}

	return stats, nil
}
