package repository

import (
	"auth-service/internal/model"
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type UserRepository struct {
	Conn *pgx.Conn
}

func NewUserRepository(ctx context.Context, dbLink string) (*UserRepository, error) {
	conn, err := pgx.Connect(ctx, dbLink)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(ctx); err != nil {
		conn.Close(ctx)
		return nil, err
	}

	query := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		surname TEXT NOT NULL,
		password TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'user',
		CONSTRAINT users_role_check CHECK (role IN ('user', 'admin'))
	);
	`

	_, err = conn.Exec(ctx, query)
	if err != nil {
		return nil, err
	}

	_, err = conn.Exec(ctx, `
		ALTER TABLE users
		ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'user'
	`)
	if err != nil {
		return nil, err
	}

	_, err = conn.Exec(ctx, `
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1
				FROM pg_constraint
				WHERE conname = 'users_role_check'
			) THEN
				ALTER TABLE users
				ADD CONSTRAINT users_role_check CHECK (role IN ('user', 'admin'));
			END IF;
		END $$;
	`)
	if err != nil {
		return nil, err
	}

	return &UserRepository{
		Conn: conn,
	}, nil
}

func (userRepository *UserRepository) CreateUser(ctx context.Context, user model.User) error {
	query := `
		INSERT INTO users (email, name, surname, password, role)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := userRepository.Conn.Exec(ctx, query,
		user.Email,
		user.Name,
		user.Surname,
		user.Password,
		user.Role,
	)

	return err
}

func (userRepository *UserRepository) DeleteUserByEmail(ctx context.Context, email string) error {
	query := `DELETE FROM users WHERE email = $1`

	cmdTag, err := userRepository.Conn.Exec(ctx, query, email)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("user not found")
	}

	return nil
}

// returns nil without errors if user wasn't found
func (userRepository *UserRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, email, name, surname, password, role
		FROM users
		WHERE email = $1
	`

	var user model.User

	err := userRepository.Conn.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.Surname,
		&user.Password,
		&user.Role,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (userRepository *UserRepository) ChangeName(ctx context.Context, email string, newName string) error {
	query := `
		UPDATE users
		SET name = $1
		WHERE email = $2
	`

	cmdTag, err := userRepository.Conn.Exec(ctx, query, newName, email)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("user not found")
	}

	return nil
}

func (userRepository *UserRepository) ChangeSurname(ctx context.Context, email string, newSurname string) error {
	query := `
		UPDATE users
		SET surname = $1
		WHERE email = $2
	`

	cmdTag, err := userRepository.Conn.Exec(ctx, query, newSurname, email)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("user not found")
	}

	return nil
}

func (userRepository *UserRepository) ChangePassword(ctx context.Context, email string, newHashedPassword string) error {
	query := `
		UPDATE users
		SET password = $1
		WHERE email = $2
	`

	cmdTag, err := userRepository.Conn.Exec(ctx, query, newHashedPassword, email)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("user not found")
	}

	return nil
}

// GetUserByID returns user info by ID without password
func (userRepository *UserRepository) GetUserByID(ctx context.Context, userID int) (*model.User, error) {
	query := `
		SELECT id, email, name, surname, role
		FROM users
		WHERE id = $1
	`

	var user model.User

	err := userRepository.Conn.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.Surname,
		&user.Role,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (userRepository *UserRepository) ChangeRole(ctx context.Context, email string, newRole string) error {
	query := `
		UPDATE users
		SET role = $1
		WHERE email = $2
	`

	cmdTag, err := userRepository.Conn.Exec(ctx, query, newRole, email)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("user not found")
	}

	return nil
}
