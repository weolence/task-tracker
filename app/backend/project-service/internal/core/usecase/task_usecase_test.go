package usecase_test

import (
	"context"
	"testing"

	"project-service/internal/core/domain"
	"project-service/internal/core/testhelper"
	"project-service/internal/core/usecase"
)

func TestGetTasksByProjectAndAssignee_Success(t *testing.T) {
	mockTaskRepo := testhelper.NewMockTaskRepository()
	uc := usecase.NewTaskUseCase(mockTaskRepo)

	projectID := int32(1)
	assigneeID := int32(5)

	_, _ = mockTaskRepo.CreateTask(context.Background(), domain.Task{
		ProjectID:  projectID,
		Name:       "Task 1",
		AssigneeID: &assigneeID,
		Status:     domain.TaskStatusNotStarted,
	})
	_, _ = mockTaskRepo.CreateTask(context.Background(), domain.Task{
		ProjectID:  projectID,
		Name:       "Task 2",
		AssigneeID: &assigneeID,
		Status:     domain.TaskStatusInWork,
	})

	response, err := uc.GetTasksByProjectAndAssignee(context.Background(), int(projectID), int(assigneeID))

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(response.Tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(response.Tasks))
	}
}

func TestUpdateTaskStatus_Success(t *testing.T) {
	mockTaskRepo := testhelper.NewMockTaskRepository()
	uc := usecase.NewTaskUseCase(mockTaskRepo)

	_, _ = mockTaskRepo.CreateTask(context.Background(), domain.Task{
		ProjectID: 1,
		Name:      "Test Task",
		Status:    domain.TaskStatusNotStarted,
	})
	taskID := mockTaskRepo.LastID

	err := uc.UpdateTaskStatus(context.Background(), taskID, domain.TaskStatusInWork)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	updatedTask, _ := mockTaskRepo.GetTaskByID(context.Background(), taskID)
	if updatedTask.Status != domain.TaskStatusInWork {
		t.Errorf("Expected status InWork, got %v", updatedTask.Status)
	}
}

func TestCreateTask_Success(t *testing.T) {
	mockTaskRepo := testhelper.NewMockTaskRepository()
	uc := usecase.NewTaskUseCase(mockTaskRepo)

	_, err := uc.CreateTask(context.Background(), domain.Task{
		ProjectID: 1,
		Name:      "New Task",
		Status:    domain.TaskStatusNotStarted,
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestAssignTask_Success(t *testing.T) {
	mockTaskRepo := testhelper.NewMockTaskRepository()
	uc := usecase.NewTaskUseCase(mockTaskRepo)

	_, _ = mockTaskRepo.CreateTask(context.Background(), domain.Task{
		ProjectID: 1,
		Name:      "Task",
		Status:    domain.TaskStatusNotStarted,
	})
	taskID := mockTaskRepo.LastID

	err := uc.AssignTask(context.Background(), taskID, 5)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	assignedTask, _ := mockTaskRepo.GetTaskByID(context.Background(), taskID)
	if assignedTask.AssigneeID == nil || *assignedTask.AssigneeID != 5 {
		t.Error("Expected assignee 5")
	}
}

func TestUnassignTask_Success(t *testing.T) {
	mockTaskRepo := testhelper.NewMockTaskRepository()
	uc := usecase.NewTaskUseCase(mockTaskRepo)

	assigneeID := int32(5)
	_, _ = mockTaskRepo.CreateTask(context.Background(), domain.Task{
		ProjectID:  1,
		Name:       "Task",
		Status:     domain.TaskStatusNotStarted,
		AssigneeID: &assigneeID,
	})
	taskID := mockTaskRepo.LastID

	err := uc.UnassignTask(context.Background(), taskID)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	unassignedTask, _ := mockTaskRepo.GetTaskByID(context.Background(), taskID)
	if unassignedTask.AssigneeID != nil {
		t.Errorf("Expected nil assignee, got %v", *unassignedTask.AssigneeID)
	}
}

func TestDeleteTask_Success(t *testing.T) {
	mockTaskRepo := testhelper.NewMockTaskRepository()
	uc := usecase.NewTaskUseCase(mockTaskRepo)

	_, _ = mockTaskRepo.CreateTask(context.Background(), domain.Task{
		ProjectID: 1,
		Name:      "Task to delete",
		Status:    domain.TaskStatusNotStarted,
	})
	taskID := mockTaskRepo.LastID

	err := uc.DeleteTask(context.Background(), taskID)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	_, err = mockTaskRepo.GetTaskByID(context.Background(), taskID)
	if err == nil {
		t.Error("Expected error for deleted task")
	}
}

func TestGetTaskByID_Success(t *testing.T) {
	mockTaskRepo := testhelper.NewMockTaskRepository()
	uc := usecase.NewTaskUseCase(mockTaskRepo)

	_, _ = mockTaskRepo.CreateTask(context.Background(), domain.Task{
		ProjectID: 1,
		Name:      "Test Task",
		Status:    domain.TaskStatusNotStarted,
	})
	taskID := mockTaskRepo.LastID

	retrievedTask, err := uc.GetTaskByID(context.Background(), taskID)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if retrievedTask.Name != "Test Task" {
		t.Errorf("Expected name 'Test Task', got '%s'", retrievedTask.Name)
	}
}

func TestGetTaskByID_NotFound(t *testing.T) {
	mockTaskRepo := testhelper.NewMockTaskRepository()
	uc := usecase.NewTaskUseCase(mockTaskRepo)

	_, err := uc.GetTaskByID(context.Background(), 999)

	if err == nil {
		t.Error("Expected error for non-existent task")
	}
}

func TestUpdateTask_Success(t *testing.T) {
	mockTaskRepo := testhelper.NewMockTaskRepository()
	uc := usecase.NewTaskUseCase(mockTaskRepo)

	_, _ = mockTaskRepo.CreateTask(context.Background(), domain.Task{
		ProjectID: 1,
		Name:      "Original Name",
		Status:    domain.TaskStatusNotStarted,
	})
	taskID := mockTaskRepo.LastID

	existing, _ := mockTaskRepo.GetTaskByID(context.Background(), taskID)
	updated := *existing
	updated.Name = "Updated Name"

	err := uc.UpdateTask(context.Background(), updated)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	result, _ := mockTaskRepo.GetTaskByID(context.Background(), taskID)
	if result.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got '%s'", result.Name)
	}
}

func TestCloseTask_Success(t *testing.T) {
	mockTaskRepo := testhelper.NewMockTaskRepository()
	uc := usecase.NewTaskUseCase(mockTaskRepo)

	_, _ = mockTaskRepo.CreateTask(context.Background(), domain.Task{
		ProjectID: 1,
		Name:      "Task",
		Status:    domain.TaskStatusInWork,
	})
	taskID := mockTaskRepo.LastID

	err := uc.CloseTask(context.Background(), taskID)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	closed, _ := mockTaskRepo.GetTaskByID(context.Background(), taskID)
	if closed.Status != domain.TaskStatusClosed {
		t.Errorf("Expected status Closed, got %v", closed.Status)
	}
}
