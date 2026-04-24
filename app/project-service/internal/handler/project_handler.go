package handler

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"project-service/internal/controller"
	"project-service/internal/middleware"
	"project-service/internal/model"
	"project-service/internal/model/dto"

	"google.golang.org/protobuf/encoding/protojson"
)

type ProjectHandler struct {
	projectController *controller.ProjectController
}

func NewProjectHandler(projectController *controller.ProjectController) *ProjectHandler {
	return &ProjectHandler{projectController: projectController}
}

func (handler *ProjectHandler) Dashboard(writer http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.GetUserID(request.Context())
	if !ok {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}

	resp, err := handler.projectController.GetDashboard(request.Context(), userID)
	if err != nil {
		http.Error(writer, "failed to load dashboard", http.StatusInternalServerError)
		return
	}

	bytes, err := protojson.MarshalOptions{EmitUnpopulated: true, UseProtoNames: true}.Marshal(&resp)
	if err != nil {
		http.Error(writer, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.Write(bytes)
}

func (handler *ProjectHandler) CreateProject(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

	var createReq dto.CreateProjectRequest
	err = protojson.UnmarshalOptions{AllowPartial: true, DiscardUnknown: true}.Unmarshal(body, &createReq)
	if err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	project := model.Project{
		Name:        createReq.Name,
		Description: createReq.Description,
	}

	projectID, err := handler.projectController.CreateProject(request.Context(), userID, project)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	resp := dto.CreateProjectResponse{
		Id: int32(projectID),
	}

	bytes, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(&resp)
	if err != nil {
		http.Error(writer, "unable to create project", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusCreated)
	writer.Write(bytes)
}

func (handler *ProjectHandler) ProjectTasks(writer http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.GetUserID(request.Context())
	if !ok {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}

	path := strings.TrimPrefix(request.URL.Path, "/api/projects/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 || parts[0] == "" {
		http.Error(writer, "invalid path", http.StatusBadRequest)
		return
	}

	projectID, err := strconv.Atoi(parts[0])
	if err != nil {
		http.Error(writer, "invalid project id", http.StatusBadRequest)
		return
	}

	if len(parts) > 1 && parts[1] == "tasks" {
		if request.Method == http.MethodGet {
			handler.GetProjectTasks(writer, request, projectID, userID)
		} else if request.Method == http.MethodPut && len(parts) > 2 {
			taskID, err := strconv.Atoi(parts[2])
			if err != nil {
				http.Error(writer, "invalid task id", http.StatusBadRequest)
				return
			}
			handler.UpdateTaskStatus(writer, request, taskID, userID)
		} else {
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		}
	} else {
		http.Error(writer, "not found", http.StatusNotFound)
	}
}

func (handler *ProjectHandler) GetProjectTasks(writer http.ResponseWriter, request *http.Request, projectID int, userID int32) {
	resp, err := handler.projectController.GetProjectTasks(request.Context(), projectID, userID)
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

func (handler *ProjectHandler) UpdateTaskStatus(writer http.ResponseWriter, request *http.Request, taskID int, userID int32) {
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

	err = handler.projectController.UpdateTaskStatus(request.Context(), taskID, model.TaskStatus(updateReq.Status), userID)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (handler *ProjectHandler) GetUserID(writer http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.GetUserID(request.Context())
	if !ok {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}

	resp := dto.UserIdResponse{UserId: userID}
	bytes, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(&resp)
	if err != nil {
		http.Error(writer, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.Write(bytes)
}

func (handler *ProjectHandler) GetProjectMembers(writer http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.GetUserID(request.Context())
	if !ok {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}

	projectIDStr := request.URL.Query().Get("project_id")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		http.Error(writer, "invalid project id", http.StatusBadRequest)
		return
	}

	members, err := handler.projectController.GetProjectMembers(request.Context(), projectID, userID)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := dto.ProjectMemberIdsResponse{Members: members}
	bytes, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(&resp)
	if err != nil {
		http.Error(writer, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.Write(bytes)
}

func (handler *ProjectHandler) IsUserManager(writer http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.GetUserID(request.Context())
	if !ok {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}

	projectIDStr := request.URL.Query().Get("project_id")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		http.Error(writer, "invalid project id", http.StatusBadRequest)
		return
	}

	isManager, err := handler.projectController.IsUserManager(request.Context(), userID, projectID)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := dto.IsUserManagerResponse{IsManager: isManager}
	bytes, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(&resp)
	if err != nil {
		http.Error(writer, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.Write(bytes)
}

func (handler *ProjectHandler) GetProjectMembersWithDetails(writer http.ResponseWriter, request *http.Request) {
	userID, ok := middleware.GetUserID(request.Context())
	if !ok {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}

	projectIDStr := request.URL.Query().Get("project_id")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		http.Error(writer, "invalid project id", http.StatusBadRequest)
		return
	}

	members, err := handler.projectController.GetProjectMembersWithDetails(request.Context(), projectID, userID)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	bytes, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(&members)
	if err != nil {
		http.Error(writer, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.Write(bytes)
}

func (handler *ProjectHandler) GetUserProjects(writer http.ResponseWriter, request *http.Request) {
	userIDStr := request.URL.Query().Get("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(writer, "invalid user_id", http.StatusBadRequest)
		return
	}

	resp, err := handler.projectController.GetUserProjects(request.Context(), int32(userID))
	if err != nil {
		http.Error(writer, "failed to load user projects", http.StatusInternalServerError)
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

func (handler *ProjectHandler) GetProjectInfo(writer http.ResponseWriter, request *http.Request) {
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

	project, err := handler.projectController.GetProjectInfo(request.Context(), projectID, userID)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	bytes, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(project)
	if err != nil {
		http.Error(writer, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.Write(bytes)
}
