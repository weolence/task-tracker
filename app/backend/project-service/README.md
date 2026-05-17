# Project Service

Manages projects, tasks, comments, and team membership. Authorization is enforced at two levels simultaneously: the application layer checks the user's role from JWT context, and PostgreSQL enforces the same constraints through database roles and column-level privileges set per connection.

---

## Responsibilities

- Project lifecycle: create, update metadata, end, resume, delete
- Team membership: add/remove members, transfer manager role
- Task lifecycle: create → IN_WORK → ON_REVIEW → CLOSED
- Comments: full CRUD, scoped to tasks
- Admin operations: unrestricted CRUD on any entity

---

## Internal Structure

```
project-service/
├── cmd/main/main.go                    # Entry point: HTTP + gRPC servers
├── internal/
│   ├── core/
│   │   ├── domain/                     # Project, Task, Comment domain structs
│   │   ├── ports/                      # Repository interfaces
│   │   └── usecase/                    # ProjectUseCase, TaskUseCase, CommentUseCase
│   ├── adapters/
│   │   ├── http/
│   │   │   ├── project_handler.go      # Project + member endpoints
│   │   │   ├── task_handler.go         # Task lifecycle endpoints
│   │   │   ├── comment_handler.go      # Comment endpoints
│   │   │   ├── admin_handler.go        # Admin CRUD endpoints
│   │   │   └── middleware.go           # JWT middleware, role extraction
│   │   ├── grpc/
│   │   │   └── project_server.go       # gRPC ProjectService implementation
│   │   └── postgres/
│   │       ├── db.go                   # pgx pool + SET ROLE per-connection hook
│   │       ├── project_repository.go
│   │       ├── task_repository.go
│   │       └── comment_repository.go
│   ├── appctx/                         # Context key helpers (user_id, role)
│   └── config/                         # YAML config loader
├── api/proto/
│   ├── project_service.proto
│   ├── project.proto
│   └── user.proto
└── migrations/
    └── 20260516000100_init_schema.up.sql
```

### Layer Responsibilities

```
HTTP handler / gRPC server
        │
        ▼
  usecase layer  (business rules, orchestration)
        │
        ▼
  ports.Repository interfaces
        │
        ▼
  postgres adapters  (pgx queries)
        │
        ▼  BeforeAcquire: SET ROLE app_member|app_manager|app_admin
  PostgreSQL  (RLS + column-level grants enforce same rules at DB level)
```

---

## HTTP API

Base URL: `http://localhost:8081`

All endpoints require a valid JWT. The user's role is extracted from the token and used to both gate application logic and set the PostgreSQL role for the connection.

### Dashboard & User Info

| Method | Path              | Role needed | Description                        |
|--------|-------------------|-------------|------------------------------------|
| GET    | `/api/dashboard`  | member      | Projects the user owns or is in    |
| GET    | `/api/user-id`    | member      | Current user's ID                  |
| GET    | `/api/user-projects` | member   | Projects where user is a member    |
| GET    | `/api/my-tasks`   | member      | Tasks assigned to the current user |
| GET    | `/api/is-manager` | member      | Whether user is manager of a given project |

### Project Operations

| Method | Path                        | Role needed | Description                     |
|--------|-----------------------------|-------------|---------------------------------|
| GET    | `/api/project-info`         | member      | Project details                 |
| GET    | `/api/project-members`      | member      | Member user IDs                 |
| GET    | `/api/project-members-details` | member   | Members with email/name info    |
| POST   | `/api/projects`             | any         | Create project (creator → manager) |
| PUT    | `/api/project/update-meta`  | manager     | Update name/description         |
| PUT    | `/api/project/end`          | manager     | End project                     |
| PUT    | `/api/project/resume`       | manager     | Resume an ended project         |
| PUT    | `/api/project/delete`       | manager     | Delete project                  |
| DELETE | `/api/project/leave`        | member      | Leave a project                 |

### Member Management

| Method | Path                        | Role needed | Description              |
|--------|-----------------------------|-------------|--------------------------|
| PUT    | `/api/project-members/add`  | manager     | Add member by email      |
| PUT    | `/api/project-members/remove` | manager   | Remove member            |
| PUT    | `/api/project-manager/transfer` | manager | Transfer manager role    |

### Task Operations

| Method | Path                           | Role needed | Description                   |
|--------|--------------------------------|-------------|-------------------------------|
| GET    | `/api/projects/{id}`           | member      | All active tasks in project   |
| GET    | `/api/project-tasks`           | manager     | All tasks (manager view)      |
| GET    | `/api/closed-project-tasks`    | manager     | Closed tasks                  |
| POST   | `/api/tasks`                   | manager     | Create task                   |
| PUT    | `/api/tasks/{id}/status`       | member      | Advance task status           |
| PUT    | `/api/tasks/{id}/close`        | manager     | Close task (must be ON_REVIEW)|
| PUT    | `/api/tasks/{id}/assign`       | manager     | Assign task to user           |
| PUT    | `/api/tasks/{id}/unassign`     | manager     | Remove assignee               |
| DELETE | `/api/tasks/{id}`              | manager     | Delete task                   |

### Comment Operations

| Method | Path                               | Role needed | Description         |
|--------|------------------------------------|-------------|---------------------|
| GET    | `/api/tasks/{id}/comments`         | member      | List comments       |
| POST   | `/api/tasks/{id}/comments`         | member      | Add comment         |
| PUT    | `/api/tasks/{id}/comments/{cid}`   | member      | Edit own comment    |
| DELETE | `/api/tasks/{id}/comments/{cid}`   | member      | Delete own comment  |

### Admin Endpoints (require `role=admin`)

| Method | Path                                 | Description                      |
|--------|--------------------------------------|----------------------------------|
| GET    | `/api/admin/projects/get`            | Get project by ID or name        |
| PUT    | `/api/admin/projects`                | Update any project               |
| DELETE | `/api/admin/projects`                | Delete any project               |
| GET    | `/api/admin/tasks/get`               | Get task by ID                   |
| PUT    | `/api/admin/tasks`                   | Update any task                  |
| DELETE | `/api/admin/tasks`                   | Delete any task                  |
| GET    | `/api/admin/comments/get`            | Get comment by ID                |
| PUT    | `/api/admin/comments`                | Update any comment               |
| DELETE | `/api/admin/comments`                | Delete any comment               |
| GET    | `/api/admin/project-members-details` | Get member details for any project |
| POST   | `/api/admin/project-members/add`     | Add member to any project        |
| POST   | `/api/admin/project-members/remove`  | Remove member from any project   |

---

## gRPC API

Port: `9091`  
Proto package: `projectsvcv1`

User identity is passed through gRPC metadata headers:

```
user-id:   <int>
user-role: user | admin
```

The server maps these to the same use cases as the HTTP layer.

---

## Database Schema

Database: `project_db` (PostgreSQL 16, port 5433)

```
┌──────────────────────────────────────────────────────────────────────┐
│                            projects                                  │
├──────────────┬───────────────────────────────────────────────────────┤
│ id           │ SERIAL PRIMARY KEY                                    │
│ name         │ TEXT NOT NULL                                         │
│ description  │ TEXT NOT NULL                                         │
│ status       │ INT NOT NULL  (0 = IN_WORK, 1 = ENDED)               │
│ start_date   │ TIMESTAMP NOT NULL                                    │
│ end_date     │ TIMESTAMP  (nullable)                                 │
└──────────────┴───────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────┐
│                       project_user_roles                             │
├──────────────┬───────────────────────────────────────────────────────┤
│ id           │ SERIAL PRIMARY KEY                                    │
│ project_id   │ INT → projects(id) ON DELETE CASCADE                 │
│ user_id      │ INT NOT NULL                                          │
│ role         │ TEXT  CHECK (role IN ('manager', 'member'))          │
│              │ UNIQUE (project_id, user_id)                          │
└──────────────┴───────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────┐
│                         task_categories                              │
│            (supertype — shared by difficulty and priority)           │
├──────────────┬───────────────────────────────────────────────────────┤
│ id           │ SERIAL PRIMARY KEY                                    │
│ name         │ TEXT NOT NULL                                         │
│ category_type│ TEXT  CHECK (IN ('difficulty', 'priority'))          │
│              │ UNIQUE (category_type, name)                          │
└──────────────┴───────────────────────────────────────────────────────┘

 difficulty_categories (id → task_categories):  level 1=Easy 2=Medium 3=Hard
 priority_categories   (id → task_categories):  level 1=Low  2=Medium 3=High

┌──────────────────────────────────────────────────────────────────────┐
│                           task_statuses                              │
├──────────────┬───────────────────────────────────────────────────────┤
│ id           │ SERIAL PRIMARY KEY                                    │
│ name         │ TEXT UNIQUE  (Not Started / In Work / On Review / Closed) │
└──────────────┴───────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────┐
│                              tasks                                   │
├───────────────────────┬──────────────────────────────────────────────┤
│ id                    │ SERIAL PRIMARY KEY                           │
│ project_id            │ INT NOT NULL → projects(id)                  │
│ assignee_id           │ INT  (nullable)                              │
│ name                  │ TEXT NOT NULL                                │
│ description           │ TEXT  (nullable)                             │
│ difficulty_category_id│ INT → task_categories(id) ON DELETE SET NULL │
│ priority_category_id  │ INT → task_categories(id) ON DELETE SET NULL │
│ status_id             │ INT → task_statuses(id)                      │
│ start_date            │ TIMESTAMP  (set by trigger on IN_WORK)       │
│ end_date              │ TIMESTAMP  (set by trigger/function on CLOSED)|
└───────────────────────┴──────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────┐
│                            comments                                  │
├──────────────┬───────────────────────────────────────────────────────┤
│ id           │ SERIAL PRIMARY KEY                                    │
│ author_id    │ INT NOT NULL                                          │
│ task_id      │ INT → tasks(id) ON DELETE CASCADE                    │
│ content      │ TEXT NOT NULL                                         │
│ creation_date│ TIMESTAMP NOT NULL                                    │
└──────────────┴───────────────────────────────────────────────────────┘
```

### Entity Relationships

```
projects ──< project_user_roles (manager / member)
projects ──< tasks ──< comments
tasks ──── task_categories (difficulty, priority)
tasks ──── task_statuses
```

### Task Status Workflow

```
NOT_STARTED  ──► IN_WORK  ──► ON_REVIEW  ──► CLOSED
     ▲                                        (final, no way back)
     │
  (manager can reset status back to NOT_STARTED)
```

- Moving to `IN_WORK` automatically sets `start_date = NOW()` (trigger)
- Closing a task (`close_task()` stored function) validates the task is in `ON_REVIEW`, then sets `status = CLOSED` and `end_date = NOW()`

### Database Roles

The application connects as `app_service` and issues `SET ROLE` before each query:

| DB Role     | Mapped to         | What it can do                                                   |
|-------------|-------------------|------------------------------------------------------------------|
| app_member  | `role=user`       | Read projects/tasks; INSERT/UPDATE/DELETE own comments; UPDATE `status_id` and `assignee_id` on tasks |
| app_manager | `role=user` (manager of project) | Full INSERT/UPDATE/DELETE on projects, tasks, comments, project_user_roles |
| app_admin   | `role=admin`      | Unrestricted access to all tables                                |

### Database Views

| View                  | Used by    | Description                                   |
|-----------------------|------------|-----------------------------------------------|
| `view_member_tasks`   | members    | Active tasks (excludes Closed) with human-readable labels |
| `view_project_summary`| managers   | Projects with total/closed task counts        |
| `view_admin_tasks`    | admins     | All tasks joined with project names           |

---

## Configuration

```yaml
http:
  addr: ":8081"
grpc:
  addr: ":9091"
postgres:
  url: "postgres://..."
jwt:
  secret: "supersecret123"
admin:
  authServiceURL: "http://auth-service:8080"
shutdownTimeout: 10s
```

### Environment Variables (Docker Compose)

| Variable           | Value                                                          |
|--------------------|----------------------------------------------------------------|
| `DATABASE_URL`     | `postgres://postgres:postgres@project-db:5432/project_db?sslmode=disable` |
| `JWT_SECRET`       | `supersecret123`                                               |
| `AUTH_SERVICE_URL` | `http://auth-service:8080`                                     |
| `PORT`             | `8081`                                                         |

---

## Running Locally

```bash
cd app/backend/project-service

DATABASE_URL="postgres://postgres:postgres@localhost:5433/project_db?sslmode=disable" \
JWT_SECRET="supersecret123" \
AUTH_SERVICE_URL="http://localhost:8080" \
  go run ./cmd/main
```
