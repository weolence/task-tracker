package postgres

import (
	"auth-service/internal/core/domain"
	"auth-service/internal/core/ports"
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

var _ ports.UserRepository = (*UserRepository)(nil)

func (r *UserRepository) CreateUser(ctx context.Context, user domain.User) error {
	const query = `
		INSERT INTO users (email, name, surname, password, role)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := r.db.Exec(ctx, query, user.Email, user.Name, user.Surname, user.Password, user.Role)
	return err
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	const query = `
		SELECT id, email, name, surname, password, role
		FROM users
		WHERE email = $1`

	var user domain.User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.Name, &user.Surname, &user.Password, &user.Role,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, userID int) (*domain.User, error) {
	const query = `
		SELECT id, email, name, surname, role
		FROM users
		WHERE id = $1`

	var user domain.User
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&user.ID, &user.Email, &user.Name, &user.Surname, &user.Role,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) DeleteUserByEmail(ctx context.Context, email string) error {
	const query = `DELETE FROM users WHERE email = $1`

	tag, err := r.db.Exec(ctx, query, email)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	return nil
}

func (r *UserRepository) DeleteUserByID(ctx context.Context, userID int32) error {
	const query = `DELETE FROM users WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	return nil
}

func (r *UserRepository) UpdateUser(ctx context.Context, user domain.User, hashedPassword *string) error {
	const query = `
		UPDATE users
		SET email    = $1,
		    name     = $2,
		    surname  = $3,
		    role     = $4,
		    password = COALESCE($5, password)
		WHERE id = $6`

	tag, err := r.db.Exec(ctx, query, user.Email, user.Name, user.Surname, user.Role, hashedPassword, user.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	return nil
}

func (r *UserRepository) ChangeRole(ctx context.Context, email string, newRole string) error {
	const query = `UPDATE users SET role = $1 WHERE email = $2`

	tag, err := r.db.Exec(ctx, query, newRole, email)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	return nil
}
