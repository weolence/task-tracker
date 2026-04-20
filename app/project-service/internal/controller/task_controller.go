package controller

import (
	"context"
	"project-service/internal/model"
	"project-service/internal/repository"
	"time"
)

type TaskController struct {
	taskRepository repository.TaskRepository
}

func NewTaskController(taskRepository repository.TaskRepository) *TaskController {
	return &TaskController{taskRepository: taskRepository}
}

func (controller *TaskController) GetTasksByProjectAndAssignee(ctx context.Context, projectID int, assigneeID int) ([]model.Task, error) {
	return controller.taskRepository.GetTasksByProjectAndAssignee(ctx, projectID, assigneeID)
}

func (controller *TaskController) GetAllTasksByProject(ctx context.Context, projectID int) ([]model.Task, error) {
	return controller.taskRepository.GetAllTasksByProject(ctx, projectID)
}

func (controller *TaskController) UpdateTaskStatus(ctx context.Context, taskID int, status model.TaskStatus) error {
	return controller.taskRepository.UpdateTaskStatus(ctx, taskID, status)
}

func (controller *TaskController) CreateTask(ctx context.Context, task model.Task) error {
	task.StartDate = time.Now()
	return controller.taskRepository.CreateTask(ctx, task)
}

func (controller *TaskController) AssignTask(ctx context.Context, taskID int, assigneeID int) error {
	return controller.taskRepository.AssignTask(ctx, taskID, assigneeID)
}

func (controller *TaskController) DeleteTask(ctx context.Context, taskID int) error {
	return controller.taskRepository.DeleteTask(ctx, taskID)
}

func (controller *TaskController) UpdateTask(ctx context.Context, task model.Task) error {
	return controller.taskRepository.UpdateTask(ctx, task)
}

func (controller *TaskController) GetTaskByID(ctx context.Context, taskID int) (*model.Task, error) {
	return controller.taskRepository.GetTaskByID(ctx, taskID)
}
