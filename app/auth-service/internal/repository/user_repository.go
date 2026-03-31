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

func UserRepositoryCreate(ctx context.Context, dbLink string) (*UserRepository, error) {
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
		password TEXT NOT NULL
	);
	`

	_, err = conn.Exec(ctx, query)
	if err != nil {
		return nil, err
	}

	return &UserRepository{
		Conn: conn,
	}, nil
}

func (userRepository *UserRepository) CreateUser(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (email, name, surname, password)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	return userRepository.Conn.QueryRow(ctx, query,
		user.Email,
		user.Name,
		user.Surname,
		user.Password,
	).Scan(&user.ID)
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

func (userRepository *UserRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, email, name, surname, password
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
	)

	if err != nil {
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
