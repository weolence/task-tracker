package controller

import (
	"context"
	"project-service/internal/model"
	"project-service/internal/model/dto"
	"project-service/internal/repository"
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

func (controller *TaskController) DeleteTask(ctx context.Context, taskID int) error {
	return controller.taskRepository.DeleteTask(ctx, taskID)
}

func (controller *TaskController) UpdateTask(ctx context.Context, task model.Task) error {
	return controller.taskRepository.UpdateTask(ctx, task)
}

func (controller *TaskController) GetTaskByID(ctx context.Context, taskID int) (*model.Task, error) {
	return controller.taskRepository.GetTaskByID(ctx, taskID)
}
