package postgres

import (
	"context"
	"fmt"

	"project-service/internal/appctx"
	"project-service/internal/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse pgxpool config: %w", err)
	}

	// Switch the DB session role to the role stored in the request context.
	// This enforces DB-level access control on top of application-level middleware.
	poolConfig.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
		dbRole := appctx.GetDBRole(ctx)
		if dbRole == "" {
			return true
		}
		switch dbRole {
		case "app_member", "app_manager", "app_admin":
		default:
			return false
		}
		if _, err := conn.Exec(ctx, "SET ROLE "+dbRole); err != nil {
			return false
		}
		return true
	}

	// Always reset the role when a connection is returned to the pool.
	poolConfig.AfterRelease = func(conn *pgx.Conn) bool {
		if _, err := conn.Exec(context.Background(), "RESET ROLE"); err != nil {
			return false
		}
		return true
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
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

	-- Supertype: common category attributes shared by all category subtypes
	CREATE TABLE IF NOT EXISTS task_categories (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		category_type TEXT NOT NULL CHECK (category_type IN ('difficulty', 'priority')),
		UNIQUE (category_type, name)
	);

	-- Subtype: difficulty ratings (Easy=1, Medium=2, Hard=3)
	CREATE TABLE IF NOT EXISTS difficulty_categories (
		id    INT PRIMARY KEY REFERENCES task_categories(id) ON DELETE CASCADE,
		level INT NOT NULL CHECK (level BETWEEN 1 AND 3)
	);

	-- Subtype: priority ratings (Low=1, Medium=2, High=3)
	CREATE TABLE IF NOT EXISTS priority_categories (
		id    INT PRIMARY KEY REFERENCES task_categories(id) ON DELETE CASCADE,
		level INT NOT NULL CHECK (level BETWEEN 1 AND 3)
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

	INSERT INTO task_categories (name, category_type)
	VALUES ('Easy', 'difficulty'), ('Medium', 'difficulty'), ('Hard', 'difficulty')
	ON CONFLICT (category_type, name) DO NOTHING;

	INSERT INTO difficulty_categories (id, level)
	SELECT id, 1 FROM task_categories WHERE category_type = 'difficulty' AND name = 'Easy'
	ON CONFLICT (id) DO NOTHING;
	INSERT INTO difficulty_categories (id, level)
	SELECT id, 2 FROM task_categories WHERE category_type = 'difficulty' AND name = 'Medium'
	ON CONFLICT (id) DO NOTHING;
	INSERT INTO difficulty_categories (id, level)
	SELECT id, 3 FROM task_categories WHERE category_type = 'difficulty' AND name = 'Hard'
	ON CONFLICT (id) DO NOTHING;

	INSERT INTO task_categories (name, category_type)
	VALUES ('Low', 'priority'), ('Medium', 'priority'), ('High', 'priority')
	ON CONFLICT (category_type, name) DO NOTHING;

	INSERT INTO priority_categories (id, level)
	SELECT id, 1 FROM task_categories WHERE category_type = 'priority' AND name = 'Low'
	ON CONFLICT (id) DO NOTHING;
	INSERT INTO priority_categories (id, level)
	SELECT id, 2 FROM task_categories WHERE category_type = 'priority' AND name = 'Medium'
	ON CONFLICT (id) DO NOTHING;
	INSERT INTO priority_categories (id, level)
	SELECT id, 3 FROM task_categories WHERE category_type = 'priority' AND name = 'High'
	ON CONFLICT (id) DO NOTHING;

	CREATE INDEX IF NOT EXISTS comments_task_id_idx              ON comments(task_id);
	CREATE INDEX IF NOT EXISTS comments_author_id_idx            ON comments(author_id);
	CREATE INDEX IF NOT EXISTS project_user_roles_project_id_idx ON project_user_roles(project_id);
	CREATE INDEX IF NOT EXISTS project_user_roles_user_id_idx    ON project_user_roles(user_id);
	CREATE INDEX IF NOT EXISTS project_user_roles_role_idx       ON project_user_roles(role);
	CREATE INDEX IF NOT EXISTS tasks_status_id_idx               ON tasks(status_id);

	DO $role$
	BEGIN
	    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'app_member') THEN
	        CREATE ROLE app_member;
	    END IF;
	    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'app_manager') THEN
	        CREATE ROLE app_manager;
	    END IF;
	    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'app_admin') THEN
	        CREATE ROLE app_admin;
	    END IF;
	END $role$;

	CREATE OR REPLACE VIEW view_member_tasks AS
	SELECT
	    t.id, t.project_id, t.assignee_id, t.name, t.description,
	    tc_diff.name AS difficulty,
	    tc_prio.name AS priority,
	    ts.name      AS status,
	    t.start_date, t.end_date
	FROM tasks t
	JOIN task_statuses ts    ON ts.id = t.status_id
	LEFT JOIN task_categories tc_diff ON tc_diff.id = t.difficulty_category_id
	LEFT JOIN task_categories tc_prio ON tc_prio.id = t.priority_category_id
	WHERE ts.name != 'Closed';

	CREATE OR REPLACE VIEW view_project_summary AS
	SELECT
	    p.id, p.name, p.description, p.status, p.start_date, p.end_date,
	    pur.user_id                                    AS manager_id,
	    COUNT(t.id)                                    AS total_tasks,
	    COUNT(CASE WHEN ts.name = 'Closed' THEN 1 END) AS closed_tasks
	FROM projects p
	LEFT JOIN project_user_roles pur ON pur.project_id = p.id AND pur.role = 'manager'
	LEFT JOIN tasks t                ON t.project_id   = p.id
	LEFT JOIN task_statuses ts       ON ts.id          = t.status_id
	GROUP BY p.id, p.name, p.description, p.status, p.start_date, p.end_date, pur.user_id;

	CREATE OR REPLACE VIEW view_admin_tasks AS
	SELECT
	    t.id, t.project_id, p.name AS project_name,
	    t.assignee_id, t.name, t.description,
	    tc_diff.name AS difficulty,
	    tc_prio.name AS priority,
	    ts.name      AS status,
	    t.start_date, t.end_date
	FROM tasks t
	JOIN projects p           ON p.id  = t.project_id
	JOIN task_statuses ts     ON ts.id = t.status_id
	LEFT JOIN task_categories tc_diff ON tc_diff.id = t.difficulty_category_id
	LEFT JOIN task_categories tc_prio ON tc_prio.id = t.priority_category_id;

	GRANT SELECT ON view_member_tasks, view_project_summary,
	    projects, project_user_roles, task_statuses,
	    task_categories, difficulty_categories, priority_categories TO app_member;
	GRANT SELECT, INSERT, UPDATE, DELETE ON comments TO app_member;
	GRANT SELECT ON tasks TO app_member;
	GRANT UPDATE (status_id, assignee_id, start_date, end_date) ON tasks TO app_member;
	GRANT USAGE, SELECT ON SEQUENCE comments_id_seq TO app_member;

	GRANT SELECT ON view_member_tasks, view_project_summary,
	    project_user_roles, task_statuses,
	    task_categories, difficulty_categories, priority_categories TO app_manager;
	GRANT SELECT, INSERT, UPDATE, DELETE ON projects           TO app_manager;
	GRANT SELECT, INSERT, UPDATE, DELETE ON comments           TO app_manager;
	GRANT SELECT, INSERT, UPDATE, DELETE ON tasks              TO app_manager;
	GRANT SELECT, INSERT, UPDATE, DELETE ON project_user_roles TO app_manager;
	GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO app_manager;

	GRANT SELECT ON view_member_tasks, view_project_summary, view_admin_tasks TO app_admin;
	GRANT SELECT, INSERT, UPDATE, DELETE ON
	    projects, tasks, comments, project_user_roles, task_statuses,
	    task_categories, difficulty_categories, priority_categories TO app_admin;
	GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO app_admin;

	CREATE OR REPLACE FUNCTION close_task(p_task_id INT)
	RETURNS VOID LANGUAGE plpgsql AS $$
	DECLARE
	    v_current_status_id INT;
	    v_review_status_id  INT;
	    v_closed_status_id  INT;
	BEGIN
	    SELECT id INTO v_review_status_id FROM task_statuses WHERE name = 'On Review';
	    SELECT id INTO v_closed_status_id FROM task_statuses WHERE name = 'Closed';
	    SELECT status_id INTO v_current_status_id FROM tasks WHERE id = p_task_id;
	    IF v_current_status_id IS NULL THEN
	        RAISE EXCEPTION 'Task % not found', p_task_id;
	    END IF;
	    IF v_current_status_id != v_review_status_id THEN
	        RAISE EXCEPTION 'Task % must be in On Review status to be closed', p_task_id;
	    END IF;
	    UPDATE tasks SET status_id = v_closed_status_id, end_date = NOW() WHERE id = p_task_id;
	END;
	$$;

	GRANT EXECUTE ON FUNCTION close_task(INT) TO app_manager, app_admin;

	CREATE OR REPLACE FUNCTION fn_task_status_dates()
	RETURNS TRIGGER LANGUAGE plpgsql AS $$
	BEGIN
	    IF NEW.status_id IS DISTINCT FROM OLD.status_id THEN
	        IF NEW.status_id = (SELECT id FROM task_statuses WHERE name = 'In Work')
	           AND NEW.start_date IS NULL THEN
	            NEW.start_date := NOW();
	        END IF;
	        IF NEW.status_id = (SELECT id FROM task_statuses WHERE name = 'Closed') THEN
	            NEW.end_date := NOW();
	        END IF;
	        IF NEW.status_id = (SELECT id FROM task_statuses WHERE name = 'Not Started') THEN
	            NEW.end_date := NULL;
	        END IF;
	    END IF;
	    RETURN NEW;
	END;
	$$;

	DROP TRIGGER IF EXISTS trg_task_status_dates ON tasks;
	CREATE TRIGGER trg_task_status_dates
	    BEFORE UPDATE ON tasks
	    FOR EACH ROW
	    EXECUTE FUNCTION fn_task_status_dates();

	DO $$
	BEGIN
	    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'app_service') THEN
	        CREATE ROLE app_service WITH LOGIN PASSWORD 'CHANGE_ME'
	            NOSUPERUSER NOCREATEDB NOCREATEROLE;
	    END IF;
	END $$;

	GRANT app_member, app_manager, app_admin TO app_service;
	GRANT USAGE ON SCHEMA public TO app_service;
	`)
	return err
}
