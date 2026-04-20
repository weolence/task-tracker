package repository

import (
	"context"
	"errors"
	"project-service/internal/model"
	"time"

	"github.com/jackc/pgx/v5"
)

type TaskRepository struct {
	Conn *pgx.Conn
}

func NewTaskRepository(ctx context.Context, dbLink string) (*TaskRepository, error) {
	conn, err := pgx.Connect(ctx, dbLink)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(ctx); err != nil {
		conn.Close(ctx)
		return nil, err
	}

	query := `
	CREATE TABLE IF NOT EXISTS tasks (
		id SERIAL PRIMARY KEY,
		project_id INT NOT NULL,
		assignee_id INT,
		name TEXT NOT NULL,
		description TEXT,
		priority INT NOT NULL,
		difficulty INT NOT NULL,
		status INT NOT NULL,
		start_date TIMESTAMP,
		end_date TIMESTAMP
	);
	`

	_, err = conn.Exec(ctx, query)
	if err != nil {
		return nil, err
	}

	return &TaskRepository{
		Conn: conn,
	}, nil
}

func (taskRepository *TaskRepository) CreateTask(ctx context.Context, task model.Task) error {
	query := `
		INSERT INTO tasks (project_id, assignee_id, name, description, priority, difficulty, status, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	return taskRepository.Conn.QueryRow(ctx, query,
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

func (taskRepository *TaskRepository) DeleteTask(ctx context.Context, taskID int) error {
	query := `DELETE FROM tasks WHERE id = $1`

	cmdTag, err := taskRepository.Conn.Exec(ctx, query, taskID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("task not found")
	}

	return nil
}

func (taskRepository *TaskRepository) ChangeAssignee(ctx context.Context, taskID int, newAssigneeID int) error {
	query := `
		UPDATE tasks
		SET assignee_id = $1
		WHERE id = $2
	`

	cmdTag, err := taskRepository.Conn.Exec(ctx, query, newAssigneeID, taskID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("task not found")
	}

	return nil
}

func (taskRepository *TaskRepository) ChangeName(ctx context.Context, taskID int, newName string) error {
	query := `
		UPDATE tasks
		SET name = $1
		WHERE id = $2
	`

	cmdTag, err := taskRepository.Conn.Exec(ctx, query, newName, taskID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("task not found")
	}

	return nil
}

func (taskRepository *TaskRepository) ChangePriority(ctx context.Context, taskID int, newPriority model.TaskPriority) error {
	query := `
		UPDATE tasks
		SET priority = $1
		WHERE id = $2
	`

	cmdTag, err := taskRepository.Conn.Exec(ctx, query, newPriority, taskID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("task not found")
	}

	return nil
}

func (taskRepository *TaskRepository) ChangeDifficulty(ctx context.Context, taskID int, newDifficulty model.TaskDifficulty) error {
	query := `
		UPDATE tasks
		SET difficulty = $1
		WHERE id = $2
	`

	cmdTag, err := taskRepository.Conn.Exec(ctx, query, newDifficulty, taskID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("task not found")
	}

	return nil
}

func (taskRepository *TaskRepository) ChangeStatus(ctx context.Context, taskID int, newStatus model.TaskStatus) error {
	query := `
		UPDATE tasks
		SET status = $1,
			start_date = CASE
				WHEN $1 = 1 AND start_date IS NULL THEN now()
				ELSE start_date
			END
		WHERE id = $2
	`

	cmdTag, err := taskRepository.Conn.Exec(ctx, query, newStatus, taskID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("task not found")
	}

	return nil
}

func (taskRepository *TaskRepository) UpdateTaskStatus(ctx context.Context, taskID int, status model.TaskStatus) error {
	return taskRepository.ChangeStatus(ctx, taskID, status)
}

func (taskRepository *TaskRepository) GetTasksByProjectAndAssignee(ctx context.Context, projectID int, assigneeID int) ([]model.Task, error) {
	query := `
		SELECT id, project_id, assignee_id, name, description, priority, difficulty, status, start_date, end_date
		FROM tasks
		WHERE project_id = $1 AND assignee_id = $2
		ORDER BY priority DESC, difficulty DESC
	`

	rows, err := taskRepository.Conn.Query(ctx, query, projectID, assigneeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]model.Task, 0)
	for rows.Next() {
		var task model.Task
		var startDate *time.Time
		var endDate *time.Time

		err := rows.Scan(
			&task.ID,
			&task.ProjectID,
			&task.AssigneeID,
			&task.Name,
			&task.Description,
			&task.Priority,
			&task.Difficulty,
			&task.Status,
			&startDate,
			&endDate,
		)
		if err != nil {
			return nil, err
		}

		if endDate != nil {
			task.EndDate = endDate
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (taskRepository *TaskRepository) GetAllTasksByProject(ctx context.Context, projectID int) ([]model.Task, error) {
	query := `
		SELECT id, project_id, assignee_id, name, description, priority, difficulty, status, start_date, end_date
		FROM tasks
		WHERE project_id = $1
		ORDER BY priority DESC, difficulty DESC
	`

	rows, err := taskRepository.Conn.Query(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]model.Task, 0)
	for rows.Next() {
		var task model.Task
		var startDate *time.Time
		var endDate *time.Time
		var assigneeID *int

		err := rows.Scan(
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

func (taskRepository *TaskRepository) AssignTask(ctx context.Context, taskID int, assigneeID int) error {
	query := `
		UPDATE tasks
		SET assignee_id = $1
		WHERE id = $2
	`

	cmdTag, err := taskRepository.Conn.Exec(ctx, query, assigneeID, taskID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("task not found")
	}

	return nil
}

func (taskRepository *TaskRepository) UpdateTask(ctx context.Context, task model.Task) error {
	query := `
		UPDATE tasks
		SET name = $1, description = $2, priority = $3, difficulty = $4, status = $5, end_date = $6
		WHERE id = $7
	`

	cmdTag, err := taskRepository.Conn.Exec(ctx, query,
		task.Name,
		task.Description,
		task.Priority,
		task.Difficulty,
		task.Status,
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

func (taskRepository *TaskRepository) GetTaskByID(ctx context.Context, taskID int) (*model.Task, error) {
	query := `
		SELECT id, project_id, assignee_id, name, priority, difficulty, status, start_date, end_date
		FROM tasks
		WHERE id = $1
	`

	var task model.Task
	var startDate *time.Time
	var endDate *time.Time
	var assigneeID *int

	err := taskRepository.Conn.QueryRow(ctx, query, taskID).Scan(
		&task.ID,
		&task.ProjectID,
		&assigneeID,
		&task.Name,
		&task.Priority,
		&task.Difficulty,
		&task.Status,
		&startDate,
		&endDate,
	)
	if err != nil {
		return nil, err
	}

	task.AssigneeID = assigneeID
	task.StartDate = startDate
	if endDate != nil {
		task.EndDate = endDate
	}

	return &task, nil
}
