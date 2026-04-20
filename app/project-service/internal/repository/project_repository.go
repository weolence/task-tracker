package repository

import (
	"context"
	"project-service/internal/model"
	"time"

	"github.com/jackc/pgx/v5"
)

type ProjectRepository struct {
	Conn *pgx.Conn
}

func NewProjectRepository(ctx context.Context, dbLink string) (*ProjectRepository, error) {
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

	CREATE TABLE IF NOT EXISTS project_members (
		project_id INT NOT NULL,
		user_id INT NOT NULL,
		PRIMARY KEY (project_id, user_id)
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

func (projectRepository *ProjectRepository) CreateProject(ctx context.Context, project model.Project) (int, error) {
	query := `
		INSERT INTO projects (manager_id, name, description, status, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id int
	err := projectRepository.Conn.QueryRow(ctx, query,
		project.ManagerID,
		project.Name,
		project.Description,
		project.Status,
		time.Now(),
		project.EndDate,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (projectRepository *ProjectRepository) GetOwnedProjects(ctx context.Context, userID int) ([]model.Project, error) {
	query := `
		SELECT id, manager_id, name, description, status, start_date, end_date
		FROM projects
		WHERE manager_id = $1
		ORDER BY start_date
	`

	rows, err := projectRepository.Conn.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := make([]model.Project, 0)
	for rows.Next() {
		var project model.Project
		var endDate *time.Time

		err := rows.Scan(
			&project.ID,
			&project.ManagerID,
			&project.Name,
			&project.Description,
			&project.Status,
			&project.StartDate,
			&endDate,
		)
		if err != nil {
			return nil, err
		}

		if endDate != nil {
			endDateStr := endDate.Format("2006-01-02")
			project.EndDate = &endDateStr
		}

		projects = append(projects, project)
	}

	return projects, nil
}

func (projectRepository *ProjectRepository) GetMemberProjects(ctx context.Context, userID int) ([]model.Project, error) {
	query := `
		SELECT p.id, p.manager_id, p.name, p.description, p.status, p.start_date, p.end_date
		FROM project_members pm
		JOIN projects p ON pm.project_id = p.id
		WHERE pm.user_id = $1
		AND p.manager_id != $1
	`

	rows, err := projectRepository.Conn.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := make([]model.Project, 0)
	for rows.Next() {
		var project model.Project
		var endDate *time.Time

		err := rows.Scan(
			&project.ID,
			&project.ManagerID,
			&project.Name,
			&project.Description,
			&project.Status,
			&project.StartDate,
			&endDate,
		)
		if err != nil {
			return nil, err
		}

		if endDate != nil {
			endDateStr := endDate.Format("2006-01-02")
			project.EndDate = &endDateStr
		}

		projects = append(projects, project)
	}

	return projects, nil
}

func (projectRepository *ProjectRepository) IsUserMemberOfProject(ctx context.Context, userID int, projectID int) (bool, error) {
	query := `
		SELECT COUNT(*) > 0
		FROM projects
		WHERE id = $1 AND (manager_id = $2 OR EXISTS (SELECT 1 FROM project_members WHERE project_id = $1 AND user_id = $2))
	`

	var isMember bool
	err := projectRepository.Conn.QueryRow(ctx, query, projectID, userID).Scan(&isMember)
	if err != nil {
		return false, err
	}

	return isMember, nil
}

func (projectRepository *ProjectRepository) GetProjectByID(ctx context.Context, projectID int) (*model.Project, error) {
	query := `
		SELECT id, manager_id, name, description, status, start_date, end_date
		FROM projects
		WHERE id = $1
	`

	var project model.Project
	var endDate *time.Time

	err := projectRepository.Conn.QueryRow(ctx, query, projectID).Scan(
		&project.ID,
		&project.ManagerID,
		&project.Name,
		&project.Description,
		&project.Status,
		&project.StartDate,
		&endDate,
	)
	if err != nil {
		return nil, err
	}

	if endDate != nil {
		endDateStr := endDate.Format("2006-01-02")
		project.EndDate = &endDateStr
	}

	return &project, nil
}

func (projectRepository *ProjectRepository) GetProjectMembers(ctx context.Context, projectID int) ([]int, error) {
	query := `
		SELECT user_id
		FROM project_members
		WHERE project_id = $1
		UNION
		SELECT manager_id
		FROM projects
		WHERE id = $1
	`

	rows, err := projectRepository.Conn.Query(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]int, 0)
	for rows.Next() {
		var userID int
		err := rows.Scan(&userID)
		if err != nil {
			return nil, err
		}
		members = append(members, userID)
	}

	return members, nil
}
