package http_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	httpadapter "project-service/internal/adapters/http"
	"project-service/internal/core/domain"
	"project-service/internal/core/testhelper"
	"project-service/internal/core/usecase"
	projectv1 "project-service/api/proto/projectv1"

	"google.golang.org/protobuf/encoding/protojson"
)

func newProjectTestSetup() (*httpadapter.ProjectHandler, *testhelper.MockProjectRepository, *testhelper.MockTaskRepository) {
	mockProjectRepo := testhelper.NewMockProjectRepository()
	mockTaskRepo := testhelper.NewMockTaskRepository()
	uc := usecase.NewProjectUseCase(mockProjectRepo, mockTaskRepo, "")
	return httpadapter.NewProjectHandler(uc), mockProjectRepo, mockTaskRepo
}

func withUserID(r *http.Request, userID int32) *http.Request {
	ctx := context.WithValue(r.Context(), httpadapter.UserIDKey, userID)
	return r.WithContext(ctx)
}

func TestProjectHandler_CreateProject_Success(t *testing.T) {
	handler, _, _ := newProjectTestSetup()

	body, _ := protojson.Marshal(&projectv1.CreateProjectRequest{
		Name:        "Test Project",
		Description: "A test project",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader(body))
	r = withUserID(r, 1)
	w := httptest.NewRecorder()

	handler.CreateProject(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}
}

func TestProjectHandler_CreateProject_Unauthorized(t *testing.T) {
	handler, _, _ := newProjectTestSetup()

	body, _ := protojson.Marshal(&projectv1.CreateProjectRequest{Name: "T", Description: "D"})
	r := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.CreateProject(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestProjectHandler_CreateProject_MissingName(t *testing.T) {
	handler, _, _ := newProjectTestSetup()

	body, _ := protojson.Marshal(&projectv1.CreateProjectRequest{Name: "", Description: "D"})
	r := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader(body))
	r = withUserID(r, 1)
	w := httptest.NewRecorder()

	handler.CreateProject(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestProjectHandler_Dashboard_Success(t *testing.T) {
	handler, mockProjectRepo, _ := newProjectTestSetup()

	userID := int32(1)
	_, _ = mockProjectRepo.CreateProject(context.Background(), domain.Project{
		ManagerID:   userID,
		Name:        "Test Project",
		Description: "Test",
		Status:      domain.ProjectStatusInWork,
		StartDate:   time.Now(),
	})

	r := httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	handler.Dashboard(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestProjectHandler_Dashboard_Unauthorized(t *testing.T) {
	handler, _, _ := newProjectTestSetup()

	r := httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
	w := httptest.NewRecorder()

	handler.Dashboard(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestProjectHandler_GetUserID_Success(t *testing.T) {
	handler, _, _ := newProjectTestSetup()

	r := httptest.NewRequest(http.MethodGet, "/api/user-id", nil)
	r = withUserID(r, int32(42))
	w := httptest.NewRecorder()

	handler.GetUserID(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestProjectHandler_GetUserID_Unauthorized(t *testing.T) {
	handler, _, _ := newProjectTestSetup()

	r := httptest.NewRequest(http.MethodGet, "/api/user-id", nil)
	w := httptest.NewRecorder()

	handler.GetUserID(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestProjectHandler_IsUserManager_True(t *testing.T) {
	handler, mockProjectRepo, _ := newProjectTestSetup()

	userID := int32(1)
	projectID, _ := mockProjectRepo.CreateProject(context.Background(), domain.Project{
		ManagerID:   userID,
		Name:        "Test Project",
		Description: "Test",
		Status:      domain.ProjectStatusInWork,
		StartDate:   time.Now(),
	})

	r := httptest.NewRequest(http.MethodGet, "/?project_id="+strconv.Itoa(projectID), nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	handler.IsUserManager(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d (body: %s)", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestProjectHandler_GetProjectMembers_Success(t *testing.T) {
	handler, mockProjectRepo, _ := newProjectTestSetup()

	userID := int32(1)
	projectID, _ := mockProjectRepo.CreateProject(context.Background(), domain.Project{
		ManagerID:   userID,
		Name:        "Test Project",
		Description: "Test",
		Status:      domain.ProjectStatusInWork,
		StartDate:   time.Now(),
	})

	r := httptest.NewRequest(http.MethodGet, "/?project_id="+strconv.Itoa(projectID), nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	handler.GetProjectMembers(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d (body: %s)", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestProjectHandler_DeleteProject_NotManager(t *testing.T) {
	handler, mockProjectRepo, _ := newProjectTestSetup()

	managerID := int32(1)
	projectID, _ := mockProjectRepo.CreateProject(context.Background(), domain.Project{
		ManagerID:   managerID,
		Name:        "Test Project",
		Description: "Test",
		Status:      domain.ProjectStatusInWork,
		StartDate:   time.Now(),
	})

	nonManagerID := int32(2)
	r := httptest.NewRequest(http.MethodDelete, "/?project_id="+strconv.Itoa(projectID), nil)
	r = withUserID(r, nonManagerID)
	w := httptest.NewRecorder()

	handler.DeleteProject(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestProjectHandler_EndProject_Success(t *testing.T) {
	handler, mockProjectRepo, _ := newProjectTestSetup()

	userID := int32(1)
	projectID, _ := mockProjectRepo.CreateProject(context.Background(), domain.Project{
		ManagerID:   userID,
		Name:        "Test Project",
		Description: "Test",
		Status:      domain.ProjectStatusInWork,
		StartDate:   time.Now(),
	})

	r := httptest.NewRequest(http.MethodPut, "/?project_id="+strconv.Itoa(projectID), nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	handler.EndProject(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d (body: %s)", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestProjectHandler_ResumeProject_Success(t *testing.T) {
	handler, mockProjectRepo, _ := newProjectTestSetup()

	userID := int32(1)
	projectID, _ := mockProjectRepo.CreateProject(context.Background(), domain.Project{
		ManagerID:   userID,
		Name:        "Test Project",
		Description: "Test",
		Status:      domain.ProjectStatusEnded,
		StartDate:   time.Now(),
	})

	r := httptest.NewRequest(http.MethodPut, "/?project_id="+strconv.Itoa(projectID), nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	handler.ResumeProject(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d (body: %s)", http.StatusOK, w.Code, w.Body.String())
	}
}
