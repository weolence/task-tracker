package httpadapter

import (
	"fmt"
	"net/http"

	authsvcv1 "auth-service/api/proto/authsvcv1"
	userv1 "auth-service/api/proto/userv1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthHandler struct {
	auth authsvcv1.AuthServiceClient
}

func NewAuthHandler(auth authsvcv1.AuthServiceClient) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req userv1.RegisterRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if _, err := h.auth.Register(r.Context(), &req); err != nil {
		grpcErr, _ := status.FromError(err)
		http.Error(w, grpcErr.Message(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req userv1.LoginRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	resp, err := h.auth.Login(r.Context(), &req)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) GetCurrentUserID(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(ctxUserID).(int32)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"user_id":%d}`, userID)
}

func (h *AuthHandler) GetUserInfo(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(ctxUserID).(int32)
	resp, err := h.auth.GetUserInfo(outgoingCtx(r), &userv1.GetUserRequest{UserId: &userID})
	if err != nil {
		grpcErr, _ := status.FromError(err)
		if grpcErr.Code() == codes.NotFound {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

// ─── Admin ────────────────────────────────────────────────────────────────────

func (h *AuthHandler) AdminGetUser(w http.ResponseWriter, r *http.Request) {
	var req userv1.GetUserRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	resp, err := h.auth.GetUser(outgoingCtx(r), &req)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		if grpcErr.Code() == codes.NotFound {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) AdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	var req userv1.UpdateUserRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	resp, err := h.auth.UpdateUser(outgoingCtx(r), &req)
	if err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) AdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	var req userv1.DeleteUserRequest
	if err := readProtoJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	resp, err := h.auth.DeleteUser(outgoingCtx(r), &req)
	if err != nil {
		grpcErr, _ := status.FromError(err)
		if grpcErr.Code() == codes.NotFound {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}
	writeProtoJSON(w, http.StatusOK, resp)
}
