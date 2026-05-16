package httpadapter

import (
	"auth-service/api/proto/userv1"
	"auth-service/internal/core/domain"
	"auth-service/internal/core/usecase"
	"net/http"
)

type AuthHandler struct {
	auth *usecase.AuthUseCase
}

func NewAuthHandler(auth *usecase.AuthUseCase) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req userv1.RegisterRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	user := domain.User{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
		Surname:  req.Surname,
	}

	if err := h.auth.Register(r.Context(), user); err != nil {
		http.Error(w, "unable to register", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req userv1.LoginRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	token, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	writeProtoJSON(w, http.StatusOK, &userv1.LoginResponse{Token: token})
}

func (h *AuthHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	var req userv1.ValidateTokenRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	payload, err := h.auth.ValidateToken(r.Context(), req.Token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	writeProtoJSON(w, http.StatusOK, &userv1.ValidateTokenResponse{
		UserId: int32(payload.UserID),
		Role:   payload.Role,
	})
}

func (h *AuthHandler) GetUserInfo(w http.ResponseWriter, r *http.Request) {
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
	case req.UserId != nil:
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
