package usecase_test

import (
	"context"
	"testing"

	"project-service/internal/core/domain"
	"project-service/internal/core/testhelper"
	"project-service/internal/core/usecase"
	projectv1 "project-service/api/proto/projectv1"
)

func TestCommentUseCase_CreateComment_Success(t *testing.T) {
	mockCommentRepo := testhelper.NewMockCommentRepository()
	uc := usecase.NewCommentUseCase(mockCommentRepo)

	authorID := int32(1)
	taskID := int32(5)
	content := "This is a test comment"

	comment, err := uc.CreateComment(context.Background(), authorID, taskID, content)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if comment.AuthorId != authorID {
		t.Errorf("Expected author ID %d, got %d", authorID, comment.AuthorId)
	}
	if comment.TaskId != taskID {
		t.Errorf("Expected task ID %d, got %d", taskID, comment.TaskId)
	}
	if comment.Content != content {
		t.Errorf("Expected content '%s', got '%s'", content, comment.Content)
	}
}

func TestCommentUseCase_GetCommentsByTaskID_Success(t *testing.T) {
	mockCommentRepo := testhelper.NewMockCommentRepository()
	uc := usecase.NewCommentUseCase(mockCommentRepo)

	taskID := int32(5)
	_, _ = mockCommentRepo.CreateComment(context.Background(), domain.Comment{AuthorID: 1, TaskID: taskID, Content: "First"})
	_, _ = mockCommentRepo.CreateComment(context.Background(), domain.Comment{AuthorID: 2, TaskID: taskID, Content: "Second"})

	response, err := uc.GetCommentsByTaskID(context.Background(), int(taskID))

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(response.Comments) != 2 {
		t.Errorf("Expected 2 comments, got %d", len(response.Comments))
	}
}

func TestCommentUseCase_GetCommentsByTaskID_Empty(t *testing.T) {
	mockCommentRepo := testhelper.NewMockCommentRepository()
	uc := usecase.NewCommentUseCase(mockCommentRepo)

	response, err := uc.GetCommentsByTaskID(context.Background(), 999)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(response.Comments) != 0 {
		t.Errorf("Expected 0 comments, got %d", len(response.Comments))
	}
}

func TestCommentUseCase_GetCommentByID_Success(t *testing.T) {
	mockCommentRepo := testhelper.NewMockCommentRepository()
	uc := usecase.NewCommentUseCase(mockCommentRepo)

	commentID, _ := mockCommentRepo.CreateComment(context.Background(), domain.Comment{
		AuthorID: 1,
		TaskID:   5,
		Content:  "Test comment",
	})

	retrieved, err := uc.GetCommentByID(context.Background(), commentID)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if retrieved == nil {
		t.Fatal("Expected comment, got nil")
	}
	if retrieved.Content != "Test comment" {
		t.Errorf("Expected content 'Test comment', got '%s'", retrieved.Content)
	}
}

func TestCommentUseCase_UpdateComment_Success(t *testing.T) {
	mockCommentRepo := testhelper.NewMockCommentRepository()
	uc := usecase.NewCommentUseCase(mockCommentRepo)

	commentID, _ := mockCommentRepo.CreateComment(context.Background(), domain.Comment{
		AuthorID: 1,
		TaskID:   5,
		Content:  "Original content",
	})

	err := uc.UpdateComment(context.Background(), domain.Comment{
		ID:       commentID,
		AuthorID: 1,
		TaskID:   5,
		Content:  "Updated content",
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	retrieved, _ := mockCommentRepo.GetCommentByID(context.Background(), commentID)
	if retrieved == nil {
		t.Fatal("Expected comment after update, got nil")
	}
	if retrieved.Content != "Updated content" {
		t.Errorf("Expected content 'Updated content', got '%s'", retrieved.Content)
	}
}

func TestCommentUseCase_DeleteComment_Success(t *testing.T) {
	mockCommentRepo := testhelper.NewMockCommentRepository()
	uc := usecase.NewCommentUseCase(mockCommentRepo)

	commentID, _ := mockCommentRepo.CreateComment(context.Background(), domain.Comment{
		AuthorID: 1,
		TaskID:   5,
		Content:  "To delete",
	})

	err := uc.DeleteComment(context.Background(), int(commentID))

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	deleted, _ := mockCommentRepo.GetCommentByID(context.Background(), commentID)
	if deleted != nil {
		t.Error("Expected comment to be deleted")
	}
}

func TestCommentUseCase_FindComment_ByID(t *testing.T) {
	mockCommentRepo := testhelper.NewMockCommentRepository()
	uc := usecase.NewCommentUseCase(mockCommentRepo)

	commentID, _ := mockCommentRepo.CreateComment(context.Background(), domain.Comment{
		AuthorID: 1,
		TaskID:   5,
		Content:  "Test comment",
	})

	retrieved, err := uc.FindComment(context.Background(), &projectv1.GetCommentRequest{CommentId: &commentID})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if retrieved.Content != "Test comment" {
		t.Errorf("Expected content 'Test comment', got '%s'", retrieved.Content)
	}
}

func TestCommentUseCase_FindComment_NoParams(t *testing.T) {
	mockCommentRepo := testhelper.NewMockCommentRepository()
	uc := usecase.NewCommentUseCase(mockCommentRepo)

	_, err := uc.FindComment(context.Background(), &projectv1.GetCommentRequest{})

	if err == nil {
		t.Error("Expected error when no parameters provided")
	}
}
