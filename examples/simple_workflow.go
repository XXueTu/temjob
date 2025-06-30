package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/XXueTu/temjob/pkg/sdk"
)

func main() {
	// Initialize client
	client, err := sdk.NewClient(sdk.ClientConfig{
		RedisAddr:     "localhost:6379",
		RedisPassword: "",
		RedisDB:       0,
	})
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}
	defer client.Close()

	// Register task handlers
	client.RegisterTaskHandler("send_email", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
		recipient := input["recipient"].(string)
		subject := input["subject"].(string)

		// Simulate sending email
		fmt.Printf("Sending email to %s with subject: %s\n", recipient, subject)
		time.Sleep(1 * time.Second)

		return map[string]interface{}{
			"email_sent": true,
			"sent_at":    time.Now().Format(time.RFC3339),
		}, nil
	}))

	client.RegisterTaskHandler("log_activity", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
		activity := input["activity"].(string)

		fmt.Printf("Logging activity: %s\n", activity)

		return map[string]interface{}{
			"logged": true,
			"log_id": "log_" + time.Now().Format("20060102150405"),
		}, nil
	}))

	// Define workflow
	workflowDef := sdk.NewWorkflowBuilder("notification_workflow").
		AddTask("send_email", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
			return input, nil
		}), 3).
		AddTask("log_activity", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
			return input, nil
		}), 3).
		AddStep("send_email").Then().
		AddStep("log_activity").DependsOn("send_email").Then().
		Build()

	// Register workflow
	client.RegisterWorkflow(workflowDef)

	// Start engine and worker
	ctx := context.Background()

	go func() {
		if err := client.StartEngine(ctx); err != nil {
			log.Printf("Engine error: %v", err)
		}
	}()

	go func() {
		if err := client.StartWorker(ctx); err != nil {
			log.Printf("Worker error: %v", err)
		}
	}()

	// Wait for services to start
	time.Sleep(2 * time.Second)

	// Submit workflow
	workflowID, err := client.SubmitWorkflow(ctx, "notification_workflow", map[string]interface{}{
		"recipient": "user@example.com",
		"subject":   "Welcome to TemJob!",
		"activity":  "User registration notification sent",
	})
	if err != nil {
		log.Fatal("Failed to submit workflow:", err)
	}

	fmt.Printf("Submitted workflow: %s\n", workflowID)

	// Monitor workflow progress
	for {
		workflow, err := client.GetWorkflow(ctx, workflowID)
		if err != nil {
			log.Printf("Failed to get workflow: %v", err)
			break
		}

		fmt.Printf("Workflow %s state: %s\n", workflowID, workflow.State)

		if workflow.State == "completed" || workflow.State == "failed" {
			fmt.Printf("Workflow finished with state: %s\n", workflow.State)
			if workflow.Output != nil {
				fmt.Printf("Output: %+v\n", workflow.Output)
			}
			break
		}

		time.Sleep(2 * time.Second)
	}
}
