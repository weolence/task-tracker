# API Gateway

The single public entry point for all client requests. Validates JWT tokens by calling auth-service over gRPC, then forwards each request to the appropriate backend service — also over gRPC. Clients interact with one address and one protocol (HTTP REST); the gateway owns the translation to typed protobuf calls.

---

## Responsibilities

- Expose a unified HTTP REST API on port 8000
- Validate Bearer tokens on every protected route (via auth-service gRPC)
- Route auth-related requests to auth-service
- Route project/task/comment requests to project-service
- Enforce admin-only access on admin routes

---

## Internal Structure

```
api-gateway/
├── cmd/main/main.go                    # Entry point: registers all routes, starts HTTP server
├── internal/
│   ├── adapters/
│   │   ├── grpc/
│   │   │   └── clients.go             # gRPC client connections to auth + project services
│   │   └── http/
│   │       ├── auth_handler.go        # Register, Login, GetUserInfo, Admin user CRUD
│   │       ├── project_handler.go     # All project/task/comment routes
│   │       └── middleware.go          # AuthMiddleware (calls ValidateToken), AdminOnly guard
│   └── config/                        # YAML config loader
├── api/
│   └── openapi.yaml                   # Full OpenAPI 3.0 specification
└── configs/
    ├── config.docker.yaml
    └── config.local.yaml
```

---

## Request Flow

```
Client
  │
  │  HTTP  Authorization: Bearer <token>
  ▼
API Gateway (HTTP :8000)
  │
  ├─ AuthMiddleware
  │     └─► auth-service.ValidateToken(token)  [gRPC]
  │               └─ returns: user_id, role
  │
  ├─ Route match
  │
  ├─ /api/auth/*  ───────────────────────────► auth-service   [gRPC]
  │
  └─ /api/projects, /api/tasks, etc. ────────► project-service [gRPC]
                                                 (user_id + role in gRPC metadata)
```

---

## HTTP API Reference

Base URL: `http://localhost:8000`

### Public (no token required)

| Method | Path                  | Description                  |
|--------|-----------------------|------------------------------|
| POST   | `/api/auth/register`  | Register a new user          |
| POST   | `/api/auth/login`     | Login → returns JWT token    |

### Auth (require Bearer token)

| Method | Path                  | Description                     |
|--------|-----------------------|---------------------------------|
| GET    | `/api/auth/user-info` | Authenticated user's profile    |
| GET    | `/api/user-id`        | Current user's numeric ID       |

### Dashboard

| Method | Path            | Description                               |
|--------|-----------------|-------------------------------------------|
| GET    | `/api/dashboard`| Projects the user owns or is a member of  |

### Projects

| Method | Path                          | Description                            |
|--------|-------------------------------|----------------------------------------|
| POST   | `/api/projects`               | Create project (caller becomes manager)|
| GET    | `/api/projects/{id}`          | Get project's tasks                    |
| GET    | `/api/project-info`           | Project metadata                       |
| GET    | `/api/user-projects`          | Caller's projects                      |
| GET    | `/api/is-manager`             | Check if caller is project manager     |

### Project Members

| Method | Path                             | Description                  |
|--------|----------------------------------|------------------------------|
| GET    | `/api/project-members`           | Member user IDs              |
| GET    | `/api/project-members-details`   | Members with names + emails  |
| POST   | `/api/project-members/add`       | Add member by email (manager)|
| DELETE | `/api/project-members/remove`    | Remove member (manager)      |
| DELETE | `/api/project/leave`             | Leave project                |
| DELETE | `/api/project/delete`            | Delete project (manager)     |
| PUT    | `/api/project-manager/transfer`  | Transfer manager role        |

### Tasks

| Method | Path                        | Description                              |
|--------|-----------------------------|------------------------------------------|
| GET    | `/api/my-tasks`             | Tasks assigned to the caller             |
| GET    | `/api/project-tasks`        | All tasks in project (manager view)      |
| GET    | `/api/closed-project-tasks` | Closed tasks in project                  |
| POST   | `/api/tasks`                | Create task (manager)                    |
| PUT    | `/api/tasks/{id}/status`    | Advance task status                      |
| PUT    | `/api/tasks/{id}/close`     | Close task — must be ON_REVIEW (manager) |
| PUT    | `/api/tasks/{id}/assign`    | Assign task to user (manager)            |
| PUT    | `/api/tasks/{id}/unassign`  | Remove assignee (manager)                |
| DELETE | `/api/tasks/{id}`           | Delete task (manager)                    |

### Comments

| Method | Path                               | Description             |
|--------|------------------------------------|-------------------------|
| GET    | `/api/tasks/{id}/comments`         | List comments on task   |
| POST   | `/api/tasks/{id}/comments`         | Add comment             |
| PUT    | `/api/tasks/{id}/comments/{cid}`   | Edit own comment        |
| DELETE | `/api/tasks/{id}/comments/{cid}`   | Delete own comment      |

### Admin (require Bearer token + `role=admin`)

| Method | Path                                 | Description                   |
|--------|--------------------------------------|-------------------------------|
| GET    | `/api/admin/users/get`               | Get user by ID or email       |
| PUT    | `/api/admin/users`                   | Update any user               |
| DELETE | `/api/admin/users`                   | Delete any user               |
| GET    | `/api/admin/projects/get`            | Get project by ID or name     |
| PUT    | `/api/admin/projects`                | Update any project            |
| DELETE | `/api/admin/projects`                | Delete any project            |
| GET    | `/api/admin/tasks/get`               | Get task by ID                |
| PUT    | `/api/admin/tasks`                   | Update any task               |
| DELETE | `/api/admin/tasks`                   | Delete any task               |
| GET    | `/api/admin/comments/get`            | Get comment by ID             |
| PUT    | `/api/admin/comments`                | Update any comment            |
| DELETE | `/api/admin/comments`                | Delete any comment            |

The full request/response schemas for every endpoint are in [`api/openapi.yaml`](api/openapi.yaml).

---

## Middleware

### AuthMiddleware

Wraps any handler that requires a valid JWT.

```
1. Read Authorization header → extract Bearer token
2. Call auth-service.ValidateToken(token) via gRPC
3. On failure → 401 Unauthorized
4. On success → store user_id and role in request context
5. Pass context to the next handler
```

The user_id and role are then forwarded as gRPC metadata to downstream services:

```
user-id:   42
user-role: admin
```

### AdminOnly

Checks that the role stored in request context equals `"admin"`. Returns `403 Forbidden` otherwise. Applied on top of `AuthMiddleware`.

---

## Configuration

```yaml
http:
  addr: ":8000"
authService:
  grpcAddr: "auth-service:9090"
projectService:
  grpcAddr: "project-service:9091"
shutdownTimeout: 10s
```

### Environment Variables (Docker Compose)

| Variable                    | Value                    |
|-----------------------------|--------------------------|
| `AUTH_SERVICE_GRPC_ADDR`    | `auth-service:9090`      |
| `PROJECT_SERVICE_GRPC_ADDR` | `project-service:9091`   |

---

## Running Locally

```bash
cd app/backend/api-gateway

AUTH_SERVICE_GRPC_ADDR="localhost:9090" \
PROJECT_SERVICE_GRPC_ADDR="localhost:9091" \
  go run ./cmd/main
```

Auth-service must be running on `localhost:9090` and project-service on `localhost:9091` before the gateway starts.
