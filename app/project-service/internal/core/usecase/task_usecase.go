package usecase

import (
	"context"
	"errors"
	"time"

	"project-service/internal/core/domain"
	"project-service/internal/core/ports"
	projectv1 "project-service/api/proto/projectv1"
)

type TaskUseCase struct {
	taskRepo ports.TaskRepository
}

func NewTaskUseCase(taskRepo ports.TaskRepository) *TaskUseCase {
	return &TaskUseCase{taskRepo: taskRepo}
}

func (uc *TaskUseCase) GetTasksByProjectAndAssignee(ctx context.Context, projectID int, assigneeID int) (projectv1.TasksResponse, error) {
	tasks, err := uc.taskRepo.GetTasksByProjectAndAssignee(ctx, projectID, assigneeID)
	if err != nil {
		return projectv1.TasksResponse{}, err
	}

	return projectv1.TasksResponse{Tasks: tasksToDTO(tasks)}, nil
}

func (uc *TaskUseCase) GetAllTasksByProject(ctx context.Context, projectID int) (projectv1.TasksResponse, error) {
	tasks, err := uc.taskRepo.GetAllTasksByProject(ctx, projectID)
	if err != nil {
		return projectv1.TasksResponse{}, err
	}

	return projectv1.TasksResponse{Tasks: tasksToDTO(tasks)}, nil
}

func (uc *TaskUseCase) GetClosedTasksByProject(ctx context.Context, projectID int) (projectv1.TasksResponse, error) {
	tasks, err := uc.taskRepo.GetClosedTasksByProject(ctx, projectID)
	if err != nil {
		return projectv1.TasksResponse{}, err
	}

	return projectv1.TasksResponse{Tasks: tasksToDTO(tasks)}, nil
}

func (uc *TaskUseCase) UpdateTaskStatus(ctx context.Context, taskID int, status domain.TaskStatus) error {
	return uc.taskRepo.UpdateTaskStatus(ctx, taskID, status)
}

func (uc *TaskUseCase) CreateTask(ctx context.Context, task domain.Task) error {
	return uc.taskRepo.CreateTask(ctx, task)
}

func (uc *TaskUseCase) AssignTask(ctx context.Context, taskID int, assigneeID int) error {
	return uc.taskRepo.AssignTask(ctx, taskID, assigneeID)
}

func (uc *TaskUseCase) UnassignTask(ctx context.Context, taskID int) error {
	return uc.taskRepo.UnassignTask(ctx, taskID)
}

func (uc *TaskUseCase) DeleteTask(ctx context.Context, taskID int) error {
	return uc.taskRepo.DeleteTask(ctx, taskID)
}

func (uc *TaskUseCase) CloseTask(ctx context.Context, taskID int) error {
	return uc.taskRepo.CloseTask(ctx, taskID)
}

func (uc *TaskUseCase) UpdateTask(ctx context.Context, task domain.Task) error {
	return uc.taskRepo.UpdateTask(ctx, task)
}

func (uc *TaskUseCase) GetTaskByID(ctx context.Context, taskID int) (*domain.Task, error) {
	return uc.taskRepo.GetTaskByID(ctx, taskID)
}

func (uc *TaskUseCase) GetTaskForAdmin(ctx context.Context, taskID *int32, projectID *int32, name *string) (*projectv1.Task, error) {
	var (
		task *domain.Task
		err  error
	)

	switch {
	case taskID != nil && *taskID > 0:
		task, err = uc.taskRepo.GetTaskByID(ctx, int(*taskID))
	case projectID != nil && *projectID > 0 && name != nil && *name != "":
		task, err = uc.taskRepo.GetTaskByProjectAndName(ctx, *projectID, *name)
	default:
		return nil, errors.New("task_id or project_id with name is required")
	}
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, errors.New("task not found")
	}

	var startDate, endDate *string
	if task.StartDate != nil {
		s := task.StartDate.Format("2006-01-02")
		startDate = &s
	}
	if task.EndDate != nil {
		s := task.EndDate.Format("2006-01-02")
		endDate = &s
	}

	return &projectv1.Task{
		Id:          task.ID,
		ProjectId:   task.ProjectID,
		AssigneeId:  task.AssigneeID,
		Name:        task.Name,
		Description: &task.Description,
		Priority:    projectv1.TaskPriority(task.Priority),
		Difficulty:  projectv1.TaskDifficulty(task.Difficulty),
		Status:      projectv1.TaskStatus(task.Status),
		StartDate:   startDate,
		EndDate:     endDate,
	}, nil
}

func (uc *TaskUseCase) UpdateTaskForAdmin(ctx context.Context, task *projectv1.Task) error {
	if task == nil || task.Id == 0 {
		return errors.New("task id is required")
	}

	var startDate *time.Time
	if task.StartDate != nil && *task.StartDate != "" {
		parsed, err := time.Parse("2006-01-02", *task.StartDate)
		if err != nil {
			return err
		}
		startDate = &parsed
	}

	var endDate *time.Time
	if task.EndDate != nil && *task.EndDate != "" {
		parsed, err := time.Parse("2006-01-02", *task.EndDate)
		if err != nil {
			return err
		}
		endDate = &parsed
	}

	description := ""
	if task.Description != nil {
		description = *task.Description
	}

	return uc.taskRepo.UpdateTask(ctx, domain.Task{
		ID:          task.Id,
		ProjectID:   task.ProjectId,
		AssigneeID:  task.AssigneeId,
		Name:        task.Name,
		Description: description,
		Priority:    domain.TaskPriority(task.Priority),
		Difficulty:  domain.TaskDifficulty(task.Difficulty),
		Status:      domain.TaskStatus(task.Status),
		StartDate:   startDate,
		EndDate:     endDate,
	})
}

func (uc *TaskUseCase) DeleteTaskForAdmin(ctx context.Context, taskID int32) error {
	if taskID == 0 {
		return errors.New("task id is required")
	}
	return uc.taskRepo.DeleteTask(ctx, int(taskID))
}
