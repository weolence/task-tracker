package handler

import (
	"encoding/json"
	"net/http"
	"project-service/internal/controller"
	"project-service/internal/middleware"
	"project-service/internal/model"
	"strconv"
	"strings"
	"time"
)

type TaskHandler struct {
	taskController    *controller.TaskController
	projectController *controller.ProjectController
}

func NewTaskHandler(taskController *controller.TaskController, projectController *controller.ProjectController) *TaskHandler {
	return &TaskHandler{
		taskController:    taskController,
		projectController: projectController,
	}
}

// GetMyTasks - получить свои назначенные задачи в проекте
func (handler *TaskHandler) GetMyTasks(writer http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.GetUserID(request.Context())
	if !ok {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}

	projectIDStr := request.URL.Query().Get("project_id")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		http.Error(writer, "invalid project_id", http.StatusBadRequest)
		return
	}

	tasks, err := handler.taskController.GetTasksByProjectAndAssignee(request.Context(), projectID, userID)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	json.NewEncoder(writer).Encode(tasks)
}

// GetAllProjectTasks - получить все задачи проекта (только для менеджера)
func (handler *TaskHandler) GetAllProjectTasks(writer http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.GetUserID(request.Context())
	if !ok {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}

	projectIDStr := request.URL.Query().Get("project_id")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		http.Error(writer, "invalid project_id", http.StatusBadRequest)
		return
	}

	// Check if user is manager
	isManager, err := handler.projectController.IsUserManager(request.Context(), userID, projectID)
	if err != nil || !isManager {
		http.Error(writer, "access denied", http.StatusForbidden)
		return
	}

	tasks, err := handler.taskController.GetAllTasksByProject(request.Context(), projectID)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	json.NewEncoder(writer).Encode(tasks)
}

func (handler *TaskHandler) UpdateTaskStatus(writer http.ResponseWriter, request *http.Request) {
	path := request.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 5 || parts[4] != "status" {
		http.Error(writer, "invalid path", http.StatusBadRequest)
		return
	}

	taskID, err := strconv.Atoi(parts[3])
	if err != nil {
		http.Error(writer, "invalid task id", http.StatusBadRequest)
		return
	}

	var payload struct {
		Status model.TaskStatus `json:"status"`
	}
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	err = handler.taskController.UpdateTaskStatus(request.Context(), taskID, payload.Status)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.WriteHeader(http.StatusOK)
}

// CreateTask - создать задачу в пул (только для менеджера, без присвоения)
func (handler *TaskHandler) CreateTask(writer http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.GetUserID(request.Context())
	if !ok {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}

	var payload struct {
		ProjectID   int                  `json:"project_id"`
		Name        string               `json:"name"`
		Description string               `json:"description"`
		Priority    model.TaskPriority   `json:"priority"`
		Difficulty  model.TaskDifficulty `json:"difficulty"`
	}

	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	// Check if user is manager
	isManager, err := handler.projectController.IsUserManager(request.Context(), userID, payload.ProjectID)
	if err != nil || !isManager {
		http.Error(writer, "access denied", http.StatusForbidden)
		return
	}

	task := model.Task{
		ProjectID:   payload.ProjectID,
		AssigneeID:  nil,
		Name:        payload.Name,
		Description: payload.Description,
		Priority:    payload.Priority,
		Difficulty:  payload.Difficulty,
		Status:      model.TaskStatusNotStarted,
		StartDate:   time.Now(),
	}

	err = handler.taskController.CreateTask(request.Context(), task)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.WriteHeader(http.StatusCreated)
}

// AssignTask - присвоить задачу участнику (только для менеджера)
func (handler *TaskHandler) AssignTask(writer http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.GetUserID(request.Context())
	if !ok {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}

	path := request.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 5 || parts[4] != "assign" {
		http.Error(writer, "invalid path", http.StatusBadRequest)
		return
	}

	taskID, err := strconv.Atoi(parts[3])
	if err != nil {
		http.Error(writer, "invalid task id", http.StatusBadRequest)
		return
	}

	var payload struct {
		AssigneeID int `json:"assignee_id"`
	}
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	// Get task to verify user is manager of the project
	task, err := handler.taskController.GetTaskByID(request.Context(), taskID)
	if err != nil {
		http.Error(writer, "task not found", http.StatusNotFound)
		return
	}

	// Check if user is manager of the project or assigning to themselves from pool
	isManager, err := handler.projectController.IsUserManager(request.Context(), userID, task.ProjectID)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	isMember, err := handler.projectController.IsUserMember(request.Context(), userID, task.ProjectID)
	if err != nil || !isMember {
		http.Error(writer, "access denied", http.StatusForbidden)
		return
	}

	// Allow if manager or (assigning to self and task was unassigned)
	if !isManager && (payload.AssigneeID != userID || task.AssigneeID != nil) {
		http.Error(writer, "access denied", http.StatusForbidden)
		return
	}

	err = handler.taskController.AssignTask(request.Context(), taskID, payload.AssigneeID)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.WriteHeader(http.StatusOK)
}

// DeleteTask - удалить задачу (только для менеджера)
func (handler *TaskHandler) DeleteTask(writer http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.GetUserID(request.Context())
	if !ok {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}

	path := request.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.Error(writer, "invalid path", http.StatusBadRequest)
		return
	}

	taskID, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		http.Error(writer, "invalid task id", http.StatusBadRequest)
		return
	}

	// Get task to verify user is manager of the project
	task, err := handler.taskController.GetTaskByID(request.Context(), taskID)
	if err != nil {
		http.Error(writer, "task not found", http.StatusNotFound)
		return
	}

	// Check if user is manager of the project
	isManager, err := handler.projectController.IsUserManager(request.Context(), userID, task.ProjectID)
	if err != nil || !isManager {
		http.Error(writer, "access denied", http.StatusForbidden)
		return
	}

	err = handler.taskController.DeleteTask(request.Context(), taskID)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.WriteHeader(http.StatusOK)
}
