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
		name TEXT NOT NULL,
		description TEXT NOT NULL,
		status INT NOT NULL,
		start_date TIMESTAMP NOT NULL,
		end_date TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS project_user_roles (
		id SERIAL PRIMARY KEY,
		project_id INT NOT NULL,
		user_id INT NOT NULL,
		role TEXT NOT NULL CHECK (role IN ('manager', 'member')),
		UNIQUE (project_id, user_id),
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS task_categories (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		category_type TEXT NOT NULL CHECK (category_type IN ('group', 'difficulty', 'priority')),
		parent_id INT REFERENCES task_categories(id) ON DELETE SET NULL,
		UNIQUE (category_type, name)
	);

	CREATE TABLE IF NOT EXISTS task_statuses (
		id   SERIAL PRIMARY KEY,
		name TEXT   NOT NULL UNIQUE
	);

	CREATE TABLE IF NOT EXISTS tasks (
		id SERIAL PRIMARY KEY,
		project_id INT NOT NULL,
		assignee_id INT,
		name TEXT NOT NULL,
		description TEXT,
		difficulty_category_id INT,
		priority_category_id INT,
		status_id INT NOT NULL REFERENCES task_statuses(id),
		start_date TIMESTAMP,
		end_date TIMESTAMP,
		FOREIGN KEY (difficulty_category_id) REFERENCES task_categories(id) ON DELETE SET NULL,
		FOREIGN KEY (priority_category_id) REFERENCES task_categories(id) ON DELETE SET NULL
	);

	CREATE TABLE IF NOT EXISTS comments (
		id SERIAL PRIMARY KEY,
		author_id INT NOT NULL,
		task_id INT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		content TEXT NOT NULL,
		creation_date TIMESTAMP NOT NULL
	);

	INSERT INTO task_statuses (name) VALUES
	('Not Started'),
	('In Work'),
	('On Review'),
	('Closed')
	ON CONFLICT (name) DO NOTHING;

	INSERT INTO task_categories (name, category_type, parent_id)
	VALUES ('Difficulty', 'group', NULL)
	ON CONFLICT (category_type, name) DO NOTHING;

	INSERT INTO task_categories (name, category_type, parent_id)
	SELECT 'Easy', 'difficulty', id FROM task_categories WHERE category_type = 'group' AND name = 'Difficulty'
	ON CONFLICT (category_type, name) DO NOTHING;

	INSERT INTO task_categories (name, category_type, parent_id)
	SELECT 'Medium', 'difficulty', id FROM task_categories WHERE category_type = 'group' AND name = 'Difficulty'
	ON CONFLICT (category_type, name) DO NOTHING;

	INSERT INTO task_categories (name, category_type, parent_id)
	SELECT 'Hard', 'difficulty', id FROM task_categories WHERE category_type = 'group' AND name = 'Difficulty'
	ON CONFLICT (category_type, name) DO NOTHING;

	INSERT INTO task_categories (name, category_type, parent_id)
	VALUES ('Priority', 'group', NULL)
	ON CONFLICT (category_type, name) DO NOTHING;

	INSERT INTO task_categories (name, category_type, parent_id)
	SELECT 'Low', 'priority', id FROM task_categories WHERE category_type = 'group' AND name = 'Priority'
	ON CONFLICT (category_type, name) DO NOTHING;

	INSERT INTO task_categories (name, category_type, parent_id)
	SELECT 'Medium', 'priority', id FROM task_categories WHERE category_type = 'group' AND name = 'Priority'
	ON CONFLICT (category_type, name) DO NOTHING;

	INSERT INTO task_categories (name, category_type, parent_id)
	SELECT 'High', 'priority', id FROM task_categories WHERE category_type = 'group' AND name = 'Priority'
	ON CONFLICT (category_type, name) DO NOTHING;

	CREATE INDEX IF NOT EXISTS comments_task_id_idx              ON comments(task_id);
	CREATE INDEX IF NOT EXISTS comments_author_id_idx            ON comments(author_id);
	CREATE INDEX IF NOT EXISTS project_user_roles_project_id_idx ON project_user_roles(project_id);
	CREATE INDEX IF NOT EXISTS project_user_roles_user_id_idx    ON project_user_roles(user_id);
	CREATE INDEX IF NOT EXISTS project_user_roles_role_idx       ON project_user_roles(role);
	CREATE INDEX IF NOT EXISTS tasks_status_id_idx               ON tasks(status_id);
	`)
	return err
}
