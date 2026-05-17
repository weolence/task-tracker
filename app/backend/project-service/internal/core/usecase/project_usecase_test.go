package usecase_test

import (
	"context"
	"testing"
	"time"

	"project-service/internal/core/domain"
	"project-service/internal/core/testhelper"
	"project-service/internal/core/usecase"
)

func TestGetDashboard(t *testing.T) {
	mockProjectRepo := testhelper.NewMockProjectRepository()
	mockTaskRepo := testhelper.NewMockTaskRepository()
	uc := usecase.NewProjectUseCase(mockProjectRepo, mockTaskRepo, "")

	userID := int32(1)
	_, _ = mockProjectRepo.CreateProject(context.Background(), domain.Project{
		ManagerID:   userID,
		Name:        "Project 1",
		Description: "Test project",
		Status:      domain.ProjectStatusInWork,
		StartDate:   time.Now(),
	})

	dashboard, err := uc.GetDashboard(context.Background(), userID)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(dashboard.OwnedProjects) != 1 {
		t.Errorf("Expected 1 owned project, got %d", len(dashboard.OwnedProjects))
	}
	if dashboard.OwnedProjects[0].Name != "Project 1" {
		t.Errorf("Expected project name 'Project 1', got '%s'", dashboard.OwnedProjects[0].Name)
	}
}

func TestCreateProject_Success(t *testing.T) {
	mockProjectRepo := testhelper.NewMockProjectRepository()
	mockTaskRepo := testhelper.NewMockTaskRepository()
	uc := usecase.NewProjectUseCase(mockProjectRepo, mockTaskRepo, "")

	userID := int32(1)
	projectID, err := uc.CreateProject(context.Background(), userID, domain.Project{
		Name:        "Test Project",
		Description: "A test project",
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if projectID == 0 {
		t.Error("Expected non-zero project ID")
	}

	retrieved, _ := mockProjectRepo.GetProjectByID(context.Background(), projectID)
	if retrieved.ManagerID != userID {
		t.Errorf("Expected manager ID %d, got %d", userID, retrieved.ManagerID)
	}
}

func TestCreateProject_MissingName(t *testing.T) {
	mockProjectRepo := testhelper.NewMockProjectRepository()
	mockTaskRepo := testhelper.NewMockTaskRepository()
	uc := usecase.NewProjectUseCase(mockProjectRepo, mockTaskRepo, "")

	_, err := uc.CreateProject(context.Background(), int32(1), domain.Project{
		Name:        "",
		Description: "A test project",
	})

	if err == nil {
		t.Error("Expected error for missing name, got nil")
	}
}

func TestCreateProject_MissingDescription(t *testing.T) {
	mockProjectRepo := testhelper.NewMockProjectRepository()
	mockTaskRepo := testhelper.NewMockTaskRepository()
	uc := usecase.NewProjectUseCase(mockProjectRepo, mockTaskRepo, "")

	_, err := uc.CreateProject(context.Background(), int32(1), domain.Project{
		Name:        "Test Project",
		Description: "",
	})

	if err == nil {
		t.Error("Expected error for missing description, got nil")
	}
}

func TestIsUserManager_Success(t *testing.T) {
	mockProjectRepo := testhelper.NewMockProjectRepository()
	mockTaskRepo := testhelper.NewMockTaskRepository()
	uc := usecase.NewProjectUseCase(mockProjectRepo, mockTaskRepo, "")

	userID := int32(1)
	projectID, _ := mockProjectRepo.CreateProject(context.Background(), domain.Project{
		ManagerID:   userID,
		Name:        "Test Project",
		Description: "Test",
		Status:      domain.ProjectStatusInWork,
		StartDate:   time.Now(),
	})

	isManager, err := uc.IsUserManager(context.Background(), userID, projectID)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !isManager {
		t.Error("Expected user to be manager")
	}
}

func TestIsUserManager_NotManager(t *testing.T) {
	mockProjectRepo := testhelper.NewMockProjectRepository()
	mockTaskRepo := testhelper.NewMockTaskRepository()
	uc := usecase.NewProjectUseCase(mockProjectRepo, mockTaskRepo, "")

	managerID := int32(1)
	memberID := int32(2)
	projectID, _ := mockProjectRepo.CreateProject(context.Background(), domain.Project{
		ManagerID:   managerID,
		Name:        "Test Project",
		Description: "Test",
		Status:      domain.ProjectStatusInWork,
		StartDate:   time.Now(),
	})

	isManager, err := uc.IsUserManager(context.Background(), memberID, projectID)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if isManager {
		t.Error("Expected user not to be manager")
	}
}

func TestIsUserManager_ProjectNotFound(t *testing.T) {
	mockProjectRepo := testhelper.NewMockProjectRepository()
	mockTaskRepo := testhelper.NewMockTaskRepository()
	uc := usecase.NewProjectUseCase(mockProjectRepo, mockTaskRepo, "")

	_, err := uc.IsUserManager(context.Background(), int32(1), 999)

	if err == nil {
		t.Error("Expected error for non-existent project")
	}
}

func TestUpdateTaskStatus(t *testing.T) {
	mockProjectRepo := testhelper.NewMockProjectRepository()
	mockTaskRepo := testhelper.NewMockTaskRepository()
	uc := usecase.NewProjectUseCase(mockProjectRepo, mockTaskRepo, "")

	_, _ = mockTaskRepo.CreateTask(context.Background(), domain.Task{
		ProjectID: 1,
		Name:      "Test Task",
		Status:    domain.TaskStatusNotStarted,
	})
	taskID := mockTaskRepo.LastID

	err := uc.UpdateTaskStatus(context.Background(), taskID, domain.TaskStatusInWork, int32(1))

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	updatedTask, _ := mockTaskRepo.GetTaskByID(context.Background(), taskID)
	if updatedTask.Status != domain.TaskStatusInWork {
		t.Errorf("Expected task status InWork, got %v", updatedTask.Status)
	}
}

func TestRepositoryErrors(t *testing.T) {
	mockProjectRepo := testhelper.NewMockProjectRepository()
	mockTaskRepo := testhelper.NewMockTaskRepository()
	mockProjectRepo.GetOwnedErr = domain.ErrProjectNotFound

	uc := usecase.NewProjectUseCase(mockProjectRepo, mockTaskRepo, "")

	_, err := uc.GetDashboard(context.Background(), int32(1))

	if err == nil {
		t.Error("Expected error from repository, got nil")
	}
}
