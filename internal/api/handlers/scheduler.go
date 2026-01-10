package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/slipstream/slipstream/internal/scheduler"
)

// SchedulerHandler handles scheduler-related API requests.
type SchedulerHandler struct {
	scheduler *scheduler.Scheduler
}

// NewSchedulerHandler creates a new scheduler handler.
func NewSchedulerHandler(sched *scheduler.Scheduler) *SchedulerHandler {
	return &SchedulerHandler{
		scheduler: sched,
	}
}

// ListTasks returns all scheduled tasks.
// GET /api/v1/scheduler/tasks
func (h *SchedulerHandler) ListTasks(c echo.Context) error {
	tasks := h.scheduler.ListTasks()
	return c.JSON(http.StatusOK, tasks)
}

// GetTask returns information about a specific task.
// GET /api/v1/scheduler/tasks/:id
func (h *SchedulerHandler) GetTask(c echo.Context) error {
	taskID := c.Param("id")
	task, err := h.scheduler.GetTask(taskID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
	}
	return c.JSON(http.StatusOK, task)
}

// RunTask manually triggers a task to run.
// POST /api/v1/scheduler/tasks/:id/run
func (h *SchedulerHandler) RunTask(c echo.Context) error {
	taskID := c.Param("id")
	if err := h.scheduler.RunNow(taskID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}
	return c.JSON(http.StatusAccepted, map[string]string{
		"message": "Task started",
		"taskId":  taskID,
	})
}
