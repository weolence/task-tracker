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
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO projects (name, description, status, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var id int
	err = tx.QueryRow(ctx, query,
		project.Name,
		project.Description,
		project.Status,
		time.Now(),
		project.EndDate,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO project_user_roles (project_id, user_id, role)
		VALUES ($1, $2, 'manager')
	`, id, project.ManagerID)
	if err != nil {
		return 0, err
	}

	return id, tx.Commit(ctx)
}

func (r *ProjectRepository) GetOwnedProjects(ctx context.Context, userID int32) ([]domain.Project, error) {
	query := `
		SELECT p.id, p.name, p.description, p.status, p.start_date, p.end_date, pur.user_id
		FROM projects p
		JOIN project_user_roles pur ON p.id = pur.project_id
		WHERE pur.user_id = $1 AND pur.role = 'manager'
		ORDER BY p.start_date
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
			&project.Name,
			&project.Description,
			&project.Status,
			&project.StartDate,
			&endDate,
			&project.ManagerID,
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
		SELECT p.id, p.name, p.description, p.status, p.start_date, p.end_date, manager_role.user_id
		FROM projects p
		JOIN project_user_roles pur ON p.id = pur.project_id
		JOIN project_user_roles manager_role ON p.id = manager_role.project_id AND manager_role.role = 'manager'
		WHERE pur.user_id = $1 AND pur.role = 'member'
		ORDER BY p.start_date
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
			&project.Name,
			&project.Description,
			&project.Status,
			&project.StartDate,
			&endDate,
			&project.ManagerID,
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
		FROM project_user_roles
		WHERE project_id = $1 AND user_id = $2
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
		SELECT p.id, p.name, p.description, p.status, p.start_date, p.end_date, manager_role.user_id
		FROM projects p
		JOIN project_user_roles manager_role ON p.id = manager_role.project_id AND manager_role.role = 'manager'
		WHERE p.id = $1
	`

	var project domain.Project
	var endDate *time.Time

	err := r.pool.QueryRow(ctx, query, projectID).Scan(
		&project.ID,
		&project.Name,
		&project.Description,
		&project.Status,
		&project.StartDate,
		&endDate,
		&project.ManagerID,
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
		SELECT p.id, p.name, p.description, p.status, p.start_date, p.end_date, manager_role.user_id
		FROM projects p
		JOIN project_user_roles manager_role ON p.id = manager_role.project_id AND manager_role.role = 'manager'
		WHERE p.name = $1
		ORDER BY p.id
		LIMIT 1
	`

	var project domain.Project
	var endDate *time.Time

	err := r.pool.QueryRow(ctx, query, name).Scan(
		&project.ID,
		&project.Name,
		&project.Description,
		&project.Status,
		&project.StartDate,
		&endDate,
		&project.ManagerID,
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
		FROM project_user_roles
		WHERE project_id = $1
		ORDER BY user_id
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
		INSERT INTO project_user_roles (project_id, user_id, role)
		VALUES ($1, $2, 'member')
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

	// Verify current manager exists
	var exists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM project_user_roles
			WHERE project_id = $1 AND user_id = $2 AND role = 'manager'
		)
	`, projectID, currentManagerID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("project not found or manager mismatch")
	}

	// Update current manager to member
	_, err = tx.Exec(ctx, `
		UPDATE project_user_roles
		SET role = 'member'
		WHERE project_id = $1 AND user_id = $2 AND role = 'manager'
	`, projectID, currentManagerID)
	if err != nil {
		return err
	}

	// Set new manager
	_, err = tx.Exec(ctx, `
		INSERT INTO project_user_roles (project_id, user_id, role)
		VALUES ($1, $2, 'manager')
		ON CONFLICT (project_id, user_id) DO UPDATE SET role = 'manager'
	`, projectID, newManagerID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *ProjectRepository) RemoveProjectMember(ctx context.Context, projectID int, userID int32) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM project_user_roles
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
		SET name = $1,
			description = $2,
			status = $3,
			start_date = $4,
			end_date = $5
		WHERE id = $6
	`

	cmdTag, err := r.pool.Exec(ctx, query,
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

	return nil
}

func (r *ProjectRepository) DeleteProject(ctx context.Context, projectID int32) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM project_user_roles WHERE project_id = $1`, projectID); err != nil {
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
