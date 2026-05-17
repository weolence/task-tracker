package http_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpadapter "project-service/internal/adapters/http"
	"project-service/internal/core/domain"
	"project-service/internal/core/testhelper"
	"project-service/internal/core/usecase"
	projectv1 "project-service/api/proto/projectv1"

	"google.golang.org/protobuf/encoding/protojson"
)

func newCommentTestSetup() (
	*httpadapter.CommentHandler,
	*testhelper.MockCommentRepository,
	*testhelper.MockTaskRepository,
	*testhelper.MockProjectRepository,
) {
	mockCommentRepo := testhelper.NewMockCommentRepository()
	mockTaskRepo := testhelper.NewMockTaskRepository()
	mockProjectRepo := testhelper.NewMockProjectRepository()

	commentUC := usecase.NewCommentUseCase(mockCommentRepo)
	taskUC := usecase.NewTaskUseCase(mockTaskRepo)
	projectUC := usecase.NewProjectUseCase(mockProjectRepo, mockTaskRepo, "")

	handler := httpadapter.NewCommentHandler(commentUC, taskUC, projectUC)
	return handler, mockCommentRepo, mockTaskRepo, mockProjectRepo
}

// seedTaskAndProject creates a project owned by userID and a task assigned to userID,
// returning the task's mock ID (always 1 for the first task).
func seedTaskAndProject(
	mockProjectRepo *testhelper.MockProjectRepository,
	mockTaskRepo *testhelper.MockTaskRepository,
	userID int32,
) int {
	projectID, _ := mockProjectRepo.CreateProject(context.Background(), domain.Project{
		ManagerID:   userID,
		Name:        "Test Project",
		Description: "Test",
		Status:      domain.ProjectStatusInWork,
		StartDate:   time.Now(),
	})

	assigneeID := userID
	_, _ = mockTaskRepo.CreateTask(context.Background(), domain.Task{
		ProjectID:  int32(projectID),
		Name:       "Test Task",
		Status:     domain.TaskStatusInWork,
		AssigneeID: &assigneeID,
	})
	return mockTaskRepo.LastID
}

func TestCommentHandler_CreateTaskComment_Unauthorized(t *testing.T) {
	handler, _, _, _ := newCommentTestSetup()

	body, _ := protojson.Marshal(&projectv1.CreateCommentRequest{Content: "Test"})
	r := httptest.NewRequest(http.MethodPost, "/api/tasks/1/comments", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.CreateTaskComment(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestCommentHandler_CreateTaskComment_TaskNotFound(t *testing.T) {
	handler, _, _, _ := newCommentTestSetup()

	body, _ := protojson.Marshal(&projectv1.CreateCommentRequest{Content: "Test"})
	r := httptest.NewRequest(http.MethodPost, "/api/tasks/1/comments", bytes.NewReader(body))
	r = r.WithContext(context.WithValue(r.Context(), httpadapter.UserIDKey, int32(1)))
	w := httptest.NewRecorder()

	handler.CreateTaskComment(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestCommentHandler_CreateTaskComment_Success(t *testing.T) {
	handler, _, mockTaskRepo, mockProjectRepo := newCommentTestSetup()

	userID := int32(1)
	seedTaskAndProject(mockProjectRepo, mockTaskRepo, userID)

	body, _ := protojson.Marshal(&projectv1.CreateCommentRequest{Content: "Hello!"})
	r := httptest.NewRequest(http.MethodPost, "/api/tasks/1/comments", bytes.NewReader(body))
	r = r.WithContext(context.WithValue(r.Context(), httpadapter.UserIDKey, userID))
	w := httptest.NewRecorder()

	handler.CreateTaskComment(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d (body: %s)", http.StatusCreated, w.Code, w.Body.String())
	}
}

func TestCommentHandler_GetTaskComments_Unauthorized(t *testing.T) {
	handler, _, _, _ := newCommentTestSetup()

	r := httptest.NewRequest(http.MethodGet, "/api/tasks/1/comments", nil)
	w := httptest.NewRecorder()

	handler.GetTaskComments(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestCommentHandler_GetTaskComments_Success(t *testing.T) {
	handler, _, mockTaskRepo, mockProjectRepo := newCommentTestSetup()

	userID := int32(1)
	seedTaskAndProject(mockProjectRepo, mockTaskRepo, userID)

	r := httptest.NewRequest(http.MethodGet, "/api/tasks/1/comments", nil)
	r = r.WithContext(context.WithValue(r.Context(), httpadapter.UserIDKey, userID))
	w := httptest.NewRecorder()

	handler.GetTaskComments(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d (body: %s)", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestCommentHandler_GetComment_MissingCriteria(t *testing.T) {
	handler, _, _, _ := newCommentTestSetup()

	body, _ := protojson.Marshal(&projectv1.GetCommentRequest{})
	r := httptest.NewRequest(http.MethodGet, "/admin/comments", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.GetComment(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCommentHandler_DeleteComment_NotFound(t *testing.T) {
	handler, _, _, _ := newCommentTestSetup()

	body, _ := protojson.Marshal(&projectv1.DeleteCommentRequest{CommentId: 999})
	r := httptest.NewRequest(http.MethodDelete, "/admin/comments", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.DeleteComment(w, r)

	if w.Code == http.StatusOK {
		t.Error("Expected error status for non-existent comment, got 200")
	}
}
