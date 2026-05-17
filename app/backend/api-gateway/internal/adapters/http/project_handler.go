package httpadapter

import (
	"net/http"
	"strconv"
	"strings"

	projectsvcv1 "project-service/api/proto/projectsvcv1"
	projectv1 "project-service/api/proto/projectv1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ProjectHandler struct {
	project projectsvcv1.ProjectServiceClient
}

func NewProjectHandler(project projectsvcv1.ProjectServiceClient) *ProjectHandler {
	return &ProjectHandler{project: project}
}

func grpcStatus(w http.ResponseWriter, err error) {
	st, _ := status.FromError(err)
	switch st.Code() {
	case codes.NotFound:
		http.Error(w, st.Message(), http.StatusNotFound)
	case codes.PermissionDenied:
		http.Error(w, st.Message(), http.StatusForbidden)
	case codes.InvalidArgument, codes.FailedPrecondition:
		http.Error(w, st.Message(), http.StatusBadRequest)
	case codes.Unauthenticated:
		http.Error(w, st.Message(), http.StatusUnauthorized)
	default:
		http.Error(w, st.Message(), http.StatusInternalServerError)
	}
}

func queryInt(r *http.Request, key string) (int32, error) {
	v, err := strconv.Atoi(r.URL.Query().Get(key))
	return int32(v), err
}

// ─── Dashboard ───────────────────────────────────────────────────────────────

func (h *ProjectHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	resp, err := h.project.GetDashboard(outgoingCtx(r), &projectsvcv1.Empty{})
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

// ─── Projects ────────────────────────────────────────────────────────────────

func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req projectv1.CreateProjectRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	resp, err := h.project.CreateProject(outgoingCtx(r), &req)
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusCreated, resp)
}

func (h *ProjectHandler) GetProjectTasks(w http.ResponseWriter, r *http.Request) {
	// path: /api/projects/{id}/tasks
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/projects/"), "/")
	projectID, err := strconv.Atoi(parts[0])
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}
	resp, err := h.project.GetProjectTasks(outgoingCtx(r), &projectsvcv1.ProjectIdRequest{ProjectId: int32(projectID)})
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (h *ProjectHandler) GetProjectInfo(w http.ResponseWriter, r *http.Request) {
	pid, err := queryInt(r, "project_id")
	if err != nil {
		http.Error(w, "invalid project_id", http.StatusBadRequest)
		return
	}
	resp, err := h.project.GetProjectInfo(outgoingCtx(r), &projectsvcv1.ProjectIdRequest{ProjectId: pid})
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (h *ProjectHandler) GetProjectMembers(w http.ResponseWriter, r *http.Request) {
	pid, err := queryInt(r, "project_id")
	if err != nil {
		http.Error(w, "invalid project_id", http.StatusBadRequest)
		return
	}
	resp, err := h.project.GetProjectMembers(outgoingCtx(r), &projectsvcv1.ProjectIdRequest{ProjectId: pid})
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (h *ProjectHandler) GetProjectMembersDetails(w http.ResponseWriter, r *http.Request) {
	pid, err := queryInt(r, "project_id")
	if err != nil {
		http.Error(w, "invalid project_id", http.StatusBadRequest)
		return
	}
	resp, err := h.project.GetProjectMembersDetails(outgoingCtx(r), &projectsvcv1.ProjectIdRequest{ProjectId: pid})
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (h *ProjectHandler) AddProjectMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pid, err := queryInt(r, "project_id")
	if err != nil {
		http.Error(w, "invalid project_id", http.StatusBadRequest)
		return
	}
	var body struct {
		Email string `json:"email"`
	}
	if err := readJSON(r, &body); err != nil || body.Email == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	resp, err := h.project.AddProjectMember(outgoingCtx(r), &projectsvcv1.AddMemberRequest{
		ProjectId: pid,
		Email:     body.Email,
	})
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusCreated, resp)
}

func (h *ProjectHandler) TransferProjectManager(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pid, err := queryInt(r, "project_id")
	if err != nil {
		http.Error(w, "invalid project_id", http.StatusBadRequest)
		return
	}
	var body struct {
		NewManagerID int32 `json:"new_manager_id"`
	}
	if err := readJSON(r, &body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	resp, err := h.project.TransferProjectManager(outgoingCtx(r), &projectsvcv1.TransferManagerRequest{
		ProjectId:    pid,
		NewManagerId: body.NewManagerID,
	})
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (h *ProjectHandler) IsUserManager(w http.ResponseWriter, r *http.Request) {
	pid, err := queryInt(r, "project_id")
	if err != nil {
		http.Error(w, "invalid project_id", http.StatusBadRequest)
		return
	}
	resp, err := h.project.IsUserManager(outgoingCtx(r), &projectsvcv1.IsManagerRequest{ProjectId: pid})
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (h *ProjectHandler) GetUserProjects(w http.ResponseWriter, r *http.Request) {
	uid, err := queryInt(r, "user_id")
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}
	resp, err := h.project.GetUserProjects(outgoingCtx(r), &projectsvcv1.UserProjectsRequest{UserId: uid})
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (h *ProjectHandler) RemoveProjectMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pid, err := queryInt(r, "project_id")
	if err != nil {
		http.Error(w, "invalid project_id", http.StatusBadRequest)
		return
	}
	var body struct {
		MemberID int32 `json:"member_id"`
	}
	if err := readJSON(r, &body); err != nil || body.MemberID == 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	resp, err := h.project.RemoveProjectMember(outgoingCtx(r), &projectsvcv1.TransferManagerRequest{
		ProjectId:    pid,
		NewManagerId: body.MemberID,
	})
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (h *ProjectHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pid, err := queryInt(r, "project_id")
	if err != nil {
		http.Error(w, "invalid project_id", http.StatusBadRequest)
		return
	}
	resp, err := h.project.DeleteProject(outgoingCtx(r), &projectsvcv1.ProjectIdRequest{ProjectId: pid})
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (h *ProjectHandler) LeaveProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pid, err := queryInt(r, "project_id")
	if err != nil {
		http.Error(w, "invalid project_id", http.StatusBadRequest)
		return
	}
	resp, err := h.project.LeaveProject(outgoingCtx(r), &projectsvcv1.ProjectIdRequest{ProjectId: pid})
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

// ─── Tasks ───────────────────────────────────────────────────────────────────

func (h *ProjectHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req projectv1.CreateTaskRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	resp, err := h.project.CreateTask(outgoingCtx(r), &req)
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusCreated, resp)
}

func (h *ProjectHandler) GetMyTasks(w http.ResponseWriter, r *http.Request) {
	pid, err := queryInt(r, "project_id")
	if err != nil {
		http.Error(w, "invalid project_id", http.StatusBadRequest)
		return
	}
	resp, err := h.project.GetMyTasks(outgoingCtx(r), &projectsvcv1.ProjectIdRequest{ProjectId: pid})
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (h *ProjectHandler) GetAllProjectTasks(w http.ResponseWriter, r *http.Request) {
	pid, err := queryInt(r, "project_id")
	if err != nil {
		http.Error(w, "invalid project_id", http.StatusBadRequest)
		return
	}
	resp, err := h.project.GetAllProjectTasks(outgoingCtx(r), &projectsvcv1.ProjectIdRequest{ProjectId: pid})
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (h *ProjectHandler) GetClosedProjectTasks(w http.ResponseWriter, r *http.Request) {
	pid, err := queryInt(r, "project_id")
	if err != nil {
		http.Error(w, "invalid project_id", http.StatusBadRequest)
		return
	}
	resp, err := h.project.GetClosedProjectTasks(outgoingCtx(r), &projectsvcv1.ProjectIdRequest{ProjectId: pid})
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (h *ProjectHandler) TasksRouter(w http.ResponseWriter, r *http.Request) {
	// /api/tasks/{id}/{action}  or  /api/tasks  (POST)
	path := strings.TrimPrefix(r.URL.Path, "/api/tasks")
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		h.CreateTask(w, r)
		return
	}

	parts := strings.SplitN(path, "/", 2)
	taskID, err := strconv.Atoi(parts[0])
	if err != nil {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch action {
	case "status":
		h.updateTaskStatus(w, r, int32(taskID))
	case "close":
		h.closeTask(w, r, int32(taskID))
	case "assign":
		h.assignTask(w, r, int32(taskID))
	case "unassign":
		h.unassignTask(w, r, int32(taskID))
	case "comments", "comments/":
		h.commentsRouter(w, r, int32(taskID), strings.TrimPrefix(action, "comments"))
	default:
		if r.Method == http.MethodDelete && action == "" {
			h.deleteTask(w, r, int32(taskID))
		} else {
			http.NotFound(w, r)
		}
	}
}

func (h *ProjectHandler) updateTaskStatus(w http.ResponseWriter, r *http.Request, taskID int32) {
	var req projectv1.UpdateTaskStatusRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	resp, err := h.project.UpdateTaskStatus(outgoingCtx(r), &projectsvcv1.UpdateTaskStatusGrpcRequest{
		TaskId: taskID,
		Status: req.Status,
	})
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (h *ProjectHandler) closeTask(w http.ResponseWriter, r *http.Request, taskID int32) {
	resp, err := h.project.CloseTask(outgoingCtx(r), &projectsvcv1.TaskIdRequest{TaskId: taskID})
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (h *ProjectHandler) assignTask(w http.ResponseWriter, r *http.Request, taskID int32) {
	var req projectv1.AssignTaskRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	resp, err := h.project.AssignTask(outgoingCtx(r), &projectsvcv1.AssignTaskGrpcRequest{
		TaskId:     taskID,
		AssigneeId: req.AssigneeId,
	})
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (h *ProjectHandler) unassignTask(w http.ResponseWriter, r *http.Request, taskID int32) {
	resp, err := h.project.UnassignTask(outgoingCtx(r), &projectsvcv1.TaskIdRequest{TaskId: taskID})
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (h *ProjectHandler) deleteTask(w http.ResponseWriter, r *http.Request, taskID int32) {
	resp, err := h.project.DeleteTask(outgoingCtx(r), &projectsvcv1.TaskIdRequest{TaskId: taskID})
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

// ─── Comments ────────────────────────────────────────────────────────────────

func (h *ProjectHandler) commentsRouter(w http.ResponseWriter, r *http.Request, taskID int32, rest string) {
	rest = strings.Trim(rest, "/")
	if rest == "" {
		switch r.Method {
		case http.MethodGet:
			h.getComments(w, r, taskID)
		case http.MethodPost:
			h.createComment(w, r, taskID)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	commentID, err := strconv.Atoi(rest)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPut:
		h.updateComment(w, r, taskID, int32(commentID))
	case http.MethodDelete:
		h.deleteComment(w, r, taskID, int32(commentID))
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ProjectHandler) getComments(w http.ResponseWriter, r *http.Request, taskID int32) {
	resp, err := h.project.GetTaskComments(outgoingCtx(r), &projectsvcv1.TaskIdRequest{TaskId: taskID})
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (h *ProjectHandler) createComment(w http.ResponseWriter, r *http.Request, taskID int32) {
	var req projectv1.CreateCommentRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	resp, err := h.project.CreateComment(outgoingCtx(r), &projectsvcv1.CreateCommentGrpcRequest{
		TaskId:  taskID,
		Content: req.Content,
	})
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusCreated, resp)
}

func (h *ProjectHandler) updateComment(w http.ResponseWriter, r *http.Request, taskID, commentID int32) {
	var req projectv1.UpdateTaskCommentRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	resp, err := h.project.UpdateComment(outgoingCtx(r), &projectsvcv1.UpdateCommentGrpcRequest{
		TaskId:    taskID,
		CommentId: commentID,
		Content:   req.Content,
	})
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (h *ProjectHandler) deleteComment(w http.ResponseWriter, r *http.Request, taskID, commentID int32) {
	resp, err := h.project.DeleteComment(outgoingCtx(r), &projectsvcv1.DeleteCommentGrpcRequest{
		TaskId:    taskID,
		CommentId: commentID,
	})
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

// ─── Admin ───────────────────────────────────────────────────────────────────

func (h *ProjectHandler) AdminGetProject(w http.ResponseWriter, r *http.Request) {
	var req projectv1.GetProjectRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	resp, err := h.project.AdminGetProject(outgoingCtx(r), &req)
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (h *ProjectHandler) AdminProjectCRUD(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		var req projectv1.UpdateProjectRequest
		if err := readProtoJSON(r, &req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		resp, err := h.project.AdminUpdateProject(outgoingCtx(r), &req)
		if err != nil {
			grpcStatus(w, err)
			return
		}
		writeProtoJSON(w, http.StatusOK, resp)
	case http.MethodDelete:
		var req projectv1.DeleteProjectRequest
		if err := readProtoJSON(r, &req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		resp, err := h.project.AdminDeleteProject(outgoingCtx(r), &req)
		if err != nil {
			grpcStatus(w, err)
			return
		}
		writeProtoJSON(w, http.StatusOK, resp)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ProjectHandler) AdminGetTask(w http.ResponseWriter, r *http.Request) {
	var req projectv1.GetTaskRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	resp, err := h.project.AdminGetTask(outgoingCtx(r), &req)
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (h *ProjectHandler) AdminTaskCRUD(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		var req projectv1.UpdateTaskRequest
		if err := readProtoJSON(r, &req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		resp, err := h.project.AdminUpdateTask(outgoingCtx(r), &req)
		if err != nil {
			grpcStatus(w, err)
			return
		}
		writeProtoJSON(w, http.StatusOK, resp)
	case http.MethodDelete:
		var req projectv1.DeleteTaskRequest
		if err := readProtoJSON(r, &req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		resp, err := h.project.AdminDeleteTask(outgoingCtx(r), &req)
		if err != nil {
			grpcStatus(w, err)
			return
		}
		writeProtoJSON(w, http.StatusOK, resp)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ProjectHandler) AdminGetComment(w http.ResponseWriter, r *http.Request) {
	var req projectv1.GetCommentRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	resp, err := h.project.AdminGetComment(outgoingCtx(r), &req)
	if err != nil {
		grpcStatus(w, err)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (h *ProjectHandler) AdminCommentCRUD(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		var req projectv1.UpdateCommentRequest
		if err := readProtoJSON(r, &req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		resp, err := h.project.AdminUpdateComment(outgoingCtx(r), &req)
		if err != nil {
			grpcStatus(w, err)
			return
		}
		writeProtoJSON(w, http.StatusOK, resp)
	case http.MethodDelete:
		var req projectv1.DeleteCommentRequest
		if err := readProtoJSON(r, &req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		resp, err := h.project.AdminDeleteComment(outgoingCtx(r), &req)
		if err != nil {
			grpcStatus(w, err)
			return
		}
		writeProtoJSON(w, http.StatusOK, resp)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
