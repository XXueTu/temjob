package web

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/XXueTu/temjob/pkg"
)

type Server struct {
	stateManager pkg.StateManager
	taskQueue    pkg.TaskQueue
	engine       pkg.WorkflowEngine
	logger       *zap.Logger
	router       *gin.Engine
}

func NewServer(stateManager pkg.StateManager, taskQueue pkg.TaskQueue, engine pkg.WorkflowEngine, logger *zap.Logger) *Server {
	server := &Server{
		stateManager: stateManager,
		taskQueue:    taskQueue,
		engine:       engine,
		logger:       logger,
		router:       gin.Default(),
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	api := s.router.Group("/api/v1")
	{
		api.GET("/workflows", s.listWorkflows)
		api.GET("/workflows/:id", s.getWorkflow)
		api.POST("/workflows/:id/cancel", s.cancelWorkflow)
		api.GET("/workflows/:id/tasks", s.getWorkflowTasks)
		api.GET("/tasks/:id", s.getTask)
		api.GET("/stats", s.getStats)
	}

	s.router.Static("/static", "./web/static")
	s.router.LoadHTMLGlob("web/templates/*")
	s.router.GET("/", s.dashboard)
	s.router.GET("/workflows", s.workflowsPage)
	s.router.GET("/workflows/:id", s.workflowDetailPage)
}

func (s *Server) listWorkflows(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	workflows, err := s.stateManager.ListWorkflows(c.Request.Context(), limit, offset)
	if err != nil {
		s.logger.Error("Failed to list workflows", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"workflows": workflows})
}

func (s *Server) getWorkflow(c *gin.Context) {
	workflowID := c.Param("id")

	workflow, err := s.stateManager.GetWorkflow(c.Request.Context(), workflowID)
	if err != nil {
		s.logger.Error("Failed to get workflow", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, workflow)
}

func (s *Server) cancelWorkflow(c *gin.Context) {
	workflowID := c.Param("id")

	err := s.engine.CancelWorkflow(c.Request.Context(), workflowID)
	if err != nil {
		s.logger.Error("Failed to cancel workflow", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Workflow canceled successfully"})
}

func (s *Server) getWorkflowTasks(c *gin.Context) {
	workflowID := c.Param("id")

	tasks, err := s.stateManager.GetWorkflowTasks(c.Request.Context(), workflowID)
	if err != nil {
		s.logger.Error("Failed to get workflow tasks", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tasks": tasks})
}

func (s *Server) getTask(c *gin.Context) {
	taskID := c.Param("id")

	task, err := s.stateManager.GetTask(c.Request.Context(), taskID)
	if err != nil {
		s.logger.Error("Failed to get task", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (s *Server) getStats(c *gin.Context) {
	ctx := c.Request.Context()

	workflows, err := s.stateManager.ListWorkflows(ctx, 1000, 0)
	if err != nil {
		s.logger.Error("Failed to get workflows for stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	stats := map[string]int{
		"total_workflows": len(workflows),
		"pending":         0,
		"running":         0,
		"completed":       0,
		"failed":          0,
		"canceled":        0,
	}

	for _, workflow := range workflows {
		stats[string(workflow.State)]++
	}

	c.JSON(http.StatusOK, stats)
}

func (s *Server) dashboard(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title": "TemJob Dashboard",
	})
}

func (s *Server) workflowsPage(c *gin.Context) {
	c.HTML(http.StatusOK, "workflows.html", gin.H{
		"title": "Workflows",
	})
}

func (s *Server) workflowDetailPage(c *gin.Context) {
	workflowID := c.Param("id")
	c.HTML(http.StatusOK, "workflow_detail.html", gin.H{
		"title":      "Workflow Details",
		"workflowID": workflowID,
	})
}

func (s *Server) Start(port string) error {
	s.logger.Info("Starting web server", zap.String("port", port))
	return s.router.Run(":" + port)
}
