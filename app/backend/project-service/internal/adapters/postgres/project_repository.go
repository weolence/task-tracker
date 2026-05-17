package postgres

import (
	"context"
	"errors"
	"time"

	"project-service/internal/core/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProjectRepository struct {
	pool *pgxpool.Pool
}

func NewProjectRepository(pool *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{pool: pool}
}

func (r *ProjectRepository) CreateProject(ctx context.Context, project domain.Project) (int, error) {
	query := `
		INSERT INTO projects (manager_id, name, description, status, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id int
	err := r.pool.QueryRow(ctx, query,
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

func (r *ProjectRepository) GetOwnedProjects(ctx context.Context, userID int32) ([]domain.Project, error) {
	query := `
		SELECT id, manager_id, name, description, status, start_date, end_date
		FROM projects
		WHERE manager_id = $1
		ORDER BY start_date
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := make([]domain.Project, 0)
	for rows.Next() {
		var project domain.Project
		var endDate *time.Time

		if err := rows.Scan(
			&project.ID,
			&project.ManagerID,
			&project.Name,
			&project.Description,
			&project.Status,
			&project.StartDate,
			&endDate,
		); err != nil {
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

func (r *ProjectRepository) GetMemberProjects(ctx context.Context, userID int32) ([]domain.Project, error) {
	query := `
		SELECT p.id, p.manager_id, p.name, p.description, p.status, p.start_date, p.end_date
		FROM project_members pm
		JOIN projects p ON pm.project_id = p.id
		WHERE pm.user_id = $1
		AND p.manager_id != $1
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := make([]domain.Project, 0)
	for rows.Next() {
		var project domain.Project
		var endDate *time.Time

		if err := rows.Scan(
			&project.ID,
			&project.ManagerID,
			&project.Name,
			&project.Description,
			&project.Status,
			&project.StartDate,
			&endDate,
		); err != nil {
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

func (r *ProjectRepository) IsUserMemberOfProject(ctx context.Context, userID int32, projectID int) (bool, error) {
	query := `
		SELECT COUNT(*) > 0
		FROM projects
		WHERE id = $1 AND (manager_id = $2 OR EXISTS (SELECT 1 FROM project_members WHERE project_id = $1 AND user_id = $2))
	`

	var isMember bool
	err := r.pool.QueryRow(ctx, query, projectID, userID).Scan(&isMember)
	if err != nil {
		return false, err
	}

	return isMember, nil
}

func (r *ProjectRepository) GetProjectByID(ctx context.Context, projectID int) (*domain.Project, error) {
	query := `
		SELECT id, manager_id, name, description, status, start_date, end_date
		FROM projects
		WHERE id = $1
	`

	var project domain.Project
	var endDate *time.Time

	err := r.pool.QueryRow(ctx, query, projectID).Scan(
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

func (r *ProjectRepository) GetProjectByName(ctx context.Context, name string) (*domain.Project, error) {
	query := `
		SELECT id, manager_id, name, description, status, start_date, end_date
		FROM projects
		WHERE name = $1
		ORDER BY id
		LIMIT 1
	`

	var project domain.Project
	var endDate *time.Time

	err := r.pool.QueryRow(ctx, query, name).Scan(
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

func (r *ProjectRepository) GetProjectMembers(ctx context.Context, projectID int) ([]int32, error) {
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

	rows, err := r.pool.Query(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]int32, 0)
	for rows.Next() {
		var userID int32
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		members = append(members, userID)
	}

	return members, nil
}

func (r *ProjectRepository) AddProjectMember(ctx context.Context, projectID int, userID int32) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO project_members (project_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (project_id, user_id) DO NOTHING
	`, projectID, userID)
	return err
}

func (r *ProjectRepository) TransferProjectManager(ctx context.Context, projectID int, currentManagerID int32, newManagerID int32) error {
	tx, err := r.pool.Begin(ctx)
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

func (r *ProjectRepository) RemoveProjectMember(ctx context.Context, projectID int, userID int32) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM project_members
		WHERE project_id = $1 AND user_id = $2
	`, projectID, userID)
	return err
}

func (r *ProjectRepository) UpdateProject(ctx context.Context, project domain.Project) error {
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

	cmdTag, err := r.pool.Exec(ctx, query,
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

	_, err = r.pool.Exec(ctx, `
		INSERT INTO project_members (project_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (project_id, user_id) DO NOTHING
	`, project.ID, project.ManagerID)
	return err
}

func (r *ProjectRepository) DeleteProject(ctx context.Context, projectID int32) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM project_members WHERE project_id = $1`, projectID); err != nil {
		return err
	}

	if _, err := r.pool.Exec(ctx, `DELETE FROM tasks WHERE project_id = $1`, projectID); err != nil {
		return err
	}

	cmdTag, err := r.pool.Exec(ctx, `DELETE FROM projects WHERE id = $1`, projectID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("project not found")
	}

	return nil
}
