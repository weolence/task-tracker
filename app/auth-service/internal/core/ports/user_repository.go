package ports

import (
	"auth-service/internal/core/domain"
	"context"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user domain.User) error
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	GetUserByID(ctx context.Context, userID int) (*domain.User, error)
	DeleteUserByEmail(ctx context.Context, email string) error
	DeleteUserByID(ctx context.Context, userID int32) error
	UpdateUser(ctx context.Context, user domain.User, hashedPassword *string) error
	ChangeRole(ctx context.Context, email string, newRole string) error
}
