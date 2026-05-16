package http

import (
	"net/http"
	"strconv"
	"strings"

	"project-service/internal/core/domain"
	"project-service/internal/core/usecase"
	projectv1 "project-service/api/proto/projectv1"

	"google.golang.org/protobuf/encoding/protojson"
)

type TaskHandler struct {
	taskUseCase    *usecase.TaskUseCase
	projectUseCase *usecase.ProjectUseCase
}

func NewTaskHandler(taskUseCase *usecase.TaskUseCase, projectUseCase *usecase.ProjectUseCase) *TaskHandler {
	return &TaskHandler{
		taskUseCase:    taskUseCase,
		projectUseCase: projectUseCase,
	}
}

func (h *TaskHandler) GetMyTasks(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	projectIDStr := r.URL.Query().Get("project_id")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		http.Error(w, "invalid project_id", http.StatusBadRequest)
		return
	}

	resp, err := h.taskUseCase.GetTasksByProjectAndAssignee(r.Context(), projectID, int(userID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(&resp)
	if err != nil {
		http.Error(w, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func (h *TaskHandler) GetAllProjectTasks(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	projectIDStr := r.URL.Query().Get("project_id")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		http.Error(w, "invalid project_id", http.StatusBadRequest)
		return
	}

	isManager, err := h.projectUseCase.IsUserManager(r.Context(), userID, projectID)
	if err != nil || !isManager {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}

	resp, err := h.taskUseCase.GetAllTasksByProject(r.Context(), projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(&resp)
	if err != nil {
		http.Error(w, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func (h *TaskHandler) GetClosedProjectTasks(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	projectIDStr := r.URL.Query().Get("project_id")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		http.Error(w, "invalid project_id", http.StatusBadRequest)
		return
	}

	isManager, err := h.projectUseCase.IsUserManager(r.Context(), userID, projectID)
	if err != nil || !isManager {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}

	resp, err := h.taskUseCase.GetClosedTasksByProject(r.Context(), projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(&resp)
	if err != nil {
		http.Error(w, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func (h *TaskHandler) UpdateTaskStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 || parts[4] != "status" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	taskID, err := strconv.Atoi(parts[3])
	if err != nil {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}

	var updateReq projectv1.UpdateTaskStatusRequest
	if err := readProtoJSON(r, &updateReq); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	task, err := h.taskUseCase.GetTaskByID(r.Context(), taskID)
	if err != nil || task == nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	isManager, err := h.projectUseCase.IsUserManager(r.Context(), userID, int(task.ProjectID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !isManager {
		if task.AssigneeID == nil || *task.AssigneeID != userID {
			http.Error(w, "access denied", http.StatusForbidden)
			return
		}
	}

	if domain.TaskStatus(updateReq.Status) == domain.TaskStatusClosed {
		http.Error(w, "closed status can only be set by manager action", http.StatusBadRequest)
		return
	}

	if err := h.taskUseCase.UpdateTaskStatus(r.Context(), taskID, domain.TaskStatus(updateReq.Status)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *TaskHandler) CloseTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 || parts[4] != "close" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	taskID, err := strconv.Atoi(parts[3])
	if err != nil {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}

	task, err := h.taskUseCase.GetTaskByID(r.Context(), taskID)
	if err != nil || task == nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	isManager, err := h.projectUseCase.IsUserManager(r.Context(), userID, int(task.ProjectID))
	if err != nil || !isManager {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}

	if task.Status != domain.TaskStatusOnReview {
		http.Error(w, "task is not ready for closing", http.StatusBadRequest)
		return
	}

	if err := h.taskUseCase.CloseTask(r.Context(), taskID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var createReq projectv1.CreateTaskRequest
	if err := readProtoJSON(r, &createReq); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	isManager, err := h.projectUseCase.IsUserManager(r.Context(), userID, int(createReq.ProjectId))
	if err != nil || !isManager {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}

	description := ""
	if createReq.Description != nil {
		description = *createReq.Description
	}

	task := domain.Task{
		ProjectID:   createReq.ProjectId,
		AssigneeID:  nil,
		Name:        createReq.Name,
		Description: description,
		Priority:    domain.TaskPriority(createReq.Priority),
		Difficulty:  domain.TaskDifficulty(createReq.Difficulty),
		Status:      domain.TaskStatusNotStarted,
		StartDate:   nil,
	}

	if err := h.taskUseCase.CreateTask(r.Context(), task); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"message":"created"}`))
}

func (h *TaskHandler) AssignTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 || parts[4] != "assign" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	taskID, err := strconv.Atoi(parts[3])
	if err != nil {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}

	var assignReq projectv1.AssignTaskRequest
	if err := readProtoJSON(r, &assignReq); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	task, err := h.taskUseCase.GetTaskByID(r.Context(), taskID)
	if err != nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	isManager, err := h.projectUseCase.IsUserManager(r.Context(), userID, int(task.ProjectID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	isMember, err := h.projectUseCase.IsUserMember(r.Context(), userID, int(task.ProjectID))
	if err != nil || !isMember {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}

	if !isManager && (int32(assignReq.AssigneeId) != userID || task.AssigneeID != nil) {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}

	if isManager {
		targetIsMember, err := h.projectUseCase.IsUserMember(r.Context(), assignReq.AssigneeId, int(task.ProjectID))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !targetIsMember {
			http.Error(w, "selected user is not a project member", http.StatusBadRequest)
			return
		}
	}

	if err := h.taskUseCase.AssignTask(r.Context(), taskID, int(assignReq.AssigneeId)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *TaskHandler) UnassignTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 || parts[4] != "unassign" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	taskID, err := strconv.Atoi(parts[3])
	if err != nil {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}

	task, err := h.taskUseCase.GetTaskByID(r.Context(), taskID)
	if err != nil || task == nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	isManager, err := h.projectUseCase.IsUserManager(r.Context(), userID, int(task.ProjectID))
	if err != nil || !isManager {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}

	if err := h.taskUseCase.UnassignTask(r.Context(), taskID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 2 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	taskID, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}

	task, err := h.taskUseCase.GetTaskByID(r.Context(), taskID)
	if err != nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	isManager, err := h.projectUseCase.IsUserManager(r.Context(), userID, int(task.ProjectID))
	if err != nil || !isManager {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}

	if err := h.taskUseCase.DeleteTask(r.Context(), taskID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
