package ports

import (
	"context"

	"project-service/internal/core/domain"
)

type CommentRepository interface {
	CreateComment(ctx context.Context, comment domain.Comment) (int32, error)
	DeleteComment(ctx context.Context, commentID int) error
	GetCommentByID(ctx context.Context, commentID int32) (*domain.Comment, error)
	GetCommentsByAuthorID(ctx context.Context, authorID int) ([]domain.Comment, error)
	GetCommentsByTaskID(ctx context.Context, taskID int) ([]domain.Comment, error)
	UpdateComment(ctx context.Context, comment domain.Comment) error
}
