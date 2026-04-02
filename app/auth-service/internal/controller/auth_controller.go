package service

import (
	errs "auth-service/internal/error"
	"auth-service/internal/model"
	"auth-service/internal/repository"
	"context"
	"errors"
	"time"
)

const (
	TokenValidityTime = 24 * time.Hour
)

type AuthController struct {
	userRepository repository.UserRepository
	securityKey    []byte
}

func AuthControllerCreate(userRepository repository.UserRepository) (*AuthController, error) {
	return &AuthController{
		userRepository: userRepository,
	}, nil
}

// registers user if there is no account with such email.
// otherwise error returned.
func (authController *AuthController) Register(ctx context.Context, user *model.User) error {
	if user.Email == "" || user.Password == "" || user.Name == "" || user.Surname == "" {
		return errs.ErrInvalidCredentials
	}

	_, err := authController.userRepository.GetUserByEmail(ctx, user.Email)
	if err != nil {
		return errs.ErrUserAlreadyExists
	}

	if !errors.Is(err, errs.ErrUserNotFound) {
		return err
	}

	hashedPassword, err := HashPassword(user.Password)
	if err != nil {
		return err
	}

	user.Password = hashedPassword

	err = authController.userRepository.CreateUser(ctx, user)

	return err
}

// returns signed jwt token if user's credentials were correct
func (authController *AuthController) Login(ctx context.Context, email, password string) (string, error) {
	if email == "" || password == "" {
		return "", errs.ErrInvalidCredentials
	}

	user, err := authController.userRepository.GetUserByEmail(ctx, email)
	if err != nil {
		return "", errs.ErrInvalidCredentials
	}

	err = CompareHashAndPassword(user.Password, password)
	if err != nil {
		return "", errs.ErrInvalidCredentials
	}

	signedToken, err := GenerateJwtToken(user, TokenValidityTime, authController.securityKey)
	if err != nil {
		return "", errs.ErrCannotCreateJwtToken
	}

	return signedToken, nil
}
