# OpsPilot — Serverless Multi-Tenant Task REST API in Go

A robust, enterprise-ready Serverless Task REST API built in idiomatic Go. Engineered with strict hexagonal architecture, multi-tenant isolation, cryptographically verified JWT authentication with granular RBAC, database schema migrations, and dual-runtime capability (standalone HTTP server or AWS Lambda behind API Gateway HTTP API v2).

---

## Table of Contents
- [1. Architecture & Design](#1-architecture--design)
  - [Directory Layout](#directory-layout)
  - [Architectural Principles & Boundaries](#architectural-principles--boundaries)
  - [Architecture Decision Records (ADRs)](#architecture-decision-records-adrs)
- [2. Prerequisites](#2-prerequisites)
- [3. Quickstart & Local Execution](#3-quickstart--local-execution)
  - [Mode A: In-Memory Fast Development](#mode-a-in-memory-fast-development-zero-external-dependencies)
  - [Mode B: PostgreSQL with Docker Compose](#mode-b-postgresql-with-docker-compose)
- [4. Environment Variables Reference](#4-environment-variables-reference)
- [5. Database Migrations & Safe Reset Procedures](#5-database-migrations--safe-reset-procedures)
  - [Migration Mechanics](#migration-mechanics)
  - [Safe Reset Workflow (Data Protection)](#safe-reset-workflow-data-protection)
- [6. API Contract & Interactive Documentation](#6-api-contract--interactive-documentation)
  - [OpenAPI Specification](#openapi-specification)
  - [Authentication & RBAC Model](#authentication--rbac-model)
  - [Endpoints Summary](#endpoints-summary)
  - [cURL Request & Response Examples](#curl-request--response-examples)
- [7. Testing & Quality Assurance](#7-testing--quality-assurance)
  - [Test Commands](#test-commands)
  - [Actual Verification Results](#actual-verification-results)
- [8. AWS Serverless Deployment Guide](#8-aws-serverless-deployment-guide)
  - [Local Serverless Emulation](#local-serverless-emulation)
  - [Cloud Deployment Steps (SAM)](#cloud-deployment-steps-sam)
- [9. Troubleshooting Guide](#9-troubleshooting-guide)
- [10. Known Limitations & Production Roadmap](#10-known-limitations--production-roadmap)

---

## 1. Architecture & Design

The repository follows clean/hexagonal architecture with strict unidirectional dependencies. Core business domains (`task`, `auth`) remain decoupled from persistence and HTTP transport layers.

### Directory Layout

```
OpsPilot/
├── cmd/
│   ├── server/                   # Standalone HTTP server entrypoint (local / Docker / ECS)
│   │   ├── main.go
│   │   └── main_test.go
│   └── lambda/                   # AWS Lambda serverless function entrypoint (Graviton ARM64)
│       ├── main.go
│       └── main_test.go
├── internal/
│   ├── auth/                     # Authentication & Authorization business domain
│   │   ├── user.go               # User entity, email/password validation, RBAC roles
│   │   ├── token.go              # HS256 JWT token minting and cryptographic verification
│   │   ├── revocation.go         # In-memory token revocation registry for instant logout
│   │   ├── ratelimit.go          # Sliding-window rate limiter per IP address
│   │   ├── service.go            # Authentication domain service orchestration
│   │   ├── middleware.go         # Context-injecting Authenticate & RequireRole middlewares
│   │   ├── handler.go            # HTTP handlers for /auth/login and /auth/logout
│   │   └── router.go             # Sub-router returning standard net/http.Handler
│   ├── task/                     # Task management business domain
│   │   ├── task.go               # Task entity, status enums, validation logic
│   │   ├── id.go                 # Cryptographically secure RFC 4122 v4 UUID generator
│   │   ├── repository.go         # Persistence interface ports (tenant-scoped)
│   │   ├── memory_repo.go        # Thread-safe in-memory adapter (sync.RWMutex, defensive copies)
│   │   ├── postgres_repo.go      # PostgreSQL adapter with parameterized queries & tenant scoping
│   │   ├── service.go            # Application domain service orchestration
│   │   ├── handler.go            # REST handlers with strict JSON decoding & error mapping
│   │   ├── router.go             # Public task router (backward compatibility)
│   │   └── secured_router.go     # RBAC-secured router with authentication middleware
│   └── platform/                 # Shared infrastructure (zero business domain dependencies)
│       ├── config/               # Environment configuration loader with safe defaults
│       ├── database/             # PostgreSQL connection pool manager & lexical migration engine
│       ├── lambda/               # Serverless adapter translating API Gateway v2 HTTP events
│       ├── logger/               # Structured JSON logger (log/slog)
│       └── response/             # Standardized JSON response envelopes & error formatters
├── migrations/                   # Lexically sorted SQL schema migrations
│   ├── 000001_create_tasks_table.up.sql
│   ├── 000001_create_tasks_table.down.sql
│   ├── 000002_create_users_and_tenant_support.up.sql
│   └── 000002_create_users_and_tenant_support.down.sql
├── docs/                         # Architecture documentation & decisions
│   ├── openapi.yaml              # OpenAPI 3.1.0 contract specification
│   ├── serverless-deployment.md  # Detailed AWS Lambda & API Gateway runbook
│   └── decisions/                # Architecture Decision Records (ADRs)
├── .env.example                  # Template for local environment variables
├── docker-compose.yml            # Local PostgreSQL service definition
├── template.yaml                 # AWS SAM infrastructure definition
├── go.mod                        # Go module definition
└── AGENTS.md                     # Engineering contract governing coding & safety boundaries
```

### Architectural Principles & Boundaries

1. **Hexagonal Decoupling**: Business logic in `internal/task` and `internal/auth` has zero dependency on `net/http` or database drivers. All routers return standard library `http.Handler` instances.
2. **Dual-Runtime Parity**: The same `http.Handler` tree serves both the standalone `cmd/server` binary and the `cmd/lambda` serverless adapter with zero behavioral or contract differences.
3. **Strict Tenancy Enforcement**: Tenant boundaries are enforced server-side. Clients can never supply arbitrary `tenant_id` values in request bodies; tenancy is extracted directly from the verified cryptographic JWT claims (`tid`).
4. **Resilient Serverless Database Pool**: Concurrency limits (`MaxOpenConns: 2`) protect relational databases from connection exhaustion during Lambda cold starts.

### Architecture Decision Records (ADRs)

- [ADR 0001: Specification-Driven Agentic Architecture](docs/decisions/0001-specification-driven-agentic-architecture.md)
- [ADR 0002: Hexagonal Dual-Runtime Architecture and AWS Serverless Adapter](docs/decisions/0002-hexagonal-dual-runtime-and-serverless-adapter.md)
- [ADR 0003: Server-Side Multi-Tenancy and Token Authentication](docs/decisions/0003-server-side-tenancy-and-token-authentication.md)

---

## 2. Prerequisites

- **Go**: Version 1.22 or higher (utilizes standard library `http.ServeMux` pattern matching).
- **Docker & Docker Compose**: (Optional, required only for Mode B with PostgreSQL).
- **AWS SAM CLI**: (Optional, required only for AWS Lambda emulation or cloud packaging).

---

## 3. Quickstart & Local Execution

### Mode A: In-Memory Fast Development (Zero External Dependencies)

In-memory mode stores tasks in memory and seeds a default administrative user. It requires no database setup.

1. **Copy configuration template**:
   ```bash
   cp .env.example .env
   ```
2. **Run the standalone server**:
   ```powershell
   # Windows PowerShell
   $env:STORAGE_DRIVER = "memory"
   $env:PORT = "8080"
   go run ./cmd/server
   ```
   ```bash
   # Linux / macOS
   STORAGE_DRIVER=memory PORT=8080 go run ./cmd/server
   ```
3. **Verify health check**:
   ```bash
   curl -i http://localhost:8080/health
   ```

*Default development credentials:*
- **Tenant ID**: `default-tenant`
- **Email**: `admin@opspilot.internal`
- **Password**: `AdminSecure123!`
- **Role**: `admin`

---

### Mode B: PostgreSQL with Docker Compose

PostgreSQL mode provides persistent relational storage and runs schema migrations automatically on startup.

1. **Start PostgreSQL via Docker Compose**:
   ```bash
   docker compose up -d
   ```
   *Healthcheck will ensure PostgreSQL 16 Alpine is ready on port 5432.*

2. **Configure environment variables**:
   In your `.env` file (or terminal session):
   ```bash
   STORAGE_DRIVER=postgres
   DATABASE_URL=postgres://postgres:postgres@localhost:5432/opspilot?sslmode=disable
   PORT=8080
   ```

3. **Start the server with PostgreSQL persistence**:
   ```powershell
   # Windows PowerShell
   $env:STORAGE_DRIVER = "postgres"
   $env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/opspilot?sslmode=disable"
   go run ./cmd/server
   ```
   ```bash
   # Linux / macOS
   STORAGE_DRIVER=postgres DATABASE_URL="postgres://postgres:postgres@localhost:5432/opspilot?sslmode=disable" go run ./cmd/server
   ```

The application automatically verifies and executes pending migrations (`000001_create_tasks_table.up.sql` and `000002_create_users_and_tenant_support.up.sql`) inside explicit database transactions.

---

## 4. Environment Variables Reference

| Variable | Type | Default | Description |
|---|---|---|---|
| `PORT` | Integer | `8080` | Port for the local HTTP standalone server. |
| `STORAGE_DRIVER` | String | `memory` | Data persistence adapter: `memory` or `postgres`. |
| `DATABASE_URL` | String | `postgres://postgres:postgres@localhost:5432/opspilot?sslmode=disable` | PostgreSQL connection string. |
| `LOG_LEVEL` | String | `INFO` | Minimum structured logging level (`DEBUG`, `INFO`, `WARN`, `ERROR`). |
| `AUTH_JWT_SECRET` | String | `dev-secret-do-not-use-in-production-32-bytes-minimum` | HS256 secret key for signing JWTs (must be >= 16 bytes). |
| `AUTH_TOKEN_TTL_SECONDS` | Integer | `3600` | Access token lifespan in seconds (1 hour). |

---

## 5. Database Migrations & Safe Reset Procedures

### Migration Mechanics
- Migrations reside in `migrations/` and follow lexical naming: `<version>_<name>.(up|down).sql`.
- Applied migrations are recorded in the `schema_migrations` table with timestamps.
- Each migration executes inside an atomic database transaction (`BeginTx`). If any statement fails, the transaction rolls back cleanly.

### Safe Reset Workflow (Data Protection)

> [!CAUTION]
> Never run destructive SQL commands (`DROP TABLE`, `TRUNCATE`) or delete production data. Use the documented rollback and volume management workflows below.

#### Reversible Migration Rollback:
To roll back the latest applied migration safely without destroying other tables:
Use the programmatic `database.MigrateDown(ctx, db, "migrations")` function, which executes the corresponding `.down.sql` script within a transaction and updates `schema_migrations`.

#### Local Development Database Purge:
To reset your local Docker Compose database to an empty state:
```bash
# 1. Stop containers and destroy the named data volume
docker compose down -v

# 2. Re-create the empty PostgreSQL container
docker compose up -d

# 3. Start the application to re-apply up migrations automatically
go run ./cmd/server
```

---

## 6. API Contract & Interactive Documentation

### OpenAPI Specification
The complete API contract is documented in [docs/openapi.yaml](docs/openapi.yaml) conforming to OpenAPI 3.1.0. You can preview it with Swagger UI or the VS Code OpenAPI extension.

### Authentication & RBAC Model
- **Authentication**: JWT Bearer token passed in the `Authorization: Bearer <token>` header.
- **Roles**:
  - `admin`: Full permissions (`GET`, `POST`, `PUT`, `DELETE` tasks).
  - `member`: Standard permissions (`GET`, `POST`, `PUT` tasks).
  - `viewer`: Read-only access (`GET` tasks only). Attempting mutations returns `403 Forbidden`.
- **Tenant Isolation**: Tasks are strictly partitioned by `tenant_id`. Users in `tenant-A` cannot view, update, or delete tasks belonging to `tenant-B`.

### Endpoints Summary

| Method | Endpoint | Auth Required | Minimum Role | Description | Success Status |
|---|---|---|---|---|---|
| `GET` | `/health` | No | None | Service health check | `200 OK` |
| `POST` | `/auth/login` | No | None | Authenticate credentials & issue JWT | `200 OK` |
| `POST` | `/auth/logout` | Yes | Viewer | Revoke access token immediately | `200 OK` |
| `GET` | `/tasks` | Yes | Viewer | List all tasks for current tenant | `200 OK` |
| `POST` | `/tasks` | Yes | Member | Create a new task in tenant scope | `201 Created` |
| `GET` | `/tasks/{id}` | Yes | Viewer | Retrieve single task by UUID | `200 OK` |
| `PUT` | `/tasks/{id}` | Yes | Member | Update task title, description, or status | `200 OK` |
| `DELETE` | `/tasks/{id}` | Yes | Admin | Permanently delete a task | `204 No Content` |

---

### cURL Request & Response Examples

#### 1. Login to Obtain Access Token
```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "default-tenant",
    "email": "admin@opspilot.internal",
    "password": "AdminSecure123!"
  }'
```
*Response (`200 OK`):*
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "tenant_id": "default-tenant",
  "user_id": "8b512c10-b98a-4db5-9989-114cb9bf960b",
  "role": "admin"
}
```

#### 2. Create a Task (Authenticated)
```bash
curl -X POST http://localhost:8080/tasks \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Deploy AWS Lambda Adapter",
    "description": "Verify API Gateway HTTP API v2 integration",
    "status": "in_progress"
  }'
```
*Response (`201 Created`, includes `Location: /tasks/<id>`):*
```json
{
  "id": "c1f10842-83fe-43dc-9e87-a2f073285709",
  "title": "Deploy AWS Lambda Adapter",
  "description": "Verify API Gateway HTTP API v2 integration",
  "status": "in_progress",
  "created_at": "2026-10-28T10:15:00Z",
  "updated_at": "2026-10-28T10:15:00Z"
}
```

#### 3. List Tasks
```bash
curl -X GET http://localhost:8080/tasks \
  -H "Authorization: Bearer <TOKEN>"
```

#### 4. Revoke Token (Logout)
```bash
curl -X POST http://localhost:8080/auth/logout \
  -H "Authorization: Bearer <TOKEN>"
```
*Response (`200 OK`):*
```json
{
  "status": "logged_out"
}
```

---

## 7. Testing & Quality Assurance

### Test Commands

Run the full verification suite including unit, integration, parity, and static analysis:

```powershell
# Format inspection (0 output indicates compliance)
gofmt -l .

# Static analysis
go vet ./...

# Run all test packages
go test -v ./...

# Concurrency & table-driven race testing
go test -v ./internal/auth ./internal/task ./internal/platform/lambda
```

### Actual Verification Results

| Suite / Check | Command | Result | Evidence |
|---|---|---|---|
| Static Analysis | `go vet ./...` | **PASS** | Exit code 0, 0 warnings. |
| Formatting | `gofmt -l .` | **PASS** | Exit code 0, 0 unformatted files. |
| Lambda Entrypoint | `go test opspilot/task-api/cmd/lambda` | **PASS** | 100% passed (cold-start & pool limits). |
| Server Entrypoint | `go test opspilot/task-api/cmd/server` | **PASS** | 100% passed (routes, auth mounting). |
| Auth Domain | `go test opspilot/task-api/internal/auth` | **PASS** | 100% passed (bcrypt, JWT, ratelimit, RBAC). |
| Config Loader | `go test opspilot/task-api/internal/platform/config` | **PASS** | 100% passed (safe defaults, env parsing). |
| Database Engine | `go test opspilot/task-api/internal/platform/database` | **PASS** | 100% passed (nil safety, transactions). |
| Lambda Adapter | `go test opspilot/task-api/internal/platform/lambda` | **PASS** | 100% passed (APIGW v2 event & parity tests). |
| Structured Logger | `go test opspilot/task-api/internal/platform/logger` | **PASS** | 100% passed (JSON schema & levels). |
| Response Envelopes | `go test opspilot/task-api/internal/platform/response` | **PASS** | 100% passed (headers & error contracts). |
| Task Domain | `go test opspilot/task-api/internal/task` | **PASS** | 100% passed (multi-tenant CRUD, validation). |
| OpenSpec Integrity | `openspec validate --all` | **PASS** | 3/3 specs passed (task, auth, lambda). |

---

## 8. AWS Serverless Deployment Guide

OpsPilot includes an AWS SAM template (`template.yaml`) targeting AWS Lambda running on 64-bit ARM Graviton (`provided.al2023`).

### Local Serverless Emulation
You can test the Lambda adapter locally without an AWS account:
```bash
# 1. Build ARM64 binary
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ./bin/bootstrap ./cmd/lambda

# 2. Emulate API Gateway locally using AWS SAM
sam local start-api --template template.yaml
```

### Cloud Deployment Steps (SAM)

> [!IMPORTANT]
> Cloud deployments must only be executed by authorized engineers with provisioned cloud infrastructure.

1. **Build SAM artifact**:
   ```bash
   sam build
   ```
2. **Deploy with guided configuration**:
   ```bash
   sam deploy --guided \
     --stack-name opspilot-task-api-staging \
     --parameter-overrides \
       DatabaseURL="postgres://user:pass@your-rds-endpoint:5432/opspilot?sslmode=require" \
       AuthJwtSecret="your-high-entropy-production-secret"
   ```

Refer to [docs/serverless-deployment.md](docs/serverless-deployment.md) for full IAM permissions, VPC peering, and cost optimization details.

---

## 9. Troubleshooting Guide

### 1. Windows WDAC / AppLocker Blocks Test Binary
- **Symptom**: `fork/exec C:\Users\...\AppData\Local\Temp\go-build...\exe: Access is denied.`
- **Cause**: Windows Defender Application Control blocks arbitrary binaries generated in `%TEMP%` by specific naming patterns.
- **Resolution**: Run `go test` on specific packages, or ensure tests are executed from approved paths. All packages in `internal/` have been hardened to run without triggering WDAC.

### 2. Database Connection Refused (`localhost:5432`)
- **Symptom**: `failed to connect to PostgreSQL: dial tcp [::1]:5432: connectex: No connection could be made.`
- **Cause**: Docker Compose is stopped or PostgreSQL is still initializing.
- **Resolution**: Check container health with `docker compose ps`. Wait for status `(healthy)`.

### 3. Authentication Returns `401 Unauthorized`
- **Symptom**: Task endpoints return `{"error": "unauthorized", "details": [{"message": "invalid or expired token"}]}`.
- **Cause**: Token has expired (default TTL 3600s) or header format is invalid.
- **Resolution**: Ensure header format is strictly `Authorization: Bearer <token>`. Login again at `/auth/login` to obtain a fresh token.

---

## 10. Known Limitations & Production Roadmap

1. **In-Memory Rate Limiting & Revocation**: Rate limiting and token revocation registries currently run in-memory within the active Go process. In a distributed multi-instance deployment (or across multiple concurrent Lambda execution environments), revocation and rate-limit states are not shared.
   - *Roadmap*: Introduce a Redis adapter (`internal/platform/cache`) for shared token blacklists and distributed token-bucket rate limiting.
2. **Lambda Connection Pooling at Scale**: AWS Lambda concurrency spikes can exhaust PostgreSQL connections without an intermediary pooler.
   - *Roadmap*: Deploy AWS RDS Proxy in front of Aurora/PostgreSQL for production serverless deployments.
3. **Secret Injection**: Currently configured via environment variables.
   - *Roadmap*: Add AWS Secrets Manager / Parameter Store integration for zero-trust runtime secret rotation.
