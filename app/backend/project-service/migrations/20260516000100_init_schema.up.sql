CREATE TABLE IF NOT EXISTS projects (
    id          SERIAL PRIMARY KEY,
    name        TEXT      NOT NULL,
    description TEXT      NOT NULL,
    status      INT       NOT NULL,
    start_date  TIMESTAMP NOT NULL,
    end_date    TIMESTAMP
);

CREATE TABLE IF NOT EXISTS project_user_roles (
    id         SERIAL PRIMARY KEY,
    project_id INT  NOT NULL,
    user_id    INT  NOT NULL,
    role       TEXT NOT NULL CHECK (role IN ('manager', 'member')),
    UNIQUE (project_id, user_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- Supertype: common category attributes shared by all category subtypes
CREATE TABLE IF NOT EXISTS task_categories (
    id            SERIAL PRIMARY KEY,
    name          TEXT NOT NULL,
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
    id                     SERIAL PRIMARY KEY,
    project_id             INT       NOT NULL,
    assignee_id            INT,
    name                   TEXT      NOT NULL,
    description            TEXT,
    difficulty_category_id INT,
    priority_category_id   INT,
    status_id              INT       NOT NULL REFERENCES task_statuses(id),
    start_date             TIMESTAMP,
    end_date               TIMESTAMP,
    FOREIGN KEY (difficulty_category_id) REFERENCES task_categories(id) ON DELETE SET NULL,
    FOREIGN KEY (priority_category_id) REFERENCES task_categories(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS comments (
    id            SERIAL PRIMARY KEY,
    author_id     INT       NOT NULL,
    task_id       INT       NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    content       TEXT      NOT NULL,
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

-- Database roles following the principle of least privilege
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

-- External schema for project members: active tasks with human-readable labels
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

-- External schema for project managers: project overview with task statistics
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

-- External schema for administrators: full task information joined with project names
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

-- Privilege grants: members can read project/task data, manage comments,
-- advance task status, and self-assign unassigned tasks.
GRANT SELECT ON view_member_tasks, view_project_summary,
    projects, project_user_roles, task_statuses,
    task_categories, difficulty_categories, priority_categories TO app_member;
GRANT SELECT, INSERT, UPDATE, DELETE ON comments TO app_member;
-- Direct SELECT on tasks is required for application queries (e.g. GetTaskByID).
-- The view view_member_tasks provides the logical external schema; this grant
-- supports the internal query patterns without broadening write privileges.
GRANT SELECT ON tasks TO app_member;
-- Column-level UPDATE: members may advance task status and self-assign tasks.
-- They cannot modify task name, description, priority, difficulty, or project.
GRANT UPDATE (status_id, assignee_id, start_date, end_date) ON tasks TO app_member;
GRANT USAGE, SELECT ON SEQUENCE comments_id_seq TO app_member;

-- Managers: full project and task management
GRANT SELECT ON view_member_tasks, view_project_summary,
    project_user_roles, task_statuses,
    task_categories, difficulty_categories, priority_categories TO app_manager;
GRANT SELECT, INSERT, UPDATE, DELETE ON projects           TO app_manager;
GRANT SELECT, INSERT, UPDATE, DELETE ON comments           TO app_manager;
GRANT SELECT, INSERT, UPDATE, DELETE ON tasks              TO app_manager;
GRANT SELECT, INSERT, UPDATE, DELETE ON project_user_roles TO app_manager;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO app_manager;

-- Admins: unrestricted access to all project data
GRANT SELECT ON view_member_tasks, view_project_summary, view_admin_tasks TO app_admin;
GRANT SELECT, INSERT, UPDATE, DELETE ON
    projects, tasks, comments, project_user_roles, task_statuses,
    task_categories, difficulty_categories, priority_categories TO app_admin;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO app_admin;

-- Stored function: close a task with business rule validation
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

-- Trigger function: automatically manage task start/end dates on status transitions
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

-- Dedicated application service account (non-superuser).
-- The application must connect as app_service, never as the PostgreSQL
-- superuser (postgres). This satisfies the requirement that the DBA role
-- must not hold unrestricted DBMS privileges.
-- Replace 'CHANGE_ME' with a strong password before running in production.
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'app_service') THEN
        CREATE ROLE app_service WITH LOGIN PASSWORD 'CHANGE_ME'
            NOSUPERUSER NOCREATEDB NOCREATEROLE;
    END IF;
END $$;

-- Grant the service account the ability to SET ROLE into each application role.
-- Without this, SET ROLE in BeforeAcquire (db.go) would be rejected.
GRANT app_member  TO app_service;
GRANT app_manager TO app_service;
GRANT app_admin   TO app_service;

-- Schema-level access needed before any SET ROLE takes effect.
GRANT USAGE ON SCHEMA public TO app_service;

-- app_service itself holds no direct table privileges; all data access is
-- gated behind SET ROLE, enforcing least privilege at the connection level.
