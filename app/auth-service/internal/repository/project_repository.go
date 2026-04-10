package repository

import (
	"auth-service/internal/model"
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type ProjectRepository struct {
	Conn *pgx.Conn
}

func ProjectRepositoryCreate(ctx context.Context, dbLink string) (*ProjectRepository, error) {
	conn, err := pgx.Connect(ctx, dbLink)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(ctx); err != nil {
		conn.Close(ctx)
		return nil, err
	}

	query := `
	CREATE TABLE IF NOT EXISTS projects (
		id SERIAL PRIMARY KEY,
		manager_id INT NOT NULL,
		name TEXT NOT NULL,
		description TEXT NOT NULL,
		status INT NOT NULL,
		start_date TIMESTAMP NOT NULL,
		end_date TIMESTAMP
	);
	`

	_, err = conn.Exec(ctx, query)
	if err != nil {
		return nil, err
	}

	return &ProjectRepository{
		Conn: conn,
	}, nil
}

func (projectRepository *ProjectRepository) CreateProject(ctx context.Context, project model.Project) error {
	query := `
		INSERT INTO projects (manager_id, name, description, status, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	return projectRepository.Conn.QueryRow(ctx, query,
		project.ManagerID,
		project.Name,
		project.Description,
		project.Status,
		project.StartDate,
		project.EndDate,
	).Scan(&project.ID)
}

func (projectRepository *ProjectRepository) DeleteProject(ctx context.Context, projectID int) error {
	query := `DELETE FROM projects WHERE id = $1`

	cmdTag, err := projectRepository.Conn.Exec(ctx, query, projectID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("comment not found")
	}

	return nil
}

func (projectRepository *ProjectRepository) GetProjectsByManagerID(ctx context.Context, managerID int) ([]model.Project, error) {
	query := `
		SELECT id, manager_id, name, description, status, start_date, end_date
		FROM projects
		WHERE manager_id = $1
		ORDER BY start_date
	`

	rows, err := projectRepository.Conn.Query(ctx, query, managerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []model.Project

	for rows.Next() {
		var project model.Project

		err := rows.Scan(
			&project.ID,
			&project.ManagerID,
			&project.Name,
			&project.Description,
			&project.Status,
			&project.StartDate,
			&project.EndDate,
		)
		if err != nil {
			return nil, err
		}

		projects = append(projects, project)
	}

	return projects, nil
}

func (projectRepository *ProjectRepository) ChangeManager(ctx context.Context, projectID int, newManagerID int) error {
	query := `
		UPDATE projects
		SET manager_id = $1
		WHERE id = $2
	`

	cmdTag, err := projectRepository.Conn.Exec(ctx, query, newManagerID, projectID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("project not found")
	}

	return nil
}

func (projectRepository *ProjectRepository) ChangeName(ctx context.Context, projectID int, newName string) error {
	query := `
		UPDATE projects
		SET name = $1
		WHERE id = $2
	`

	cmdTag, err := projectRepository.Conn.Exec(ctx, query, newName, projectID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("project not found")
	}

	return nil
}

func (projectRepository *ProjectRepository) ChangeDescription(ctx context.Context, projectID int, newDescription string) error {
	query := `
		UPDATE projects
		SET description = $1
		WHERE id = $2
	`

	cmdTag, err := projectRepository.Conn.Exec(ctx, query, newDescription, projectID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("project not found")
	}

	return nil
}

func (projectRepository *ProjectRepository) ChangeStatus(ctx context.Context, projectID int, newStatus model.ProjectStatus) error {
	query := `
		UPDATE projects
		SET status = $1
		WHERE id = $2
	`

	cmdTag, err := projectRepository.Conn.Exec(ctx, query, newStatus, projectID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("project not found")
	}

	return nil
}

func (projectRepository *ProjectRepository) ChangeEndDate(ctx context.Context, projectID int, newEndDate *time.Time) error {
	query := `
		UPDATE projects
		SET end_date = $1
		WHERE id = $2
	`

	cmdTag, err := projectRepository.Conn.Exec(ctx, query, newEndDate, projectID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("project not found")
	}

	return nil
}
