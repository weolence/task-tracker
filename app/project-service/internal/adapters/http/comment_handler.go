package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"project-service/internal/core/domain"
	"project-service/internal/core/usecase"
	projectv1 "project-service/api/proto/projectv1"
)

type CommentHandler struct {
	commentUseCase *usecase.CommentUseCase
	taskUseCase    *usecase.TaskUseCase
	projectUseCase *usecase.ProjectUseCase
}

func NewCommentHandler(commentUseCase *usecase.CommentUseCase, taskUseCase *usecase.TaskUseCase, projectUseCase *usecase.ProjectUseCase) *CommentHandler {
	return &CommentHandler{
		commentUseCase: commentUseCase,
		taskUseCase:    taskUseCase,
		projectUseCase: projectUseCase,
	}
}

func (h *CommentHandler) GetTaskComments(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	taskID, _, err := parseTaskCommentsPath(r.URL.Path)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	if _, _, err := h.authorizeCommentAccess(r.Context(), userID, taskID); err != nil {
		writeCommentAccessError(w, err)
		return
	}

	response, err := h.commentUseCase.GetCommentsByTaskID(r.Context(), taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeProtoJSON(w, http.StatusOK, &response)
}

func (h *CommentHandler) CreateTaskComment(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	taskID, _, err := parseTaskCommentsPath(r.URL.Path)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	if _, _, err := h.authorizeCommentAccess(r.Context(), userID, taskID); err != nil {
		writeCommentAccessError(w, err)
		return
	}

	var req projectv1.CreateCommentRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}

	comment, err := h.commentUseCase.CreateComment(r.Context(), userID, int32(taskID), content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeProtoJSON(w, http.StatusCreated, comment)
}

func (h *CommentHandler) UpdateTaskComment(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	taskID, commentID, err := parseTaskCommentsPath(r.URL.Path)
	if err != nil || commentID == nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	if _, _, err := h.authorizeCommentAccess(r.Context(), userID, taskID); err != nil {
		writeCommentAccessError(w, err)
		return
	}

	comment, err := h.commentUseCase.GetCommentByID(r.Context(), int32(*commentID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if comment == nil {
		http.Error(w, "comment not found", http.StatusNotFound)
		return
	}
	if int(comment.TaskId) != taskID {
		http.Error(w, "comment does not belong to task", http.StatusBadRequest)
		return
	}
	if comment.AuthorId != userID {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}

	var req projectv1.UpdateTaskCommentRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}

	creationDate, err := parseFlexibleTime(comment.CreationDate)
	if err != nil {
		http.Error(w, "failed to read comment creation date", http.StatusInternalServerError)
		return
	}

	if err := h.commentUseCase.UpdateComment(r.Context(), domain.Comment{
		ID:           comment.Id,
		AuthorID:     comment.AuthorId,
		TaskID:       comment.TaskId,
		Content:      content,
		CreationDate: creationDate,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeProtoJSON(w, http.StatusOK, &projectv1.OperationResponse{Message: "comment updated"})
}

func (h *CommentHandler) DeleteTaskComment(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	taskID, commentID, err := parseTaskCommentsPath(r.URL.Path)
	if err != nil || commentID == nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	_, isManager, err := h.authorizeCommentAccess(r.Context(), userID, taskID)
	if err != nil {
		writeCommentAccessError(w, err)
		return
	}

	comment, err := h.commentUseCase.GetCommentByID(r.Context(), int32(*commentID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if comment == nil {
		http.Error(w, "comment not found", http.StatusNotFound)
		return
	}
	if int(comment.TaskId) != taskID {
		http.Error(w, "comment does not belong to task", http.StatusBadRequest)
		return
	}
	if !isManager && comment.AuthorId != userID {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}

	if err := h.commentUseCase.DeleteComment(r.Context(), *commentID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeProtoJSON(w, http.StatusOK, &projectv1.OperationResponse{Message: "comment deleted"})
}

func (h *CommentHandler) GetComment(w http.ResponseWriter, r *http.Request) {
	var req projectv1.GetCommentRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	comment, err := h.commentUseCase.FindComment(r.Context(), &req)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, usecase.ErrCommentNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, usecase.ErrMissingCommentCriteria) {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}

	writeProtoJSON(w, http.StatusOK, comment)
}

func (h *CommentHandler) UpdateComment(w http.ResponseWriter, r *http.Request) {
	var req projectv1.UpdateCommentRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if req.Comment == nil || req.Comment.Id == 0 || strings.TrimSpace(req.Comment.Content) == "" {
		http.Error(w, "comment id and content are required", http.StatusBadRequest)
		return
	}

	creationDate, err := parseFlexibleTime(req.Comment.CreationDate)
	if err != nil {
		http.Error(w, "invalid creation_date", http.StatusBadRequest)
		return
	}

	if err := h.commentUseCase.UpdateComment(r.Context(), domain.Comment{
		ID:           req.Comment.Id,
		AuthorID:     req.Comment.AuthorId,
		TaskID:       req.Comment.TaskId,
		Content:      strings.TrimSpace(req.Comment.Content),
		CreationDate: creationDate,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeProtoJSON(w, http.StatusOK, &projectv1.OperationResponse{Message: "comment updated"})
}

func (h *CommentHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	var req projectv1.DeleteCommentRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := h.commentUseCase.DeleteComment(r.Context(), int(req.CommentId)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeProtoJSON(w, http.StatusOK, &projectv1.OperationResponse{Message: "comment deleted"})
}

func (h *CommentHandler) authorizeCommentAccess(ctx context.Context, userID int32, taskID int) (*domain.Task, bool, error) {
	task, err := h.taskUseCase.GetTaskByID(ctx, taskID)
	if err != nil {
		return nil, false, err
	}
	if task == nil {
		return nil, false, errors.New("task not found")
	}

	isManager, err := h.projectUseCase.IsUserManager(ctx, userID, int(task.ProjectID))
	if err != nil {
		return nil, false, err
	}
	if isManager {
		return task, true, nil
	}

	if task.AssigneeID == nil || *task.AssigneeID != userID {
		return nil, false, errors.New("access denied")
	}

	return task, false, nil
}

func parseTaskCommentsPath(path string) (int, *int, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 4 || parts[0] != "api" || parts[1] != "tasks" || parts[3] != "comments" {
		return 0, nil, errors.New("invalid path")
	}

	taskID, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, nil, err
	}

	if len(parts) == 4 {
		return taskID, nil, nil
	}
	if len(parts) != 5 {
		return 0, nil, errors.New("invalid path")
	}

	commentID, err := strconv.Atoi(parts[4])
	if err != nil {
		return 0, nil, err
	}

	return taskID, &commentID, nil
}

func writeCommentAccessError(w http.ResponseWriter, err error) {
	switch err.Error() {
	case "task not found":
		http.Error(w, err.Error(), http.StatusNotFound)
	case "access denied":
		http.Error(w, err.Error(), http.StatusForbidden)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func parseFlexibleTime(value string) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, errors.New("invalid time format")
}
