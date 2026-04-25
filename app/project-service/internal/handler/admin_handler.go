package handler

import (
	"io"
	"net/http"

	"project-service/internal/controller"
	"project-service/internal/model/dto"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type AdminHandler struct {
	projectController *controller.ProjectController
	taskController    *controller.TaskController
}

func NewAdminHandler(projectController *controller.ProjectController, taskController *controller.TaskController) *AdminHandler {
	return &AdminHandler{
		projectController: projectController,
		taskController:    taskController,
	}
}

func (handler *AdminHandler) GetProject(writer http.ResponseWriter, request *http.Request) {
	var req dto.GetProjectRequest
	if err := readProtoJSON(request, &req); err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	project, err := handler.projectController.GetProjectForAdmin(request.Context(), req.ProjectId, req.Name)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusNotFound)
		return
	}

	writeProtoJSON(writer, http.StatusOK, project)
}

func (handler *AdminHandler) UpdateProject(writer http.ResponseWriter, request *http.Request) {
	var req dto.UpdateProjectRequest
	if err := readProtoJSON(request, &req); err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	if err := handler.projectController.UpdateProjectForAdmin(request.Context(), req.Project); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	writeProtoJSON(writer, http.StatusOK, &dto.OperationResponse{Message: "project updated"})
}

func (handler *AdminHandler) DeleteProject(writer http.ResponseWriter, request *http.Request) {
	var req dto.DeleteProjectRequest
	if err := readProtoJSON(request, &req); err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	if err := handler.projectController.DeleteProjectForAdmin(request.Context(), req.ProjectId); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	writeProtoJSON(writer, http.StatusOK, &dto.OperationResponse{Message: "project deleted"})
}

func (handler *AdminHandler) GetTask(writer http.ResponseWriter, request *http.Request) {
	var req dto.GetTaskRequest
	if err := readProtoJSON(request, &req); err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	task, err := handler.taskController.GetTaskForAdmin(request.Context(), req.TaskId, req.ProjectId, req.Name)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusNotFound)
		return
	}

	writeProtoJSON(writer, http.StatusOK, task)
}

func (handler *AdminHandler) UpdateTask(writer http.ResponseWriter, request *http.Request) {
	var req dto.UpdateTaskRequest
	if err := readProtoJSON(request, &req); err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	if err := handler.taskController.UpdateTaskForAdmin(request.Context(), req.Task); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	writeProtoJSON(writer, http.StatusOK, &dto.OperationResponse{Message: "task updated"})
}

func (handler *AdminHandler) DeleteTask(writer http.ResponseWriter, request *http.Request) {
	var req dto.DeleteTaskRequest
	if err := readProtoJSON(request, &req); err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	if err := handler.taskController.DeleteTaskForAdmin(request.Context(), req.TaskId); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	writeProtoJSON(writer, http.StatusOK, &dto.OperationResponse{Message: "task deleted"})
}

func readProtoJSON(request *http.Request, message proto.Message) error {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return err
	}
	defer request.Body.Close()

	return protojson.UnmarshalOptions{AllowPartial: true, DiscardUnknown: true}.Unmarshal(body, message)
}

func writeProtoJSON(writer http.ResponseWriter, status int, message proto.Message) {
	bytes, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}.Marshal(message)
	if err != nil {
		http.Error(writer, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	writer.Write(bytes)
}
