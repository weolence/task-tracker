package handler

import (
	"encoding/json"
	"net/http"

	"project-service/internal/controller"
	"project-service/internal/middleware"
	"project-service/internal/model"
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

	writer.Header().Set("Content-Type", "application/json")
	json.NewEncoder(writer).Encode(resp)
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

	var payload createProjectRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	project := model.Project{
		Name:        payload.Name,
		Description: payload.Description,
	}

	projectID, err := handler.projectController.CreateProject(request.Context(), userID, project)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusCreated)
	json.NewEncoder(writer).Encode(createProjectResponse{ID: projectID})
}
