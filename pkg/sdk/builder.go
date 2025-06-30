package sdk

import (
	"context"
	"temjob/pkg"
)

type WorkflowBuilder struct {
	name  string
	tasks map[string]pkg.TaskDefinition
	flow  []pkg.WorkflowStep
}

func NewWorkflowBuilder(name string) *WorkflowBuilder {
	return &WorkflowBuilder{
		name:  name,
		tasks: make(map[string]pkg.TaskDefinition),
		flow:  []pkg.WorkflowStep{},
	}
}

func (wb *WorkflowBuilder) AddTask(taskType string, handler pkg.TaskHandler, maxRetries int) *WorkflowBuilder {
	wb.tasks[taskType] = pkg.TaskDefinition{
		Type:       taskType,
		Handler:    handler,
		MaxRetries: maxRetries,
	}
	return wb
}

func (wb *WorkflowBuilder) AddStep(taskType string) *StepBuilder {
	return &StepBuilder{
		workflowBuilder: wb,
		step: pkg.WorkflowStep{
			TaskType:  taskType,
			DependsOn: []string{},
		},
	}
}

func (wb *WorkflowBuilder) Build() pkg.WorkflowDefinition {
	return pkg.WorkflowDefinition{
		Name:  wb.name,
		Tasks: wb.tasks,
		Flow:  wb.flow,
	}
}

type StepBuilder struct {
	workflowBuilder *WorkflowBuilder
	step            pkg.WorkflowStep
}

func (sb *StepBuilder) DependsOn(dependencies ...string) *StepBuilder {
	sb.step.DependsOn = append(sb.step.DependsOn, dependencies...)
	return sb
}

func (sb *StepBuilder) When(condition func(map[string]interface{}) bool) *StepBuilder {
	sb.step.Condition = condition
	return sb
}

func (sb *StepBuilder) OnError(errorHandler string) *StepBuilder {
	sb.step.OnError = errorHandler
	return sb
}

func (sb *StepBuilder) Then() *WorkflowBuilder {
	sb.workflowBuilder.flow = append(sb.workflowBuilder.flow, sb.step)
	return sb.workflowBuilder
}

type TaskHandlerFunc func(input map[string]interface{}) (map[string]interface{}, error)

func SimpleTaskHandler(fn TaskHandlerFunc) pkg.TaskHandler {
	return func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return fn(input)
	}
}

func AsyncTaskHandler(fn func(input map[string]interface{}) (map[string]interface{}, error)) pkg.TaskHandler {
	return func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		resultCh := make(chan struct {
			output map[string]interface{}
			err    error
		}, 1)

		go func() {
			output, err := fn(input)
			resultCh <- struct {
				output map[string]interface{}
				err    error
			}{output, err}
		}()

		select {
		case result := <-resultCh:
			return result.output, result.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}