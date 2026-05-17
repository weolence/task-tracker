package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"project-service/internal/core/domain"
	"project-service/internal/core/usecase"
	projectv1 "project-service/api/proto/projectv1"

	"google.golang.org/protobuf/encoding/protojson"
)

type ProjectHandler struct {
	projectUseCase *usecase.ProjectUseCase
}

func NewProjectHandler(projectUseCase *usecase.ProjectUseCase) *ProjectHandler {
	return &ProjectHandler{projectUseCase: projectUseCase}
}

func (h *ProjectHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	resp, err := h.projectUseCase.GetDashboard(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to load dashboard", http.StatusInternalServerError)
		return
	}

	b, err := protojson.MarshalOptions{EmitUnpopulated: true, UseProtoNames: true}.Marshal(&resp)
	if err != nil {
		http.Error(w, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var createReq projectv1.CreateProjectRequest
	if err := readProtoJSON(r, &createReq); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	project := domain.Project{
		Name:        createReq.Name,
		Description: createReq.Description,
	}

	projectID, err := h.projectUseCase.CreateProject(r.Context(), userID, project)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp := projectv1.CreateProjectResponse{Id: int32(projectID)}
	writeProtoJSON(w, http.StatusCreated, &resp)
}

func (h *ProjectHandler) ProjectTasks(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 || parts[0] == "" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	projectID, err := strconv.Atoi(parts[0])
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	if len(parts) > 1 && parts[1] == "tasks" {
		if r.Method == http.MethodGet {
			h.getProjectTasks(w, r, projectID, userID)
		} else if r.Method == http.MethodPut && len(parts) > 2 {
			taskID, err := strconv.Atoi(parts[2])
			if err != nil {
				http.Error(w, "invalid task id", http.StatusBadRequest)
				return
			}
			h.updateTaskStatus(w, r, taskID, userID)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	} else {
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (h *ProjectHandler) getProjectTasks(w http.ResponseWriter, r *http.Request, projectID int, userID int32) {
	resp, err := h.projectUseCase.GetProjectTasks(r.Context(), projectID, userID)
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

func (h *ProjectHandler) updateTaskStatus(w http.ResponseWriter, r *http.Request, taskID int, userID int32) {
	var updateReq projectv1.UpdateTaskStatusRequest
	if err := readProtoJSON(r, &updateReq); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := h.projectUseCase.UpdateTaskStatus(r.Context(), taskID, domain.TaskStatus(updateReq.Status), userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ProjectHandler) GetUserID(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	resp := projectv1.UserIdResponse{UserId: userID}
	writeProtoJSON(w, http.StatusOK, &resp)
}

func (h *ProjectHandler) GetProjectMembers(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	projectIDStr := r.URL.Query().Get("project_id")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	members, err := h.projectUseCase.GetProjectMembers(r.Context(), projectID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := projectv1.ProjectMemberIdsResponse{Members: members}
	writeProtoJSON(w, http.StatusOK, &resp)
}

func (h *ProjectHandler) IsUserManager(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	projectIDStr := r.URL.Query().Get("project_id")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	isManager, err := h.projectUseCase.IsUserManager(r.Context(), userID, projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := projectv1.IsUserManagerResponse{IsManager: isManager}
	writeProtoJSON(w, http.StatusOK, &resp)
}

func (h *ProjectHandler) GetProjectMembersWithDetails(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	projectIDStr := r.URL.Query().Get("project_id")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	members, err := h.projectUseCase.GetProjectMembersWithDetails(r.Context(), projectID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeProtoJSON(w, http.StatusOK, &members)
}

func (h *ProjectHandler) AddProjectMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	projectIDStr := r.URL.Query().Get("project_id")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	member, err := h.projectUseCase.AddProjectMember(r.Context(), projectID, userID, req.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeProtoJSON(w, http.StatusCreated, member)
}

func (h *ProjectHandler) TransferProjectManager(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	projectIDStr := r.URL.Query().Get("project_id")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	var req struct {
		NewManagerID int32 `json:"new_manager_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := h.projectUseCase.TransferProjectManager(r.Context(), projectID, userID, req.NewManagerID); err != nil {
		status := http.StatusBadRequest
		if err.Error() == "access denied" {
			status = http.StatusForbidden
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ProjectHandler) RemoveProjectMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

	var req struct {
		MemberID int32 `json:"member_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MemberID == 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := h.projectUseCase.RemoveProjectMember(r.Context(), projectID, userID, req.MemberID); err != nil {
		status := http.StatusBadRequest
		if err.Error() == "access denied" {
			status = http.StatusForbidden
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ProjectHandler) LeaveProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

	if err := h.projectUseCase.LeaveProject(r.Context(), projectID, userID); err != nil {
		status := http.StatusBadRequest
		if err.Error() == "manager cannot leave the project; transfer management first" {
			status = http.StatusForbidden
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ProjectHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

	if err := h.projectUseCase.DeleteProject(r.Context(), projectID, userID); err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "access denied" {
			status = http.StatusForbidden
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ProjectHandler) GetUserProjects(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	resp, err := h.projectUseCase.GetUserProjects(r.Context(), int32(userID))
	if err != nil {
		http.Error(w, "failed to load user projects", http.StatusInternalServerError)
		return
	}

	writeProtoJSON(w, http.StatusOK, &resp)
}

func (h *ProjectHandler) GetProjectInfo(w http.ResponseWriter, r *http.Request) {
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

	project, err := h.projectUseCase.GetProjectInfo(r.Context(), projectID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeProtoJSON(w, http.StatusOK, project)
}
