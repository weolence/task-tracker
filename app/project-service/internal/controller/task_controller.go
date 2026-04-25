package controller

import (
	"context"
	"errors"
	"project-service/internal/model"
	"project-service/internal/model/dto"
	"project-service/internal/repository"
	"time"
)

type TaskController struct {
	taskRepository repository.TaskRepository
}

func NewTaskController(taskRepository repository.TaskRepository) *TaskController {
	return &TaskController{taskRepository: taskRepository}
}

func (controller *TaskController) GetTasksByProjectAndAssignee(ctx context.Context, projectID int, assigneeID int) (dto.TasksResponse, error) {
	tasks, err := controller.taskRepository.GetTasksByProjectAndAssignee(ctx, projectID, assigneeID)
	if err != nil {
		return dto.TasksResponse{}, err
	}

	tasksDto := make([]*dto.Task, len(tasks))
	for i, t := range tasks {
		var startDate, endDate *string
		if t.StartDate != nil {
			s := (*t.StartDate).Format("2006-01-02")
			startDate = &s
		}
		if t.EndDate != nil {
			s := (*t.EndDate).Format("2006-01-02")
			endDate = &s
		}
		tasksDto[i] = &dto.Task{
			Id:          t.ID,
			ProjectId:   t.ProjectID,
			AssigneeId:  t.AssigneeID,
			Name:        t.Name,
			Description: &t.Description,
			Priority:    dto.TaskPriority(t.Priority),
			Difficulty:  dto.TaskDifficulty(t.Difficulty),
			Status:      dto.TaskStatus(t.Status),
			StartDate:   startDate,
			EndDate:     endDate,
		}
	}

	return dto.TasksResponse{Tasks: tasksDto}, nil
}

func (controller *TaskController) GetAllTasksByProject(ctx context.Context, projectID int) (dto.TasksResponse, error) {
	tasks, err := controller.taskRepository.GetAllTasksByProject(ctx, projectID)
	if err != nil {
		return dto.TasksResponse{}, err
	}

	tasksDto := make([]*dto.Task, len(tasks))
	for i, t := range tasks {
		var startDate, endDate *string
		if t.StartDate != nil {
			s := (*t.StartDate).Format("2006-01-02")
			startDate = &s
		}
		if t.EndDate != nil {
			s := (*t.EndDate).Format("2006-01-02")
			endDate = &s
		}
		tasksDto[i] = &dto.Task{
			Id:          t.ID,
			ProjectId:   t.ProjectID,
			AssigneeId:  t.AssigneeID,
			Name:        t.Name,
			Description: &t.Description,
			Priority:    dto.TaskPriority(t.Priority),
			Difficulty:  dto.TaskDifficulty(t.Difficulty),
			Status:      dto.TaskStatus(t.Status),
			StartDate:   startDate,
			EndDate:     endDate,
		}
	}

	return dto.TasksResponse{Tasks: tasksDto}, nil
}

func (controller *TaskController) GetClosedTasksByProject(ctx context.Context, projectID int) (dto.TasksResponse, error) {
	tasks, err := controller.taskRepository.GetClosedTasksByProject(ctx, projectID)
	if err != nil {
		return dto.TasksResponse{}, err
	}

	tasksDto := make([]*dto.Task, len(tasks))
	for i, t := range tasks {
		var startDate, endDate *string
		if t.StartDate != nil {
			s := (*t.StartDate).Format("2006-01-02")
			startDate = &s
		}
		if t.EndDate != nil {
			s := (*t.EndDate).Format("2006-01-02")
			endDate = &s
		}
		tasksDto[i] = &dto.Task{
			Id:          t.ID,
			ProjectId:   t.ProjectID,
			AssigneeId:  t.AssigneeID,
			Name:        t.Name,
			Description: &t.Description,
			Priority:    dto.TaskPriority(t.Priority),
			Difficulty:  dto.TaskDifficulty(t.Difficulty),
			Status:      dto.TaskStatus(t.Status),
			StartDate:   startDate,
			EndDate:     endDate,
		}
	}

	return dto.TasksResponse{Tasks: tasksDto}, nil
}

func (controller *TaskController) UpdateTaskStatus(ctx context.Context, taskID int, status model.TaskStatus) error {
	return controller.taskRepository.UpdateTaskStatus(ctx, taskID, status)
}

func (controller *TaskController) CreateTask(ctx context.Context, task model.Task) error {
	// Do not set StartDate on creation. It should be set when the task moves into work.
	return controller.taskRepository.CreateTask(ctx, task)
}

func (controller *TaskController) AssignTask(ctx context.Context, taskID int, assigneeID int) error {
	return controller.taskRepository.AssignTask(ctx, taskID, assigneeID)
}

func (controller *TaskController) UnassignTask(ctx context.Context, taskID int) error {
	return controller.taskRepository.UnassignTask(ctx, taskID)
}

func (controller *TaskController) DeleteTask(ctx context.Context, taskID int) error {
	return controller.taskRepository.DeleteTask(ctx, taskID)
}

func (controller *TaskController) CloseTask(ctx context.Context, taskID int) error {
	return controller.taskRepository.CloseTask(ctx, taskID)
}

func (controller *TaskController) UpdateTask(ctx context.Context, task model.Task) error {
	return controller.taskRepository.UpdateTask(ctx, task)
}

func (controller *TaskController) GetTaskByID(ctx context.Context, taskID int) (*model.Task, error) {
	return controller.taskRepository.GetTaskByID(ctx, taskID)
}

func (controller *TaskController) GetTaskForAdmin(ctx context.Context, taskID *int32, projectID *int32, name *string) (*dto.Task, error) {
	var (
		task *model.Task
		err  error
	)

	switch {
	case taskID != nil && *taskID > 0:
		task, err = controller.taskRepository.GetTaskByID(ctx, int(*taskID))
	case projectID != nil && *projectID > 0 && name != nil && *name != "":
		task, err = controller.taskRepository.GetTaskByProjectAndName(ctx, *projectID, *name)
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

	return &dto.Task{
		Id:          task.ID,
		ProjectId:   task.ProjectID,
		AssigneeId:  task.AssigneeID,
		Name:        task.Name,
		Description: &task.Description,
		Priority:    dto.TaskPriority(task.Priority),
		Difficulty:  dto.TaskDifficulty(task.Difficulty),
		Status:      dto.TaskStatus(task.Status),
		StartDate:   startDate,
		EndDate:     endDate,
	}, nil
}

func (controller *TaskController) UpdateTaskForAdmin(ctx context.Context, task *dto.Task) error {
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

	return controller.taskRepository.UpdateTask(ctx, model.Task{
		ID:          task.Id,
		ProjectID:   task.ProjectId,
		AssigneeID:  task.AssigneeId,
		Name:        task.Name,
		Description: description,
		Priority:    model.TaskPriority(task.Priority),
		Difficulty:  model.TaskDifficulty(task.Difficulty),
		Status:      model.TaskStatus(task.Status),
		StartDate:   startDate,
		EndDate:     endDate,
	})
}

func (controller *TaskController) DeleteTaskForAdmin(ctx context.Context, taskID int32) error {
	if taskID == 0 {
		return errors.New("task id is required")
	}
	return controller.taskRepository.DeleteTask(ctx, int(taskID))
}
