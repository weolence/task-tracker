package http

import (
	"net/http"

	"project-service/internal/core/usecase"
	projectv1 "project-service/api/proto/projectv1"
)

type AdminHandler struct {
	projectUseCase *usecase.ProjectUseCase
	taskUseCase    *usecase.TaskUseCase
}

func NewAdminHandler(projectUseCase *usecase.ProjectUseCase, taskUseCase *usecase.TaskUseCase) *AdminHandler {
	return &AdminHandler{
		projectUseCase: projectUseCase,
		taskUseCase:    taskUseCase,
	}
}

func (h *AdminHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	var req projectv1.GetProjectRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	project, err := h.projectUseCase.GetProjectForAdmin(r.Context(), req.ProjectId, req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	writeProtoJSON(w, http.StatusOK, project)
}

func (h *AdminHandler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	var req projectv1.UpdateProjectRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := h.projectUseCase.UpdateProjectForAdmin(r.Context(), req.Project); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeProtoJSON(w, http.StatusOK, &projectv1.OperationResponse{Message: "project updated"})
}

func (h *AdminHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	var req projectv1.DeleteProjectRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := h.projectUseCase.DeleteProjectForAdmin(r.Context(), req.ProjectId); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeProtoJSON(w, http.StatusOK, &projectv1.OperationResponse{Message: "project deleted"})
}

func (h *AdminHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	var req projectv1.GetTaskRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	task, err := h.taskUseCase.GetTaskForAdmin(r.Context(), req.TaskId, req.ProjectId, req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	writeProtoJSON(w, http.StatusOK, task)
}

func (h *AdminHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	var req projectv1.UpdateTaskRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := h.taskUseCase.UpdateTaskForAdmin(r.Context(), req.Task); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeProtoJSON(w, http.StatusOK, &projectv1.OperationResponse{Message: "task updated"})
}

func (h *AdminHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	var req projectv1.DeleteTaskRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := h.taskUseCase.DeleteTaskForAdmin(r.Context(), req.TaskId); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeProtoJSON(w, http.StatusOK, &projectv1.OperationResponse{Message: "task deleted"})
}
