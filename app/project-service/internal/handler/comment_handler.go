package handler

import (
	"context"
	"errors"
	"net/http"
	"project-service/internal/controller"
	"project-service/internal/middleware"
	"project-service/internal/model"
	"project-service/internal/model/dto"
	"strconv"
	"strings"
	"time"
)

type CommentHandler struct {
	commentController *controller.CommentController
	taskController    *controller.TaskController
	projectController *controller.ProjectController
}

func NewCommentHandler(commentController *controller.CommentController, taskController *controller.TaskController, projectController *controller.ProjectController) *CommentHandler {
	return &CommentHandler{
		commentController: commentController,
		taskController:    taskController,
		projectController: projectController,
	}
}

func (handler *CommentHandler) GetTaskComments(writer http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.GetUserID(request.Context())
	if !ok {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}

	taskID, _, err := parseTaskCommentsPath(request.URL.Path)
	if err != nil {
		http.Error(writer, "invalid path", http.StatusBadRequest)
		return
	}

	_, _, err = handler.authorizeCommentAccess(request.Context(), userID, taskID)
	if err != nil {
		writeCommentAccessError(writer, err)
		return
	}

	response, err := handler.commentController.GetCommentsByTaskID(request.Context(), taskID)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writeProtoJSON(writer, http.StatusOK, &response)
}

func (handler *CommentHandler) CreateTaskComment(writer http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.GetUserID(request.Context())
	if !ok {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}

	taskID, _, err := parseTaskCommentsPath(request.URL.Path)
	if err != nil {
		http.Error(writer, "invalid path", http.StatusBadRequest)
		return
	}

	_, _, err = handler.authorizeCommentAccess(request.Context(), userID, taskID)
	if err != nil {
		writeCommentAccessError(writer, err)
		return
	}

	var req dto.CreateCommentRequest
	if err := readProtoJSON(request, &req); err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		http.Error(writer, "content is required", http.StatusBadRequest)
		return
	}

	comment, err := handler.commentController.CreateComment(request.Context(), userID, int32(taskID), content)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writeProtoJSON(writer, http.StatusCreated, comment)
}

func (handler *CommentHandler) UpdateTaskComment(writer http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.GetUserID(request.Context())
	if !ok {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}

	taskID, commentID, err := parseTaskCommentsPath(request.URL.Path)
	if err != nil || commentID == nil {
		http.Error(writer, "invalid path", http.StatusBadRequest)
		return
	}

	_, _, err = handler.authorizeCommentAccess(request.Context(), userID, taskID)
	if err != nil {
		writeCommentAccessError(writer, err)
		return
	}

	comment, err := handler.commentController.GetCommentByID(request.Context(), int32(*commentID))
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	if comment == nil {
		http.Error(writer, "comment not found", http.StatusNotFound)
		return
	}
	if int(comment.TaskId) != taskID {
		http.Error(writer, "comment does not belong to task", http.StatusBadRequest)
		return
	}
	if comment.AuthorId != userID {
		http.Error(writer, "access denied", http.StatusForbidden)
		return
	}

	var req dto.UpdateTaskCommentRequest
	if err := readProtoJSON(request, &req); err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		http.Error(writer, "content is required", http.StatusBadRequest)
		return
	}

	creationDate, err := parseFlexibleTime(comment.CreationDate)
	if err != nil {
		http.Error(writer, "failed to read comment creation date", http.StatusInternalServerError)
		return
	}

	err = handler.commentController.UpdateComment(request.Context(), model.Comment{
		ID:           comment.Id,
		AuthorID:     comment.AuthorId,
		TaskID:       comment.TaskId,
		Content:      content,
		CreationDate: creationDate,
	})
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	writeProtoJSON(writer, http.StatusOK, &dto.OperationResponse{Message: "comment updated"})
}

func (handler *CommentHandler) DeleteTaskComment(writer http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.GetUserID(request.Context())
	if !ok {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}

	taskID, commentID, err := parseTaskCommentsPath(request.URL.Path)
	if err != nil || commentID == nil {
		http.Error(writer, "invalid path", http.StatusBadRequest)
		return
	}

	_, isManager, err := handler.authorizeCommentAccess(request.Context(), userID, taskID)
	if err != nil {
		writeCommentAccessError(writer, err)
		return
	}

	comment, err := handler.commentController.GetCommentByID(request.Context(), int32(*commentID))
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	if comment == nil {
		http.Error(writer, "comment not found", http.StatusNotFound)
		return
	}
	if int(comment.TaskId) != taskID {
		http.Error(writer, "comment does not belong to task", http.StatusBadRequest)
		return
	}
	if !isManager && comment.AuthorId != userID {
		http.Error(writer, "access denied", http.StatusForbidden)
		return
	}

	if err := handler.commentController.DeleteComment(request.Context(), *commentID); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	writeProtoJSON(writer, http.StatusOK, &dto.OperationResponse{Message: "comment deleted"})
}

func (handler *CommentHandler) GetComment(writer http.ResponseWriter, request *http.Request) {
	var req dto.GetCommentRequest
	if err := readProtoJSON(request, &req); err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	comment, err := handler.commentController.FindComment(request.Context(), &req)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, controller.ErrCommentNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, controller.ErrMissingCommentCriteria) {
			status = http.StatusBadRequest
		}
		http.Error(writer, err.Error(), status)
		return
	}

	writeProtoJSON(writer, http.StatusOK, comment)
}

func (handler *CommentHandler) UpdateComment(writer http.ResponseWriter, request *http.Request) {
	var req dto.UpdateCommentRequest
	if err := readProtoJSON(request, &req); err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	if req.Comment == nil || req.Comment.Id == 0 || strings.TrimSpace(req.Comment.Content) == "" {
		http.Error(writer, "comment id and content are required", http.StatusBadRequest)
		return
	}

	creationDate, err := parseFlexibleTime(req.Comment.CreationDate)
	if err != nil {
		http.Error(writer, "invalid creation_date", http.StatusBadRequest)
		return
	}

	err = handler.commentController.UpdateComment(request.Context(), model.Comment{
		ID:           req.Comment.Id,
		AuthorID:     req.Comment.AuthorId,
		TaskID:       req.Comment.TaskId,
		Content:      strings.TrimSpace(req.Comment.Content),
		CreationDate: creationDate,
	})
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	writeProtoJSON(writer, http.StatusOK, &dto.OperationResponse{Message: "comment updated"})
}

func (handler *CommentHandler) DeleteComment(writer http.ResponseWriter, request *http.Request) {
	var req dto.DeleteCommentRequest
	if err := readProtoJSON(request, &req); err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	if err := handler.commentController.DeleteComment(request.Context(), int(req.CommentId)); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	writeProtoJSON(writer, http.StatusOK, &dto.OperationResponse{Message: "comment deleted"})
}

func (handler *CommentHandler) authorizeCommentAccess(ctx context.Context, userID int32, taskID int) (*model.Task, bool, error) {
	task, err := handler.taskController.GetTaskByID(ctx, taskID)
	if err != nil {
		return nil, false, err
	}
	if task == nil {
		return nil, false, errors.New("task not found")
	}

	isManager, err := handler.projectController.IsUserManager(ctx, userID, int(task.ProjectID))
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

func writeCommentAccessError(writer http.ResponseWriter, err error) {
	switch err.Error() {
	case "task not found":
		http.Error(writer, err.Error(), http.StatusNotFound)
	case "access denied":
		http.Error(writer, err.Error(), http.StatusForbidden)
	default:
		http.Error(writer, err.Error(), http.StatusInternalServerError)
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
