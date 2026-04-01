package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type ProjectMembersRepository struct {
	Conn *pgx.Conn
}

func ProjectMembersRepositoryCreate(ctx context.Context, dbLink string) (*ProjectMembersRepository, error) {
	conn, err := pgx.Connect(ctx, dbLink)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(ctx); err != nil {
		conn.Close(ctx)
		return nil, err
	}

	query := `
		CREATE TABLE IF NOT EXISTS project_members (
			project_id INT NOT NULL,
			user_id INT NOT NULL,
			PRIMARY KEY (project_id, user_id)
		)
	`

	_, err = conn.Exec(ctx, query)
	if err != nil {
		return nil, err
	}

	return &ProjectMembersRepository{
		Conn: conn,
	}, nil
}

func (repo *ProjectMembersRepository) GetUsersByProjectID(ctx context.Context, projectID int) ([]int, error) {
	query := `
		SELECT user_id
		FROM project_members
		WHERE project_id = $1
		ORDER BY user_id
	`

	rows, err := repo.Conn.Query(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []int

	for rows.Next() {
		var userID int

		err := rows.Scan(&userID)
		if err != nil {
			return nil, err
		}

		users = append(users, userID)
	}

	return users, nil
}

func (repo *ProjectMembersRepository) GetProjectsByUserID(ctx context.Context, userID int) ([]int, error) {
	query := `
		SELECT project_id
		FROM project_members
		WHERE user_id = $1
		ORDER BY project_id
	`

	rows, err := repo.Conn.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []int

	for rows.Next() {
		var projectID int

		err := rows.Scan(&projectID)
		if err != nil {
			return nil, err
		}

		projects = append(projects, projectID)
	}

	return projects, nil
}

func (repo *ProjectMembersRepository) AddUserToProject(ctx context.Context, projectID int, userID int) error {
	query := `
		INSERT INTO project_members (project_id, user_id)
		VALUES ($1, $2)
	`

	_, err := repo.Conn.Exec(ctx, query, projectID, userID)
	return err
}

func (repo *ProjectMembersRepository) RemoveUserFromProject(ctx context.Context, projectID int, userID int) error {
	query := `
		DELETE FROM project_members
		WHERE project_id = $1 AND user_id = $2
	`

	cmdTag, err := repo.Conn.Exec(ctx, query, projectID, userID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("membership not found")
	}

	return nil
}
