package controller

import (
	errs "auth-service/internal/error"
	"auth-service/internal/model"
	"auth-service/internal/repository"
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenValidityTime = 24 * time.Hour
)

type AuthController struct {
	userRepository repository.UserRepository
	securityKey    []byte
}

type TokenPayload struct {
	UserID int
	Role   string
}

func NewAuthController(userRepository repository.UserRepository, securityKey []byte) (*AuthController, error) {
	return &AuthController{
		userRepository: userRepository,
		securityKey:    securityKey,
	}, nil
}

// registers user if there is no account with such email.
// otherwise error returned.
func (authController *AuthController) Register(ctx context.Context, user model.User) error {
	if user.Email == "" || user.Password == "" || user.Name == "" || user.Surname == "" {
		return errs.ErrInvalidCredentials
	}

	u, err := authController.userRepository.GetUserByEmail(ctx, user.Email)
	if err != nil {
		return errs.ErrConnectionToDB
	}

	if u != nil {
		return errs.ErrUserAlreadyExists
	}

	hashedPassword, err := HashPassword(user.Password)
	if err != nil {
		return err
	}

	user.Password = hashedPassword
	user.Role = model.RoleUser

	err = authController.userRepository.CreateUser(ctx, user)

	return err
}

func (authController *AuthController) DeleteUser(ctx context.Context, email string) error {
	if email == "" {
		return errs.ErrInvalidCredentials
	}

	return authController.userRepository.DeleteUserByEmail(ctx, email)
}

// returns signed jwt token if user's credentials were correct
func (authController *AuthController) Login(ctx context.Context, email, password string) (string, error) {
	if email == "" || password == "" {
		return "", errs.ErrInvalidCredentials
	}

	user, err := authController.userRepository.GetUserByEmail(ctx, email)
	if err != nil {
		return "", errs.ErrConnectionToDB
	}

	if user == nil {
		return "", errs.ErrInvalidCredentials
	}

	if err := CompareHashAndPassword(user.Password, password); err != nil {
		return "", errs.ErrInvalidCredentials
	}

	signedToken, err := GenerateJwtToken(user, TokenValidityTime, authController.securityKey)
	if err != nil {
		return "", errs.ErrJwtTokenCreation
	}

	return signedToken, nil
}

func (authController *AuthController) ValidateToken(ctx context.Context, tokenString string) (int, error) {
	payload, err := authController.ValidateTokenWithRole(ctx, tokenString)
	if err != nil {
		return 0, err
	}

	return payload.UserID, nil
}

func (authController *AuthController) ValidateTokenWithRole(ctx context.Context, tokenString string) (*TokenPayload, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return authController.securityKey, nil
	})

	if err != nil || !token.Valid {
		return nil, errs.ErrInvalidCredentials
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errs.ErrInvalidCredentials
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return nil, errs.ErrInvalidCredentials
	}

	role, ok := claims["role"].(string)
	if !ok || role == "" {
		role = model.RoleUser
	}
	if !isValidRole(role) {
		return nil, errs.ErrInvalidCredentials
	}

	return &TokenPayload{
		UserID: int(userIDFloat),
		Role:   role,
	}, nil
}

func (authController *AuthController) GetUser(ctx context.Context, userID int) (*model.User, error) {
	return authController.userRepository.GetUserByID(ctx, userID)
}

func (authController *AuthController) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	return authController.userRepository.GetUserByEmail(ctx, email)
}

func (authController *AuthController) ChangeRole(ctx context.Context, email string, role string) error {
	if email == "" || !isValidRole(role) {
		return errs.ErrInvalidCredentials
	}

	return authController.userRepository.ChangeRole(ctx, email, role)
}

func isValidRole(role string) bool {
	return role == model.RoleUser || role == model.RoleAdmin
}
