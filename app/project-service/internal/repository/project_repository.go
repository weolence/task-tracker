package repository

import (
	"context"
	"errors"
	"project-service/internal/model"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProjectRepository struct {
	Conn *pgxpool.Pool
}

func NewProjectRepository(ctx context.Context, dbLink string) (*ProjectRepository, error) {
	conn, err := pgxpool.New(ctx, dbLink)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(ctx); err != nil {
		conn.Close()
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

func (projectRepository *ProjectRepository) GetOwnedProjects(ctx context.Context, userID int32) ([]model.Project, error) {
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

func (projectRepository *ProjectRepository) GetMemberProjects(ctx context.Context, userID int32) ([]model.Project, error) {
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

func (projectRepository *ProjectRepository) IsUserMemberOfProject(ctx context.Context, userID int32, projectID int) (bool, error) {
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if endDate != nil {
		endDateStr := endDate.Format("2006-01-02")
		project.EndDate = &endDateStr
	}

	return &project, nil
}

func (projectRepository *ProjectRepository) GetProjectByName(ctx context.Context, name string) (*model.Project, error) {
	query := `
		SELECT id, manager_id, name, description, status, start_date, end_date
		FROM projects
		WHERE name = $1
		ORDER BY id
		LIMIT 1
	`

	var project model.Project
	var endDate *time.Time

	err := projectRepository.Conn.QueryRow(ctx, query, name).Scan(
		&project.ID,
		&project.ManagerID,
		&project.Name,
		&project.Description,
		&project.Status,
		&project.StartDate,
		&endDate,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if endDate != nil {
		endDateStr := endDate.Format("2006-01-02")
		project.EndDate = &endDateStr
	}

	return &project, nil
}

func (projectRepository *ProjectRepository) GetProjectMembers(ctx context.Context, projectID int) ([]int32, error) {
	query := `
		SELECT DISTINCT user_id
		FROM project_members
		WHERE project_id = $1
		UNION
		SELECT manager_id
		FROM projects
		WHERE id = $1
		ORDER BY 1
	`

	rows, err := projectRepository.Conn.Query(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]int32, 0)
	for rows.Next() {
		var userID int32
		err := rows.Scan(&userID)
		if err != nil {
			return nil, err
		}
		members = append(members, int32(userID))
	}

	return members, nil
}

func (projectRepository *ProjectRepository) AddProjectMember(ctx context.Context, projectID int, userID int32) error {
	_, err := projectRepository.Conn.Exec(ctx, `
		INSERT INTO project_members (project_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (project_id, user_id) DO NOTHING
	`, projectID, userID)
	return err
}

func (projectRepository *ProjectRepository) TransferProjectManager(ctx context.Context, projectID int, currentManagerID int32, newManagerID int32) error {
	tx, err := projectRepository.Conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	cmdTag, err := tx.Exec(ctx, `
		UPDATE projects
		SET manager_id = $1
		WHERE id = $2 AND manager_id = $3
	`, newManagerID, projectID, currentManagerID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("project not found or manager mismatch")
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO project_members (project_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (project_id, user_id) DO NOTHING
	`, projectID, currentManagerID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (projectRepository *ProjectRepository) UpdateProject(ctx context.Context, project model.Project) error {
	var endDate any
	if project.EndDate != nil && *project.EndDate != "" {
		parsedEndDate, err := time.Parse("2006-01-02", *project.EndDate)
		if err != nil {
			return err
		}
		endDate = parsedEndDate
	}

	query := `
		UPDATE projects
		SET manager_id = $1,
			name = $2,
			description = $3,
			status = $4,
			start_date = $5,
			end_date = $6
		WHERE id = $7
	`

	cmdTag, err := projectRepository.Conn.Exec(ctx, query,
		project.ManagerID,
		project.Name,
		project.Description,
		project.Status,
		project.StartDate,
		endDate,
		project.ID,
	)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("project not found")
	}

	_, err = projectRepository.Conn.Exec(ctx, `
		INSERT INTO project_members (project_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (project_id, user_id) DO NOTHING
	`, project.ID, project.ManagerID)
	return err
}

func (projectRepository *ProjectRepository) DeleteProject(ctx context.Context, projectID int32) error {
	if _, err := projectRepository.Conn.Exec(ctx, `DELETE FROM project_members WHERE project_id = $1`, projectID); err != nil {
		return err
	}

	if _, err := projectRepository.Conn.Exec(ctx, `DELETE FROM tasks WHERE project_id = $1`, projectID); err != nil {
		return err
	}

	cmdTag, err := projectRepository.Conn.Exec(ctx, `DELETE FROM projects WHERE id = $1`, projectID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("project not found")
	}

	return nil
}
