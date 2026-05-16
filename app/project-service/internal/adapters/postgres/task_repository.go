package postgres

import (
	"context"
	"errors"
	"time"

	"project-service/internal/core/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TaskRepository struct {
	pool *pgxpool.Pool
}

func NewTaskRepository(pool *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{pool: pool}
}

func (r *TaskRepository) CreateTask(ctx context.Context, task domain.Task) error {
	query := `
		INSERT INTO tasks (project_id, assignee_id, name, description, priority, difficulty, status, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	return r.pool.QueryRow(ctx, query,
		task.ProjectID,
		task.AssigneeID,
		task.Name,
		task.Description,
		task.Priority,
		task.Difficulty,
		task.Status,
		task.StartDate,
		task.EndDate,
	).Scan(&task.ID)
}

func (r *TaskRepository) DeleteTask(ctx context.Context, taskID int) error {
	cmdTag, err := r.pool.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, taskID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("task not found")
	}

	return nil
}

func (r *TaskRepository) GetTaskByID(ctx context.Context, taskID int) (*domain.Task, error) {
	query := `
		SELECT id, project_id, assignee_id, name, description, priority, difficulty, status, start_date, end_date
		FROM tasks
		WHERE id = $1
	`

	var task domain.Task
	var startDate *time.Time
	var endDate *time.Time
	var assigneeID *int32

	err := r.pool.QueryRow(ctx, query, taskID).Scan(
		&task.ID,
		&task.ProjectID,
		&assigneeID,
		&task.Name,
		&task.Description,
		&task.Priority,
		&task.Difficulty,
		&task.Status,
		&startDate,
		&endDate,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	task.AssigneeID = assigneeID
	task.StartDate = startDate
	if endDate != nil {
		task.EndDate = endDate
	}

	return &task, nil
}

func (r *TaskRepository) GetTasksByProjectAndAssignee(ctx context.Context, projectID int, assigneeID int) ([]domain.Task, error) {
	query := `
		SELECT id, project_id, assignee_id, name, description, priority, difficulty, status, start_date, end_date
		FROM tasks
		WHERE project_id = $1 AND assignee_id = $2 AND status != $3
		ORDER BY priority DESC, difficulty DESC
	`

	rows, err := r.pool.Query(ctx, query, projectID, assigneeID, domain.TaskStatusClosed)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTasks(rows)
}

func (r *TaskRepository) GetAllTasksByProject(ctx context.Context, projectID int) ([]domain.Task, error) {
	query := `
		SELECT id, project_id, assignee_id, name, description, priority, difficulty, status, start_date, end_date
		FROM tasks
		WHERE project_id = $1 AND status != $2
		ORDER BY priority DESC, difficulty DESC
	`

	rows, err := r.pool.Query(ctx, query, projectID, domain.TaskStatusClosed)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTasks(rows)
}

func (r *TaskRepository) GetClosedTasksByProject(ctx context.Context, projectID int) ([]domain.Task, error) {
	query := `
		SELECT id, project_id, assignee_id, name, description, priority, difficulty, status, start_date, end_date
		FROM tasks
		WHERE project_id = $1 AND status = $2
		ORDER BY end_date DESC, priority DESC, difficulty DESC
	`

	rows, err := r.pool.Query(ctx, query, projectID, domain.TaskStatusClosed)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTasks(rows)
}

func (r *TaskRepository) GetTaskByProjectAndName(ctx context.Context, projectID int32, name string) (*domain.Task, error) {
	query := `
		SELECT id, project_id, assignee_id, name, description, priority, difficulty, status, start_date, end_date
		FROM tasks
		WHERE project_id = $1 AND name = $2
		ORDER BY id
		LIMIT 1
	`

	var task domain.Task
	var startDate *time.Time
	var endDate *time.Time
	var assigneeID *int32

	err := r.pool.QueryRow(ctx, query, projectID, name).Scan(
		&task.ID,
		&task.ProjectID,
		&assigneeID,
		&task.Name,
		&task.Description,
		&task.Priority,
		&task.Difficulty,
		&task.Status,
		&startDate,
		&endDate,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	task.AssigneeID = assigneeID
	task.StartDate = startDate
	task.EndDate = endDate

	return &task, nil
}

func (r *TaskRepository) AssignTask(ctx context.Context, taskID int, assigneeID int) error {
	query := `
		UPDATE tasks
		SET assignee_id = $1,
			status = $2,
			start_date = NULL,
			end_date = NULL
		WHERE id = $3 AND status != $4
	`

	cmdTag, err := r.pool.Exec(ctx, query, assigneeID, domain.TaskStatusNotStarted, taskID, domain.TaskStatusClosed)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("task not found")
	}

	return nil
}

func (r *TaskRepository) UnassignTask(ctx context.Context, taskID int) error {
	query := `
		UPDATE tasks
		SET assignee_id = NULL,
			status = $1,
			start_date = NULL,
			end_date = NULL
		WHERE id = $2 AND status != $3
	`

	cmdTag, err := r.pool.Exec(ctx, query, domain.TaskStatusNotStarted, taskID, domain.TaskStatusClosed)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("task not found")
	}

	return nil
}

func (r *TaskRepository) UpdateTaskStatus(ctx context.Context, taskID int, status domain.TaskStatus) error {
	query := `
		UPDATE tasks
		SET status = $1,
			start_date = CASE
				WHEN $1 = 2 AND start_date IS NULL THEN now()
				ELSE start_date
			END,
			end_date = CASE
				WHEN $1 = 4 THEN now()
				WHEN $1 = 1 THEN NULL
				ELSE end_date
			END
		WHERE id = $2 AND status != $3
	`

	cmdTag, err := r.pool.Exec(ctx, query, status, taskID, domain.TaskStatusClosed)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("task not found")
	}

	return nil
}

func (r *TaskRepository) UpdateTask(ctx context.Context, task domain.Task) error {
	query := `
		UPDATE tasks
		SET project_id = $1,
			assignee_id = $2,
			name = $3,
			description = $4,
			priority = $5,
			difficulty = $6,
			status = $7,
			start_date = $8,
			end_date = $9
		WHERE id = $10
	`

	cmdTag, err := r.pool.Exec(ctx, query,
		task.ProjectID,
		task.AssigneeID,
		task.Name,
		task.Description,
		task.Priority,
		task.Difficulty,
		task.Status,
		task.StartDate,
		task.EndDate,
		task.ID,
	)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("task not found")
	}

	return nil
}

func (r *TaskRepository) CloseTask(ctx context.Context, taskID int) error {
	query := `
		UPDATE tasks
		SET status = $2,
			end_date = now()
		WHERE id = $1 AND status != $3
	`

	cmdTag, err := r.pool.Exec(ctx, query, taskID, domain.TaskStatusClosed, domain.TaskStatusClosed)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("task not found")
	}

	return nil
}

func scanTasks(rows interface {
	Next() bool
	Scan(dest ...any) error
	Close()
}) ([]domain.Task, error) {
	tasks := make([]domain.Task, 0)
	for rows.Next() {
		var task domain.Task
		var startDate *time.Time
		var endDate *time.Time
		var assigneeID *int32

		if err := rows.Scan(
			&task.ID,
			&task.ProjectID,
			&assigneeID,
			&task.Name,
			&task.Description,
			&task.Priority,
			&task.Difficulty,
			&task.Status,
			&startDate,
			&endDate,
		); err != nil {
			return nil, err
		}

		task.AssigneeID = assigneeID
		task.StartDate = startDate
		if endDate != nil {
			task.EndDate = endDate
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}
