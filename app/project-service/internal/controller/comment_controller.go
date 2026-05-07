package controller

import (
	"context"
	"errors"
	"project-service/internal/model"
	"project-service/internal/model/dto"
	"project-service/internal/repository"
	"time"
)

var (
	ErrCommentNotFound        = errors.New("comment not found")
	ErrMissingCommentCriteria = errors.New("comment_id, author_id or task_id is required")
)

type CommentController struct {
	commentRepository repository.CommentRepository
}

func NewCommentController(commentRepository repository.CommentRepository) *CommentController {
	return &CommentController{commentRepository: commentRepository}
}

func (controller *CommentController) GetCommentsByTaskID(ctx context.Context, taskID int) (dto.CommentsResponse, error) {
	comments, err := controller.commentRepository.GetCommentsByTaskID(ctx, taskID)
	if err != nil {
		return dto.CommentsResponse{}, err
	}

	commentDTOs := make([]*dto.Comment, 0, len(comments))
	for _, comment := range comments {
		commentDTOs = append(commentDTOs, toCommentDTO(comment))
	}

	return dto.CommentsResponse{Comments: commentDTOs}, nil
}

func (controller *CommentController) CreateComment(ctx context.Context, authorID int32, taskID int32, content string) (*dto.Comment, error) {
	createdAt := time.Now().UTC()
	comment := model.Comment{
		AuthorID:     authorID,
		TaskID:       taskID,
		Content:      content,
		CreationDate: createdAt,
	}

	commentID, err := controller.commentRepository.CreateComment(ctx, comment)
	if err != nil {
		return nil, err
	}

	comment.ID = commentID
	return toCommentDTO(comment), nil
}

func (controller *CommentController) GetCommentByID(ctx context.Context, commentID int32) (*dto.Comment, error) {
	comment, err := controller.commentRepository.GetCommentByID(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if comment == nil {
		return nil, nil
	}

	return toCommentDTO(*comment), nil
}

func (controller *CommentController) FindComment(ctx context.Context, req *dto.GetCommentRequest) (*dto.Comment, error) {
	switch {
	case req.CommentId != nil && *req.CommentId > 0:
		comment, err := controller.commentRepository.GetCommentByID(ctx, *req.CommentId)
		if err != nil {
			return nil, err
		}
		if comment == nil {
			return nil, ErrCommentNotFound
		}
		return toCommentDTO(*comment), nil
	case req.AuthorId != nil && *req.AuthorId > 0:
		comments, err := controller.commentRepository.GetCommentsByAuthorID(ctx, int(*req.AuthorId))
		if err != nil {
			return nil, err
		}
		if len(comments) == 0 {
			return nil, ErrCommentNotFound
		}
		return toCommentDTO(comments[0]), nil
	case req.TaskId != nil && *req.TaskId > 0:
		comments, err := controller.commentRepository.GetCommentsByTaskID(ctx, int(*req.TaskId))
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

func (controller *CommentController) UpdateComment(ctx context.Context, comment model.Comment) error {
	return controller.commentRepository.UpdateComment(ctx, comment)
}

func (controller *CommentController) DeleteComment(ctx context.Context, commentID int) error {
	return controller.commentRepository.DeleteComment(ctx, commentID)
}

func toCommentDTO(comment model.Comment) *dto.Comment {
	return &dto.Comment{
		Id:           comment.ID,
		AuthorId:     comment.AuthorID,
		TaskId:       comment.TaskID,
		Content:      comment.Content,
		CreationDate: comment.CreationDate.Format(time.RFC3339),
	}
}
