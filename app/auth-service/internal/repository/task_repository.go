package repository

import (
	"auth-service/internal/model"
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type TaskRepository struct {
	Conn *pgx.Conn
}

func TaskRepositoryCreate(ctx context.Context, dbLink string) (*TaskRepository, error) {
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
		assignee_id INT NOT NULL,
		name TEXT NOT NULL,
		priority INT NOT NULL,
		difficulty INT NOT NULL,
		status INT NOT NULL,
		start_date TIMESTAMP NOT NULL,
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

func (taskRepository *TaskRepository) CreateTask(ctx context.Context, task *model.Task) error {
	query := `
		INSERT INTO tasks (project_id, assignee_id, name, priority, difficulty, status, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	return taskRepository.Conn.QueryRow(ctx, query,
		task.ProjectID,
		task.AssigneeID,
		task.Name,
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
		SET status = $1
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
