package usecase

import (
	"auth-service/internal/core/domain"
	"auth-service/internal/core/ports"
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const TokenValidityTime = 24 * time.Hour

type TokenPayload struct {
	UserID int
	Role   string
}

type AuthUseCase struct {
	users     ports.UserRepository
	jwtSecret []byte
}

func NewAuthUseCase(users ports.UserRepository, jwtSecret []byte) *AuthUseCase {
	return &AuthUseCase{users: users, jwtSecret: jwtSecret}
}

func (uc *AuthUseCase) Register(ctx context.Context, user domain.User) error {
	if user.Email == "" || user.Password == "" || user.Name == "" || user.Surname == "" {
		return domain.ErrInvalidCredentials
	}

	existing, err := uc.users.GetUserByEmail(ctx, user.Email)
	if err != nil {
		return domain.ErrConnectionToDB
	}
	if existing != nil {
		return domain.ErrUserAlreadyExists
	}

	hashed, err := HashPassword(user.Password)
	if err != nil {
		return err
	}

	user.Password = hashed
	user.Role = domain.RoleUser

	return uc.users.CreateUser(ctx, user)
}

func (uc *AuthUseCase) Login(ctx context.Context, email, password string) (string, error) {
	if email == "" || password == "" {
		return "", domain.ErrInvalidCredentials
	}

	user, err := uc.users.GetUserByEmail(ctx, email)
	if err != nil {
		return "", domain.ErrConnectionToDB
	}
	if user == nil {
		return "", domain.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", domain.ErrInvalidCredentials
	}

	token, err := generateJWTToken(user, TokenValidityTime, uc.jwtSecret)
	if err != nil {
		return "", domain.ErrJwtTokenCreation
	}

	return token, nil
}

func (uc *AuthUseCase) ValidateToken(_ context.Context, tokenString string) (*TokenPayload, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return uc.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, domain.ErrInvalidCredentials
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, domain.ErrInvalidCredentials
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return nil, domain.ErrInvalidCredentials
	}

	role, ok := claims["role"].(string)
	if !ok || !domain.IsValidRole(role) {
		return nil, domain.ErrInvalidCredentials
	}

	return &TokenPayload{UserID: int(userIDFloat), Role: role}, nil
}

func (uc *AuthUseCase) GetUserByID(ctx context.Context, userID int) (*domain.User, error) {
	return uc.users.GetUserByID(ctx, userID)
}

func (uc *AuthUseCase) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return uc.users.GetUserByEmail(ctx, email)
}

func (uc *AuthUseCase) DeleteUser(ctx context.Context, email string) error {
	if email == "" {
		return domain.ErrInvalidCredentials
	}
	return uc.users.DeleteUserByEmail(ctx, email)
}

func (uc *AuthUseCase) DeleteUserByID(ctx context.Context, userID int32) error {
	return uc.users.DeleteUserByID(ctx, userID)
}

func (uc *AuthUseCase) UpdateUser(ctx context.Context, user domain.User, hashedPassword *string) error {
	return uc.users.UpdateUser(ctx, user, hashedPassword)
}

func (uc *AuthUseCase) ChangeRole(ctx context.Context, email, role string) error {
	if email == "" || !domain.IsValidRole(role) {
		return domain.ErrInvalidCredentials
	}
	return uc.users.ChangeRole(ctx, email, role)
}

func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashed), err
}

func generateJWTToken(user *domain.User, validity time.Duration, secret []byte) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(validity).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}
