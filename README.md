# Task Tracker

A microservice-based project and task management system. Teams can create projects, assign tasks, track progress through a status workflow, and leave comments — all secured via JWT authentication and enforced through database-level role privileges.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Client (Browser)                           │
└───────────────────────────────┬─────────────────────────────────────┘
                                │ HTTP :8000
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          API Gateway                                │
│             Validates JWT · Routes requests via gRPC                │
└──────────────┬──────────────────────────────┬───────────────────────┘
               │ gRPC :9090                   │ gRPC :9091
               ▼                              ▼
┌──────────────────────────┐   ┌──────────────────────────────────────┐
│       Auth Service       │   │           Project Service            │
│  HTTP :8080 / gRPC :9090 │   │       HTTP :8081 / gRPC :9091        │
│                          │   │                                      │
│  • Register / Login      │   │  • Projects CRUD                     │
│  • JWT issue/validate    │   │  • Tasks lifecycle                   │
│  • Admin panel (HTML)    │   │  • Comments                          │
│  • Role management       │   │  • Member management                 │
└────────────┬─────────────┘   └─────────────────┬────────────────────┘
             │                                   │
             ▼                                   ▼
    ┌─────────────────┐                 ┌─────────────────┐
    │   auth_db       │                 │   project_db    │
    │  PostgreSQL :5432                 │  PostgreSQL :5433
    └─────────────────┘                 └─────────────────┘
```

### Service Map

| Service         | HTTP port | gRPC port | Database         |
|-----------------|-----------|-----------|------------------|
| api-gateway     | 8000      | —         | —                |
| auth-service    | 8080      | 9090      | auth_db (:5432)  |
| project-service | 8081      | 9091      | project_db (:5433) |

### Internal Communication

Services never talk to each other over HTTP. The API gateway holds gRPC client connections to both backend services and forwards all requests through typed protobuf contracts. The auth-service admin panel is the only place that uses HTTP-level proxying (from auth-service → project-service) and only for admin operations.

```
Client ──HTTP──► API Gateway ──gRPC──► Auth Service
                             ──gRPC──► Project Service
Auth Service admin panel ──HTTP──► Project Service (admin proxy only)
```

### Authentication Flow

```
1. POST /api/auth/login  →  JWT token returned
2. All subsequent requests include:  Authorization: Bearer <token>
3. API Gateway calls auth-service.ValidateToken(token) via gRPC
4. On success: user_id + role are forwarded to project-service via gRPC metadata
5. Project-service issues SET ROLE app_member|app_manager|app_admin
   to the PostgreSQL connection before executing any query
```

---

## Technology Stack

| Layer        | Technology              |
|--------------|-------------------------|
| Language     | Go 1.26                 |
| Web          | stdlib `net/http`       |
| RPC          | gRPC + Protocol Buffers |
| Database     | PostgreSQL 16           |
| Auth         | JWT (HS256)             |
| E2E Testing  | Cypress 13              |
| Seed / Tests | Node.js 22              |
| Containers   | Docker / Docker Compose |

---

## Running the Project

### Prerequisites

- Docker and Docker Compose

### Start Everything

```bash
cd app
docker compose up
```

This starts all six containers in order:

1. `auth-db` — PostgreSQL for auth data
2. `auth-migrate` — runs schema migration on auth-db, then exits
3. `project-db` — PostgreSQL for project/task data
4. `auth-service` — HTTP :8080, gRPC :9090
5. `project-service` — HTTP :8081, gRPC :9091
6. `api-gateway` — HTTP :8000 (public entry point)
7. `seed` — populates both databases with demo data, then exits

> **Note:** Database storage is in-memory (`tmpfs`). All data is lost when the containers stop. This is intentional for development — re-running `docker compose up` starts fresh.

### Stop

```bash
docker compose down
```

---

## Seeding Demo Data

The seed container runs automatically on `docker compose up`. If you need to re-seed manually (e.g. after the containers are already running), run:

```bash
# from the app/ directory
cd app
node e2e-tests/scripts/seed.js
```

Or using npm:

```bash
cd app/e2e-tests
npm install
npm run seed
```

### What the Seed Creates

**Users:**

| Role    | Email               | Password    |
|---------|---------------------|-------------|
| admin   | admin@demo.local    | Admin1234!  |
| manager | alice@demo.local    | Alice1234!  |
| member  | bob@demo.local      | Bob12345!   |
| member  | carol@demo.local    | Carol123!   |

**Projects:**

```
Project 1 — "E-Commerce Platform"  (alice is manager, bob + carol are members)
  ├── "Design product landing page"     → Bob   → IN_WORK
  ├── "Integrate payment gateway"       → Carol → ON_REVIEW
  ├── "Build product catalog API"       → Bob   → CLOSED
  └── "Set up CI/CD pipeline"           →  —    → NOT_STARTED

Project 2 — "Mobile App MVP"  (alice is manager, carol is member)
  ├── "Implement authentication screen" → Carol → IN_WORK
  └── "Add push notification support"   →  —    → NOT_STARTED
```

### Seed Environment Variables

All variables are optional. Defaults match the Docker Compose setup.

| Variable       | Default               | Description             |
|----------------|-----------------------|-------------------------|
| `API_URL`      | `http://localhost:8000` | API gateway address   |
| `AUTH_DB_HOST` | `localhost`           | auth-db host            |
| `AUTH_DB_PORT` | `5432`                | auth-db port            |
| `AUTH_DB_NAME` | `auth_db`             | Database name           |
| `AUTH_DB_USER` | `postgres`            | DB user                 |
| `AUTH_DB_PASS` | `postgres`            | DB password             |

---

## Running E2E Tests

Tests use Cypress and require the full stack running locally (not in Docker, as Cypress needs localhost access to both the API and auth-db).

### Setup

```bash
# Start backend services (in Docker)
cd app
docker compose up -d

# Install test dependencies
cd e2e-tests
npm install
```

### Run All Tests (seed + Cypress headless)

```bash
cd app/e2e-tests
npm test
```

This runs `seed.js` first, then executes all Cypress specs.

### Interactive Cypress UI

```bash
cd app/e2e-tests
npm run cy:open
```

### Headless Only (skip seed)

```bash
cd app/e2e-tests
npm run cy:run
```

### Test Suites

| File                  | What it covers                              |
|-----------------------|---------------------------------------------|
| `01_auth.cy.js`       | Register, login, token validation           |
| `02_projects.cy.js`   | Project CRUD, member add/remove             |
| `03_tasks.cy.js`      | Full task lifecycle (create→close)          |
| `04_comments.cy.js`   | Comment CRUD and permission enforcement     |
| `05_admin.cy.js`      | Admin operations on users/projects/tasks    |

### Cypress Environment Variables

```bash
# Override API base URL (default: http://localhost:8000)
CYPRESS_BASE_URL=http://localhost:8000

# auth-db connection (so Cypress can promote users to admin)
AUTH_DB_HOST=localhost
AUTH_DB_PORT=5432
AUTH_DB_NAME=auth_db
AUTH_DB_USER=postgres
AUTH_DB_PASS=postgres
```

---

## Environment Variables Reference

### auth-service

| Variable              | Required | Description                          |
|-----------------------|----------|--------------------------------------|
| `DATABASE_URL`        | yes      | PostgreSQL connection string         |
| `JWT_SECRET`          | yes      | HS256 signing secret                 |
| `PROJECT_SERVICE_URL` | yes      | Base URL of project-service (for admin proxy) |
| `PORT`                | no       | HTTP listen port (default: 8080)     |
| `AUTH_CONFIG_PATH`    | no       | Override config file path            |

### project-service

| Variable           | Required | Description                      |
|--------------------|----------|----------------------------------|
| `DATABASE_URL`     | yes      | PostgreSQL connection string     |
| `JWT_SECRET`       | yes      | Same secret as auth-service      |
| `AUTH_SERVICE_URL` | yes      | Base URL of auth-service         |
| `PORT`             | no       | HTTP listen port (default: 8081) |

### api-gateway

| Variable                   | Required | Description                  |
|----------------------------|----------|------------------------------|
| `AUTH_SERVICE_GRPC_ADDR`   | yes      | auth-service gRPC address    |
| `PROJECT_SERVICE_GRPC_ADDR`| yes      | project-service gRPC address |

---

## Project Structure

```
task-tracker/
├── README.md                        ← this file
├── app/
│   ├── docker-compose.yml           ← full stack orchestration
│   ├── backend/
│   │   ├── auth-service/            ← user auth, JWT, admin panel
│   │   ├── project-service/         ← projects, tasks, comments
│   │   └── api-gateway/             ← public REST → gRPC proxy
│   └── e2e-tests/
│       ├── cypress.config.js
│       ├── package.json
│       ├── scripts/
│       │   └── seed.js              ← demo data seeder
│       └── cypress/
│           └── e2e/                 ← Cypress test specs
└── view/                            ← architecture diagrams (drawio)
```

See each service's own README for detailed internals:

- [auth-service](app/backend/auth-service/README.md)
- [project-service](app/backend/project-service/README.md)
- [api-gateway](app/backend/api-gateway/README.md)
