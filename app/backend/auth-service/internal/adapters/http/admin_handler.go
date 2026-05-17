package httpadapter

import (
	"auth-service/api/proto/userv1"
	"auth-service/internal/core/domain"
	"auth-service/internal/core/usecase"
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"
)

var proxyClient = &http.Client{Timeout: 15 * time.Second}

type AdminHandler struct {
	auth              *usecase.AuthUseCase
	projectServiceURL string
}

func NewAdminHandler(auth *usecase.AuthUseCase, projectServiceURL string) *AdminHandler {
	return &AdminHandler{
		auth:              auth,
		projectServiceURL: strings.TrimRight(projectServiceURL, "/"),
	}
}

func (h *AdminHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	var req userv1.GetUserRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var (
		user *domain.User
		err  error
	)

	switch {
	case req.UserId != nil && *req.UserId > 0:
		user, err = h.auth.GetUserByID(r.Context(), int(*req.UserId))
	case req.Email != nil && *req.Email != "":
		user, err = h.auth.GetUserByEmail(r.Context(), *req.Email)
	default:
		http.Error(w, "user_id or email is required", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	writeProtoJSON(w, http.StatusOK, &userv1.User{
		Id:      user.ID,
		Email:   user.Email,
		Name:    user.Name,
		Surname: user.Surname,
		Role:    user.Role,
	})
}

func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	var req userv1.UpdateUserRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if req.User == nil || req.User.Id == 0 || req.User.Email == "" || req.User.Name == "" || req.User.Surname == "" {
		http.Error(w, "all user fields are required", http.StatusBadRequest)
		return
	}
	if !domain.IsValidRole(req.User.Role) {
		http.Error(w, "invalid role", http.StatusBadRequest)
		return
	}

	var hashedPassword *string
	if req.Password != nil && *req.Password != "" {
		hashed, err := usecase.HashPassword(*req.Password)
		if err != nil {
			http.Error(w, "failed to hash password", http.StatusInternalServerError)
			return
		}
		hashedPassword = &hashed
	}

	err := h.auth.UpdateUser(r.Context(), domain.User{
		ID:      req.User.Id,
		Email:   req.User.Email,
		Name:    req.User.Name,
		Surname: req.User.Surname,
		Role:    req.User.Role,
	}, hashedPassword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeProtoJSON(w, http.StatusOK, &userv1.OperationResponse{Message: "user updated"})
}

func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	var req userv1.DeleteUserRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := h.auth.DeleteUserByID(r.Context(), req.UserId); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeProtoJSON(w, http.StatusOK, &userv1.OperationResponse{Message: "user deleted"})
}

func (h *AdminHandler) ProxyProject(w http.ResponseWriter, r *http.Request, path string) {
	h.proxyRequest(w, r, path)
}

func (h *AdminHandler) ProxyTask(w http.ResponseWriter, r *http.Request, path string) {
	h.proxyRequest(w, r, path)
}

func (h *AdminHandler) ProxyComment(w http.ResponseWriter, r *http.Request, path string) {
	h.proxyRequest(w, r, path)
}

func (h *AdminHandler) ProxyMembers(w http.ResponseWriter, r *http.Request, path string) {
	h.proxyRequest(w, r, path)
}

func (h *AdminHandler) proxyRequest(w http.ResponseWriter, r *http.Request, path string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	target := h.projectServiceURL + path
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}

	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, target, bytes.NewReader(body))
	if err != nil {
		http.Error(w, "failed to create request", http.StatusInternalServerError)
		return
	}
	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", r.Header.Get("Authorization"))

	resp, err := proxyClient.Do(proxyReq)
	if err != nil {
		http.Error(w, "project service unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "failed to read response", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
}
