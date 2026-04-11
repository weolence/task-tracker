package handler

import (
	"auth-service/internal/controller"
	"auth-service/internal/model"
	"encoding/json"
	"net/http"
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
	var registerRequest RegisterRequest

	if err := json.NewDecoder(request.Body).Decode(&registerRequest); err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	user := model.User{
		Email:    registerRequest.Email,
		Password: registerRequest.Password,
		Name:     registerRequest.Name,
		Surname:  registerRequest.Surname,
	}

	err := authHandler.authController.Register(request.Context(), user)
	if err != nil {
		http.Error(writer, "unable to register", http.StatusBadRequest)
		return
	}

	writer.WriteHeader(http.StatusCreated)
}

func (authHandler *AuthHandler) Login(writer http.ResponseWriter, request *http.Request) {
	var loginRequest LoginRequest

	if err := json.NewDecoder(request.Body).Decode(&loginRequest); err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	token, err := authHandler.authController.Login(request.Context(), loginRequest.Email, loginRequest.Password)
	if err != nil {
		http.Error(writer, "invalid credentials", http.StatusUnauthorized)
		return
	}

	resp := LoginResponse{
		Token: token,
	}

	writer.Header().Set("Content-Type", "application/json")
	json.NewEncoder(writer).Encode(resp)
}

func (authHandler *AuthHandler) ValidateToken(writer http.ResponseWriter, request *http.Request) {
	var validateRequest ValidateTokenRequest

	if err := json.NewDecoder(request.Body).Decode(&validateRequest); err != nil {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}

	userID, err := authHandler.authController.ValidateToken(request.Context(), validateRequest.Token)
	if err != nil {
		http.Error(writer, "invalid token", http.StatusUnauthorized)
		return
	}

	resp := ValidateTokenResponse{
		UserID: userID,
	}

	writer.Header().Set("Content-Type", "application/json")
	json.NewEncoder(writer).Encode(resp)
}
