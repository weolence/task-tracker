package handler

import (
	"auth-service/internal/controller"
	"auth-service/internal/model"
	"auth-service/internal/model/dto"
	"io"
	"net/http"
	"strconv"

	"google.golang.org/protobuf/encoding/protojson"
)

/*
end-points for registration/authorization will be here
POST /register
POST /login
POST /validate-token
GET  /profile
*/
type AuthHandler struct {
	authController *controller.AuthController
}

func NewAuthHandler(authController *controller.AuthController) *AuthHandler {
	return &AuthHandler{
		authController: authController,
	}
}

func (authHandler *AuthHandler) Register(writer http.ResponseWriter, request *http.Request) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}
	defer request.Body.Close()

	var registerRequest dto.RegisterRequest

	err = protojson.UnmarshalOptions{AllowPartial: true, DiscardUnknown: true}.Unmarshal(body, &registerRequest)
	if err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	user := model.User{
		Email:    registerRequest.Email,
		Password: registerRequest.Password,
		Name:     registerRequest.Name,
		Surname:  registerRequest.Surname,
	}

	err = authHandler.authController.Register(request.Context(), user)
	if err != nil {
		http.Error(writer, "unable to register", http.StatusBadRequest)
		return
	}

	writer.WriteHeader(http.StatusCreated)
}

func (authHandler *AuthHandler) Login(writer http.ResponseWriter, request *http.Request) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}
	defer request.Body.Close()

	var loginRequest dto.LoginRequest

	err = protojson.UnmarshalOptions{AllowPartial: true, DiscardUnknown: true}.Unmarshal(body, &loginRequest)
	if err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	token, err := authHandler.authController.Login(request.Context(), loginRequest.Email, loginRequest.Password)
	if err != nil {
		http.Error(writer, "invalid credentials", http.StatusUnauthorized)
		return
	}

	resp := dto.LoginResponse{
		Token: token,
	}

	bytes, err := protojson.Marshal(&resp)
	if err != nil {
		http.Error(writer, "unable to login", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.Write(bytes)
}

func (authHandler *AuthHandler) ValidateToken(writer http.ResponseWriter, request *http.Request) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}
	defer request.Body.Close()

	var validateRequest dto.ValidateTokenRequest

	err = protojson.UnmarshalOptions{AllowPartial: true, DiscardUnknown: true}.Unmarshal(body, &validateRequest)
	if err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	userID, err := authHandler.authController.ValidateToken(request.Context(), validateRequest.Token)
	if err != nil {
		http.Error(writer, "invalid token", http.StatusUnauthorized)
		return
	}

	resp := dto.ValidateTokenResponse{
		UserId: int32(userID),
	}

	bytes, err := protojson.Marshal(&resp)
	if err != nil {
		http.Error(writer, "unable to validate token", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.Write(bytes)
}

func (authHandler *AuthHandler) GetUserInfo(writer http.ResponseWriter, request *http.Request) {
	userIDStr := request.URL.Query().Get("user_id")
	if userIDStr == "" {
		http.Error(writer, "user_id query parameter is required", http.StatusBadRequest)
		return
	}

	// Парсим ID как обычное число
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(writer, "invalid user_id", http.StatusBadRequest)
		return
	}

	user, err := authHandler.authController.GetUser(request.Context(), userID)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	if user == nil {
		http.Error(writer, "user not found", http.StatusNotFound)
		return
	}

	resp := dto.User{
		Id:      user.ID,
		Email:   user.Email,
		Name:    user.Name,
		Surname: user.Surname,
	}

	response, err := protojson.Marshal(&resp)
	if err != nil {
		http.Error(writer, "unable to get user info", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.Write(response)
}
