package sdk

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"github.com/XXueTu/temjob/pkg"
	"github.com/XXueTu/temjob/pkg/queue"
	"github.com/XXueTu/temjob/pkg/state"
	"github.com/XXueTu/temjob/pkg/worker"
	"github.com/XXueTu/temjob/pkg/workflow"
)

type Client struct {
	engine       pkg.WorkflowEngine
	worker       pkg.Worker
	stateManager pkg.StateManager
	taskQueue    pkg.TaskQueue
	logger       *zap.Logger
}

type ClientConfig struct {
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

func NewClient(config ClientConfig) (*Client, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	stateManager := state.NewRedisStateManager(redisClient)
	taskQueue := queue.NewRedisTaskQueue(redisClient, logger, stateManager)
	engine := workflow.NewEngine(stateManager, taskQueue, logger)
	workerInstance := worker.NewWorker(taskQueue, stateManager, logger)

	return &Client{
		engine:       engine,
		worker:       workerInstance,
		stateManager: stateManager,
		taskQueue:    taskQueue,
		logger:       logger,
	}, nil
}

func (c *Client) RegisterWorkflow(definition pkg.WorkflowDefinition) {
	c.engine.RegisterWorkflow(definition)
}

func (c *Client) RegisterTaskHandler(taskType string, handler pkg.TaskHandler) {
	c.worker.RegisterTaskHandler(taskType, handler)
}

func (c *Client) SubmitWorkflow(ctx context.Context, workflowName string, input map[string]interface{}) (string, error) {
	return c.engine.SubmitWorkflow(ctx, workflowName, input)
}

func (c *Client) GetWorkflow(ctx context.Context, workflowID string) (*pkg.Workflow, error) {
	return c.engine.GetWorkflow(ctx, workflowID)
}

func (c *Client) CancelWorkflow(ctx context.Context, workflowID string) error {
	return c.engine.CancelWorkflow(ctx, workflowID)
}

func (c *Client) GetTask(ctx context.Context, taskID string) (*pkg.Task, error) {
	return c.stateManager.GetTask(ctx, taskID)
}

func (c *Client) ListWorkflows(ctx context.Context, limit, offset int) ([]*pkg.Workflow, error) {
	return c.stateManager.ListWorkflows(ctx, limit, offset)
}

func (c *Client) GetWorkflowTasks(ctx context.Context, workflowID string) ([]*pkg.Task, error) {
	return c.stateManager.GetWorkflowTasks(ctx, workflowID)
}

func (c *Client) StartWorker(ctx context.Context) error {
	return c.worker.Start(ctx)
}

func (c *Client) StopWorker() error {
	return c.worker.Stop()
}

func (c *Client) StartEngine(ctx context.Context) error {
	return c.engine.Start(ctx)
}

func (c *Client) StopEngine() error {
	return c.engine.Stop()
}

func (c *Client) GetWorkerID() string {
	return c.worker.GetID()
}

func (c *Client) Close() error {
	c.worker.Stop()
	c.engine.Stop()
	return nil
}
