package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"temjob/pkg"
	"temjob/pkg/config"
	"temjob/pkg/models"
	"temjob/pkg/queue"
	"temjob/pkg/sdk"
	"temjob/pkg/state"
	"temjob/pkg/worker"
	"temjob/pkg/workflow"
	"temjob/web"
)

func main() {
	// Parse command line flags
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Initialize logger
	logger, err := initLogger(cfg.Logging)
	if err != nil {
		log.Fatal("Failed to create logger:", err)
	}
	defer logger.Sync()

	// Set Gin mode
	gin.SetMode(cfg.Server.Mode)

	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}

	// Initialize MySQL
	db, err := gorm.Open(mysql.Open(cfg.Database.MySQL.DSN()), &gorm.Config{})
	if err != nil {
		logger.Fatal("Failed to connect to MySQL", zap.Error(err))
	}

	// Auto-migrate database tables
	if err := models.AutoMigrate(db); err != nil {
		logger.Fatal("Failed to migrate database", zap.Error(err))
	}

	// Initialize components
	stateManager := state.NewMySQLStateManager(db, redisClient, logger)
	taskQueue := queue.NewRedisTaskQueue(redisClient, logger, stateManager)
	engine := workflow.NewEngine(stateManager, taskQueue, logger)
	workerInstance := worker.NewWorker(taskQueue, stateManager, logger)

	// Register example workflow
	registerExampleWorkflow(engine, workerInstance)

	// Start components
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// Start workflow engine
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := engine.Start(ctx); err != nil {
			logger.Error("Workflow engine error", zap.Error(err))
		}
	}()

	// Start worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := workerInstance.Start(ctx); err != nil {
			logger.Error("Worker error", zap.Error(err))
		}
	}()

	// Start web server
	webServer := web.NewServer(stateManager, taskQueue, engine, logger)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := webServer.Start(cfg.Server.Port); err != nil {
			logger.Error("Web server error", zap.Error(err))
		}
	}()

	// Submit example workflow
	go func() {
		time.Sleep(2 * time.Second)
		workflowID, err := engine.SubmitWorkflow(ctx, "data_processing", map[string]interface{}{
			"input_file": "data.csv",
			"output_dir": "/tmp/output",
		})
		if err != nil {
			logger.Error("Failed to submit workflow", zap.Error(err))
		} else {
			logger.Info("Submitted example workflow", zap.String("workflow_id", workflowID))
		}
	}()

	logger.Info("TemJob started successfully")
	logger.Info("Web UI available", zap.String("url", "http://localhost:"+cfg.Server.Port))

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutting down...")
	cancel()

	// Stop components
	engine.Stop()
	workerInstance.Stop()

	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("Shutdown completed")
	case <-time.After(30 * time.Second):
		logger.Warn("Shutdown timeout exceeded")
	}
}

func registerExampleWorkflow(engine pkg.WorkflowEngine, worker pkg.Worker) {
	// Register task handlers
	worker.RegisterTaskHandler("validate_input", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
		inputFile := input["input_file"].(string)
		return map[string]interface{}{
			"validated":  true,
			"file_size":  1024,
			"input_file": inputFile,
		}, nil
	}))

	worker.RegisterTaskHandler("process_data", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
		inputFile := input["input_file"].(string)
		return map[string]interface{}{
			"processed_records": 100,
			"input_file":        inputFile,
			"output_file":       "processed_" + inputFile,
		}, nil
	}))

	worker.RegisterTaskHandler("generate_report", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
		outputFile, ok := input["output_file"].(string)
		if !ok || outputFile == "" {
			// If output_file is not available, use input_file as fallback
			inputFile, _ := input["input_file"].(string)
			outputFile = "processed_" + inputFile
		}
		return map[string]interface{}{
			"report_file": "report_" + outputFile,
			"summary":     "Data processing completed successfully",
		}, nil
	}))

	// Register workflow
	workflowDef := sdk.NewWorkflowBuilder("data_processing").
		AddTask("validate_input", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
			return input, nil
		}), 3).
		AddTask("process_data", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
			return input, nil
		}), 3).
		AddTask("generate_report", sdk.SimpleTaskHandler(func(input map[string]interface{}) (map[string]interface{}, error) {
			return input, nil
		}), 3).
		AddStep("validate_input").Then().
		AddStep("process_data").DependsOn("validate_input").Then().
		AddStep("generate_report").DependsOn("process_data").Then().
		Build()

	engine.RegisterWorkflow(workflowDef)
}

func initLogger(cfg config.LoggingConfig) (*zap.Logger, error) {
	var zapConfig zap.Config

	switch cfg.Level {
	case "debug":
		zapConfig = zap.NewDevelopmentConfig()
	case "info", "warn", "error":
		zapConfig = zap.NewProductionConfig()
		switch cfg.Level {
		case "warn":
			zapConfig.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
		case "error":
			zapConfig.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
		}
	default:
		zapConfig = zap.NewProductionConfig()
	}

	if cfg.Format == "console" {
		zapConfig.Encoding = "console"
		zapConfig.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	}

	return zapConfig.Build()
}
