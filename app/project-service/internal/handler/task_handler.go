package handler

import (
	"io"
	"net/http"
	"project-service/internal/controller"
	"project-service/internal/middleware"
	"project-service/internal/model"
	"project-service/internal/model/dto"
	"strconv"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
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

	resp, err := handler.taskController.GetTasksByProjectAndAssignee(request.Context(), projectID, int(userID))
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	bytes, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(&resp)
	if err != nil {
		http.Error(writer, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.Write(bytes)
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

	resp, err := handler.taskController.GetAllTasksByProject(request.Context(), projectID)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	bytes, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(&resp)
	if err != nil {
		http.Error(writer, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.Write(bytes)
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

	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}
	defer request.Body.Close()

	var updateReq dto.UpdateTaskStatusRequest
	err = protojson.UnmarshalOptions{AllowPartial: true, DiscardUnknown: true}.Unmarshal(body, &updateReq)
	if err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	err = handler.taskController.UpdateTaskStatus(request.Context(), taskID, model.TaskStatus(updateReq.Status))
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.WriteHeader(http.StatusOK)
}

// creating task in pool for manager to assign later or for members to self-assign (only for manager)
func (handler *TaskHandler) CreateTask(writer http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.GetUserID(request.Context())
	if !ok {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}
	defer request.Body.Close()

	var createReq dto.CreateTaskRequest
	err = protojson.UnmarshalOptions{AllowPartial: true, DiscardUnknown: true}.Unmarshal(body, &createReq)
	if err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	// Check if user is manager
	isManager, err := handler.projectController.IsUserManager(request.Context(), userID, int(createReq.ProjectId))
	if err != nil || !isManager {
		http.Error(writer, "access denied", http.StatusForbidden)
		return
	}

	description := ""
	if createReq.Description != nil {
		description = *createReq.Description
	}

	task := model.Task{
		ProjectID:   createReq.ProjectId,
		AssigneeID:  nil,
		Name:        createReq.Name,
		Description: description,
		Priority:    model.TaskPriority(createReq.Priority),
		Difficulty:  model.TaskDifficulty(createReq.Difficulty),
		Status:      model.TaskStatusNotStarted,
		StartDate:   nil,
	}

	err = handler.taskController.CreateTask(request.Context(), task)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusCreated)
	writer.Write([]byte(`{"message":"created"}`))
}

// assign task to member (only for manager)
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

	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}
	defer request.Body.Close()

	var assignReq dto.AssignTaskRequest
	err = protojson.UnmarshalOptions{AllowPartial: true, DiscardUnknown: true}.Unmarshal(body, &assignReq)
	if err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	// get task to verify user is manager of the project
	task, err := handler.taskController.GetTaskByID(request.Context(), taskID)
	if err != nil {
		http.Error(writer, "task not found", http.StatusNotFound)
		return
	}

	// Check if user is manager of the project or assigning to themselves from pool
	isManager, err := handler.projectController.IsUserManager(request.Context(), userID, int(task.ProjectID))
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	isMember, err := handler.projectController.IsUserMember(request.Context(), userID, int(task.ProjectID))
	if err != nil || !isMember {
		http.Error(writer, "access denied", http.StatusForbidden)
		return
	}

	// Allow if manager or (assigning to self and task was unassigned)
	if !isManager && (int32(assignReq.AssigneeId) != userID || task.AssigneeID != nil) {
		http.Error(writer, "access denied", http.StatusForbidden)
		return
	}

	err = handler.taskController.AssignTask(request.Context(), taskID, int(assignReq.AssigneeId))
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.WriteHeader(http.StatusOK)
}

// removes task (available only for manager)
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
	isManager, err := handler.projectController.IsUserManager(request.Context(), userID, int(task.ProjectID))
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
