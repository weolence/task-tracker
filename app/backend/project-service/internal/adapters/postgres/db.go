package postgres

import (
	"context"
	"fmt"

	"project-service/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("create pgxpool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return pool, nil
}

func InitSchema(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
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

	CREATE TABLE IF NOT EXISTS comments (
		id SERIAL PRIMARY KEY,
		author_id INT NOT NULL,
		task_id INT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		content TEXT NOT NULL,
		creation_date TIMESTAMP NOT NULL
	);

	CREATE INDEX IF NOT EXISTS comments_task_id_idx ON comments(task_id);
	CREATE INDEX IF NOT EXISTS comments_author_id_idx ON comments(author_id);
	`)
	return err
}
