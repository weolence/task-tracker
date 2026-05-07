package handler

import (
	"auth-service/internal/controller"
	"auth-service/internal/model"
	"auth-service/internal/model/dto"
	"auth-service/internal/repository"
	"bytes"
	"io"
	"net/http"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type AdminHandler struct {
	userRepository    *repository.UserRepository
	projectServiceURL string
}

func NewAdminHandler(userRepository *repository.UserRepository, projectServiceURL string) *AdminHandler {
	return &AdminHandler{
		userRepository:    userRepository,
		projectServiceURL: strings.TrimRight(projectServiceURL, "/"),
	}
}

func (handler *AdminHandler) GetUser(writer http.ResponseWriter, request *http.Request) {
	var req dto.GetUserRequest
	if err := readProtoJSON(request, &req); err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	var (
		user *model.User
		err  error
	)

	switch {
	case req.UserId != nil && *req.UserId > 0:
		user, err = handler.userRepository.GetUserByID(request.Context(), int(*req.UserId))
	case req.Email != nil && *req.Email != "":
		user, err = handler.userRepository.GetUserByEmail(request.Context(), *req.Email)
	default:
		http.Error(writer, "user_id or email is required", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(writer, "user not found", http.StatusNotFound)
		return
	}

	writeProtoJSON(writer, http.StatusOK, &dto.User{
		Id:      user.ID,
		Email:   user.Email,
		Name:    user.Name,
		Surname: user.Surname,
		Role:    user.Role,
	})
}

func (handler *AdminHandler) UpdateUser(writer http.ResponseWriter, request *http.Request) {
	var req dto.UpdateUserRequest
	if err := readProtoJSON(request, &req); err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	if req.User == nil || req.User.Id == 0 || req.User.Email == "" || req.User.Name == "" || req.User.Surname == "" {
		http.Error(writer, "all user fields are required", http.StatusBadRequest)
		return
	}
	if req.User.Role != model.RoleAdmin && req.User.Role != model.RoleUser {
		http.Error(writer, "invalid role", http.StatusBadRequest)
		return
	}

	var hashedPassword *string
	if req.Password != nil && *req.Password != "" {
		value, err := controller.HashPassword(*req.Password)
		if err != nil {
			http.Error(writer, "failed to hash password", http.StatusInternalServerError)
			return
		}
		hashedPassword = &value
	}

	err := handler.userRepository.UpdateUser(request.Context(), model.User{
		ID:      req.User.Id,
		Email:   req.User.Email,
		Name:    req.User.Name,
		Surname: req.User.Surname,
		Role:    req.User.Role,
	}, hashedPassword)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	writeProtoJSON(writer, http.StatusOK, &dto.OperationResponse{Message: "user updated"})
}

func (handler *AdminHandler) DeleteUser(writer http.ResponseWriter, request *http.Request) {
	var req dto.DeleteUserRequest
	if err := readProtoJSON(request, &req); err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	if err := handler.userRepository.DeleteUserByID(request.Context(), req.UserId); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	writeProtoJSON(writer, http.StatusOK, &dto.OperationResponse{Message: "user deleted"})
}

func (handler *AdminHandler) ProxyProject(writer http.ResponseWriter, request *http.Request, path string) {
	handler.proxyRequest(writer, request, path)
}

func (handler *AdminHandler) ProxyTask(writer http.ResponseWriter, request *http.Request, path string) {
	handler.proxyRequest(writer, request, path)
}

func (handler *AdminHandler) ProxyComment(writer http.ResponseWriter, request *http.Request, path string) {
	handler.proxyRequest(writer, request, path)
}

func (handler *AdminHandler) proxyRequest(writer http.ResponseWriter, request *http.Request, path string) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}
	defer request.Body.Close()

	proxyRequest, err := http.NewRequestWithContext(request.Context(), request.Method, handler.projectServiceURL+path, bytes.NewReader(body))
	if err != nil {
		http.Error(writer, "failed to create request", http.StatusInternalServerError)
		return
	}
	proxyRequest.Header.Set("Content-Type", "application/json")
	proxyRequest.Header.Set("Authorization", request.Header.Get("Authorization"))

	response, err := http.DefaultClient.Do(proxyRequest)
	if err != nil {
		http.Error(writer, "project service unavailable", http.StatusBadGateway)
		return
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		http.Error(writer, "failed to read response", http.StatusBadGateway)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(response.StatusCode)
	writer.Write(responseBody)
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
