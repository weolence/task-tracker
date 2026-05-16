CREATE TABLE IF NOT EXISTS projects (
    id          SERIAL PRIMARY KEY,
    manager_id  INT       NOT NULL,
    name        TEXT      NOT NULL,
    description TEXT      NOT NULL,
    status      INT       NOT NULL,
    start_date  TIMESTAMP NOT NULL,
    end_date    TIMESTAMP
);

CREATE TABLE IF NOT EXISTS project_members (
    project_id INT NOT NULL,
    user_id    INT NOT NULL,
    PRIMARY KEY (project_id, user_id)
);

CREATE TABLE IF NOT EXISTS tasks (
    id          SERIAL PRIMARY KEY,
    project_id  INT       NOT NULL,
    assignee_id INT,
    name        TEXT      NOT NULL,
    description TEXT,
    priority    INT       NOT NULL,
    difficulty  INT       NOT NULL,
    status      INT       NOT NULL,
    start_date  TIMESTAMP,
    end_date    TIMESTAMP
);

CREATE TABLE IF NOT EXISTS comments (
    id            SERIAL PRIMARY KEY,
    author_id     INT       NOT NULL,
    task_id       INT       NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    content       TEXT      NOT NULL,
    creation_date TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS comments_task_id_idx   ON comments(task_id);
CREATE INDEX IF NOT EXISTS comments_author_id_idx ON comments(author_id);
