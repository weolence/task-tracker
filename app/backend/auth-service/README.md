# Auth Service

Handles user registration, authentication, JWT token issuance and validation, and user management. Exposes both an HTTP API (for the browser-facing admin panel) and a gRPC API (for internal service-to-service communication).

---

## Responsibilities

- Register new users (bcrypt-hashed passwords)
- Login and issue signed JWT tokens (HS256)
- Validate tokens on behalf of the API gateway
- Return user profile information
- Admin panel: manage users, proxy admin requests to project-service

---

## Internal Structure

```
auth-service/
├── cmd/main/main.go              # Entry point: HTTP + gRPC servers, interactive console
├── internal/
│   ├── core/
│   │   ├── domain/               # User struct — pure domain model
│   │   ├── ports/                # UserRepository interface
│   │   └── usecase/              # AuthUseCase: Register, Login, ValidateToken, ...
│   ├── adapters/
│   │   ├── http/
│   │   │   ├── auth_handler.go   # Login, ValidateToken, GetUserInfo
│   │   │   ├── admin_handler.go  # Admin CRUD + project-service proxy
│   │   │   └── middleware.go     # JWT auth middleware, AdminOnly guard
│   │   ├── grpc/
│   │   │   └── auth_server.go    # gRPC AuthService implementation
│   │   └── postgres/
│   │       ├── db.go             # pgx connection pool
│   │       └── user_repository.go # UserRepository (PostgreSQL)
│   ├── config/                   # YAML config loader
│   └── testhelper/               # In-memory mock repository for unit tests
├── api/proto/
│   ├── auth_service.proto        # gRPC service definition
│   └── user.proto                # User message type
├── migrations/
│   └── 20260515000100_init_schema.up.sql
└── configs/
    ├── config.docker.yaml
    └── config.local.yaml
```

### Layer Responsibilities

```
cmd/main ──► uses ──► usecase (AuthUseCase)
                         │
                    implements
                         │
                    ports.UserRepository
                         │
                    implemented by
                         │
              adapters/postgres.UserRepository

HTTP handler ──► AuthUseCase ──► UserRepository ──► PostgreSQL
gRPC server  ──► AuthUseCase ──► UserRepository ──► PostgreSQL
```

---

## HTTP API

Base URL: `http://localhost:8080`

### Public Endpoints

| Method | Path              | Description                         |
|--------|-------------------|-------------------------------------|
| GET    | `/`               | Serves `index.html` (login page)    |
| POST   | `/login`          | Login with email + password → JWT   |
| POST   | `/validate-token` | Validate a JWT token                |

### Protected Endpoints (require valid JWT)

| Method | Path         | Description              |
|--------|--------------|--------------------------|
| GET    | `/user-info`  | Return authenticated user's profile |

### Admin Endpoints (require JWT + `role=admin`)

| Method | Path                             | Description                              |
|--------|----------------------------------|------------------------------------------|
| GET    | `/admin`                         | Serves `admin.html`                      |
| GET    | `/admin/api/users/get`           | Get user by ID or email                  |
| PUT    | `/admin/api/users`               | Update user fields                       |
| DELETE | `/admin/api/users`               | Delete user by ID                        |
| GET    | `/admin/api/projects/get`        | Proxy → project-service admin            |
| PUT    | `/admin/api/projects`            | Proxy → project-service admin            |
| GET    | `/admin/api/tasks/get`           | Proxy → project-service admin            |
| PUT    | `/admin/api/tasks`               | Proxy → project-service admin            |
| DELETE | `/admin/api/tasks`               | Proxy → project-service admin            |
| GET    | `/admin/api/comments/get`        | Proxy → project-service admin            |
| PUT    | `/admin/api/comments`            | Proxy → project-service admin            |
| DELETE | `/admin/api/comments`            | Proxy → project-service admin            |
| GET    | `/admin/api/project-members-details` | Proxy → project-service admin        |
| POST   | `/admin/api/project-members/add`    | Proxy → project-service admin         |
| POST   | `/admin/api/project-members/remove` | Proxy → project-service admin         |

---

## gRPC API

Port: `9090`  
Proto package: `authsvcv1`

```protobuf
service AuthService {
  rpc Register(RegisterRequest)           returns (OperationResponse);
  rpc Login(LoginRequest)                 returns (LoginResponse);
  rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);
  rpc GetUserInfo(GetUserRequest)         returns (User);
  rpc GetUser(GetUserRequest)             returns (User);          // admin
  rpc UpdateUser(UpdateUserRequest)       returns (OperationResponse); // admin
  rpc DeleteUser(DeleteUserRequest)       returns (OperationResponse); // admin
}
```

The API gateway calls `ValidateToken` on every protected request and uses the returned `user_id` + `role` to populate gRPC metadata sent downstream to project-service.

---

## Database Schema

Database: `auth_db` (PostgreSQL 16, port 5432)

```
┌─────────────────────────────────────────────┐
│                   users                     │
├──────────┬──────────────────────────────────┤
│ id       │ SERIAL PRIMARY KEY               │
│ email    │ TEXT UNIQUE NOT NULL             │
│ name     │ TEXT NOT NULL                    │
│ surname  │ TEXT NOT NULL                    │
│ password │ TEXT NOT NULL  (bcrypt hash)     │
│ role     │ TEXT DEFAULT 'user'              │
│          │ CHECK (role IN ('user','admin'))  │
└──────────┴──────────────────────────────────┘

view_user_profile  →  id, email, name, surname, role  (no password)
```

### Database Roles

| Role       | Privileges                                    |
|------------|-----------------------------------------------|
| auth_user  | SELECT on `view_user_profile` only            |
| auth_admin | Full SELECT/INSERT/UPDATE/DELETE on `users`   |

---

## Configuration

Config is loaded from a YAML file. Path resolution order:

1. `--config` CLI flag
2. `AUTH_CONFIG_PATH` environment variable
3. Default (`configs/config.local.yaml` or `configs/config.docker.yaml`)

### Key Config Fields

```yaml
http:
  addr: ":8080"
grpc:
  addr: ":9090"
postgres:
  url: "postgres://..."
jwt:
  secret: "supersecret123"
admin:
  projectServiceURL: "http://project-service:8081"
staticDir: "static"
shutdownTimeout: 10s
```

### Environment Variables (Docker Compose)

| Variable              | Value (docker-compose)                                    |
|-----------------------|-----------------------------------------------------------|
| `DATABASE_URL`        | `postgres://postgres:postgres@auth-db:5432/auth_db?sslmode=disable` |
| `JWT_SECRET`          | `supersecret123`                                          |
| `PROJECT_SERVICE_URL` | `http://project-service:8081`                             |
| `PORT`                | `8080`                                                    |

---

## Interactive Console

When `stdin` is a terminal (i.e. running locally, not in a pipe), auth-service starts an interactive management console:

```
Commands: register | delete | set-role | help | exit
> register
Email: alice@example.com
Password: hunter2
Name: Alice
Surname: Smith
user registered successfully
> set-role
Email: alice@example.com
Role (user/admin): admin
user role updated successfully
> exit
Shutting down...
```

In Docker, `stdin_open: true` and `tty: true` are set in `docker-compose.yml` so you can attach with `docker attach`.

---

## Running Locally

```bash
cd app/backend/auth-service

# Ensure auth-db is running (e.g. via docker compose)
# Apply migrations
DATABASE_URL="postgres://postgres:postgres@localhost:5432/auth_db?sslmode=disable" \
  go run ./cmd/migrate   # or use the Dockerfile.migrate approach

# Run the service
DATABASE_URL="postgres://postgres:postgres@localhost:5432/auth_db?sslmode=disable" \
JWT_SECRET="supersecret123" \
PROJECT_SERVICE_URL="http://localhost:8081" \
  go run ./cmd/main
```
