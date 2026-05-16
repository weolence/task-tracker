package usecase

import (
	"context"
	"errors"
	"time"

	"project-service/internal/core/domain"
	"project-service/internal/core/ports"
	projectv1 "project-service/api/proto/projectv1"
)

var (
	ErrCommentNotFound        = errors.New("comment not found")
	ErrMissingCommentCriteria = errors.New("comment_id, author_id or task_id is required")
)

type CommentUseCase struct {
	commentRepo ports.CommentRepository
}

func NewCommentUseCase(commentRepo ports.CommentRepository) *CommentUseCase {
	return &CommentUseCase{commentRepo: commentRepo}
}

func (uc *CommentUseCase) GetCommentsByTaskID(ctx context.Context, taskID int) (projectv1.CommentsResponse, error) {
	comments, err := uc.commentRepo.GetCommentsByTaskID(ctx, taskID)
	if err != nil {
		return projectv1.CommentsResponse{}, err
	}

	commentDTOs := make([]*projectv1.Comment, 0, len(comments))
	for _, comment := range comments {
		commentDTOs = append(commentDTOs, toCommentDTO(comment))
	}

	return projectv1.CommentsResponse{Comments: commentDTOs}, nil
}

func (uc *CommentUseCase) CreateComment(ctx context.Context, authorID int32, taskID int32, content string) (*projectv1.Comment, error) {
	createdAt := time.Now().UTC()
	comment := domain.Comment{
		AuthorID:     authorID,
		TaskID:       taskID,
		Content:      content,
		CreationDate: createdAt,
	}

	commentID, err := uc.commentRepo.CreateComment(ctx, comment)
	if err != nil {
		return nil, err
	}

	comment.ID = commentID
	return toCommentDTO(comment), nil
}

func (uc *CommentUseCase) GetCommentByID(ctx context.Context, commentID int32) (*projectv1.Comment, error) {
	comment, err := uc.commentRepo.GetCommentByID(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if comment == nil {
		return nil, nil
	}

	return toCommentDTO(*comment), nil
}

func (uc *CommentUseCase) FindComment(ctx context.Context, req *projectv1.GetCommentRequest) (*projectv1.Comment, error) {
	switch {
	case req.CommentId != nil && *req.CommentId > 0:
		comment, err := uc.commentRepo.GetCommentByID(ctx, *req.CommentId)
		if err != nil {
			return nil, err
		}
		if comment == nil {
			return nil, ErrCommentNotFound
		}
		return toCommentDTO(*comment), nil
	case req.AuthorId != nil && *req.AuthorId > 0:
		comments, err := uc.commentRepo.GetCommentsByAuthorID(ctx, int(*req.AuthorId))
		if err != nil {
			return nil, err
		}
		if len(comments) == 0 {
			return nil, ErrCommentNotFound
		}
		return toCommentDTO(comments[0]), nil
	case req.TaskId != nil && *req.TaskId > 0:
		comments, err := uc.commentRepo.GetCommentsByTaskID(ctx, int(*req.TaskId))
		if err != nil {
			return nil, err
		}
		if len(comments) == 0 {
			return nil, ErrCommentNotFound
		}
		return toCommentDTO(comments[0]), nil
	default:
		return nil, ErrMissingCommentCriteria
	}
}

func (uc *CommentUseCase) UpdateComment(ctx context.Context, comment domain.Comment) error {
	return uc.commentRepo.UpdateComment(ctx, comment)
}

func (uc *CommentUseCase) DeleteComment(ctx context.Context, commentID int) error {
	return uc.commentRepo.DeleteComment(ctx, commentID)
}

func toCommentDTO(comment domain.Comment) *projectv1.Comment {
	return &projectv1.Comment{
		Id:           comment.ID,
		AuthorId:     comment.AuthorID,
		TaskId:       comment.TaskID,
		Content:      comment.Content,
		CreationDate: comment.CreationDate.Format(time.RFC3339),
	}
}
