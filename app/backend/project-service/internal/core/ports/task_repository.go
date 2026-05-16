package ports

import (
	"context"

	"project-service/internal/core/domain"
)

type TaskRepository interface {
	CreateTask(ctx context.Context, task domain.Task) error
	DeleteTask(ctx context.Context, taskID int) error
	GetTaskByID(ctx context.Context, taskID int) (*domain.Task, error)
	GetTasksByProjectAndAssignee(ctx context.Context, projectID int, assigneeID int) ([]domain.Task, error)
	GetAllTasksByProject(ctx context.Context, projectID int) ([]domain.Task, error)
	GetClosedTasksByProject(ctx context.Context, projectID int) ([]domain.Task, error)
	GetTaskByProjectAndName(ctx context.Context, projectID int32, name string) (*domain.Task, error)
	AssignTask(ctx context.Context, taskID int, assigneeID int) error
	UnassignTask(ctx context.Context, taskID int) error
	UpdateTaskStatus(ctx context.Context, taskID int, status domain.TaskStatus) error
	UpdateTask(ctx context.Context, task domain.Task) error
	CloseTask(ctx context.Context, taskID int) error
}
