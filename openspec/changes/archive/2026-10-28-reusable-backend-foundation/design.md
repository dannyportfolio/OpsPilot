# Design: Reusable Backend Architecture Foundation

## Context

OpsPilot serves as a reusable foundation for building modular ERP and CRM business applications in Go. The reference implementation in `internal/task` demonstrated idiomatic Go standard library routing, in-memory concurrency safety (`sync.RWMutex`), and zero external dependencies. To support production ERP/CRM workflows, the backend must evolve into a layered, hexagonal architecture that supports relational database persistence (PostgreSQL), SQL migrations, application use case orchestration, and structured logging, while retaining pure standard library HTTP handler decoupling for serverless (AWS Lambda) portability.

See `proposal.md` for business motivation and non-goals.

## Goals / Non-Goals

**Goals:**
- Provide a clean hexagonal architecture separating Transport, Application Use Cases, Domain Core, and Persistence Adapters.
- Provide PostgreSQL repository implementation alongside existing in-memory repository for rapid testing and production persistence.
- Implement explicit forward and rollback SQL migrations (`migrations/*.sql`).
- Standardize on `log/slog` for structured JSON logging with configurable log levels.
- Provide 12-factor environment configuration with safe local development defaults.
- Document the complete HTTP API in OpenAPI 3.1 (`docs/openapi.yaml`).
- Provide Docker Compose configuration for local PostgreSQL 16 testing.
- Maintain decoupled `http.Handler` routing for zero-code-change AWS Lambda and API Gateway deployment.

**Non-Goals:**
- No premature microservices; all modules reside in a modular monolith structure.
- No heavy third-party web frameworks (e.g. Gin, Echo, Fiber) or ORMs (e.g. GORM); rely on `net/http` and `database/sql`.
- No immediate cloud deployment or Terraform provisioning in this foundation stage.
- No complex distributed transactions; use standard PostgreSQL ACID transactions.

## Decisions

### 1. Package Structure & Architectural Separation
We adopt hexagonal/clean architecture with clear unidirectional boundaries:

```
OpsPilot/
├── cmd/
│   ├── server/main.go             # Local HTTP server entrypoint
│   └── lambda/main.go             # AWS Lambda adapter entrypoint (deployment-ready)
├── internal/
│   ├── platform/                  # Shared infrastructure (zero business domain awareness)
│   │   ├── config/config.go       # Environment configuration loading with defaults
│   │   ├── database/postgres.go   # PostgreSQL connection pool lifecycle and ping
│   │   ├── logger/logger.go       # Structured JSON logging setup via log/slog
│   │   └── response/response.go   # Consistent JSON API envelopes (JSON, Error)
│   └── task/                      # Task business domain
│       ├── task.go                # Domain entity, status enum, validation rules
│       ├── repository.go          # Persistence interface (Port)
│       ├── memory_repo.go         # Concurrency-safe in-memory adapter (sync.RWMutex)
│       ├── postgres_repo.go       # PostgreSQL relational persistence adapter
│       ├── service.go             # Application service / use cases (orchestration)
│       ├── handler.go             # HTTP transport handlers (request decoding & validation)
│       └── router.go              # Route registration returning standard http.Handler
├── migrations/                    # Versioned SQL migrations (up/down)
├── docs/                          # Architecture, OpenAPI, and workflow guides
├── docker-compose.yml             # Local PostgreSQL 16 service definition
├── go.mod                         # Go module definition
└── AGENTS.md                      # Shared engineering contract
```

*Alternatives considered*:
- *Flat package layout*: Rejected because mixing HTTP transport logic with SQL queries causes spaghetti dependencies and hinders testing.
- *Microservices layout*: Rejected as premature overhead for a foundation repository.

### 2. Task Entity Lifecycle & Invariants
- **ID Strategy**: RFC 4122 v4 UUID generated securely via `crypto/rand` (`internal/task/id.go`).
- **Timestamp Strategy**: RFC 3339 UTC timestamps (`created_at`, `updated_at`).
- **State Machine**:
  - Initial state: `pending`.
  - Permitted transitions: `pending` → `in_progress` → `completed`.
  - Terminal state: `completed` (cannot revert without explicit administrative action in future ERP specs).

### 3. Relational Persistence & SQL Migrations
- **Database Driver**: `github.com/jackc/pgx/v5/stdlib` for high-performance PostgreSQL access through the Go standard `database/sql` interface.
- **Table Schema**:
  ```sql
  CREATE TABLE IF NOT EXISTS tasks (
      id UUID PRIMARY KEY,
      title VARCHAR(200) NOT NULL,
      description VARCHAR(2000) NOT NULL DEFAULT '',
      status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'in_progress', 'completed')),
      tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
      created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
  );

  CREATE INDEX IF NOT EXISTS idx_tasks_tenant_created 
  ON tasks(tenant_id, created_at DESC);
  ```
- **Migration Strategy**: Sequential file-based SQL migrations:
  - `migrations/000001_create_tasks_table.up.sql`
  - `migrations/000001_create_tasks_table.down.sql`
  Every migration must have an idempotent, non-destructive forward path and a clean rollback path.

### 4. Application Service Layer
Transport handlers delegate to a `TaskService`:
```go
type Service interface {
    CreateTask(ctx context.Context, req CreateTaskRequest) (*Task, error)
    GetTask(ctx context.Context, id string) (*Task, error)
    ListTasks(ctx context.Context) ([]*Task, error)
    UpdateTask(ctx context.Context, id string, req UpdateTaskRequest) (*Task, error)
    DeleteTask(ctx context.Context, id string) error
}
```
This isolates validation, transactional boundaries, and logging from HTTP request/response handling.

### 5. API Contracts & Error Envelope
- **Status Codes**:
  - `200 OK`: GET, PUT success.
  - `201 Created`: POST success (with `Location: /tasks/{id}` header).
  - `204 No Content`: DELETE success (idempotent).
  - `400 Bad Request`: Validation failure, malformed JSON, unknown fields.
  - `404 Not Found`: Task ID not found.
  - `405 Method Not Allowed`: Unsupported HTTP verb.
  - `500 Internal Server Error`: Unhandled infrastructure failure.
- **Error Format**:
  ```json
  {
    "error": "validation failed",
    "details": [
      {
        "field": "title",
        "message": "title is required and cannot be empty"
      }
    ]
  }
  ```

### 6. Configuration & Structured Logging
- **Configuration**: `internal/platform/config/config.go` reads environment variables with safe defaults:
  - `PORT`: Server port (default: `"8080"`).
  - `DATABASE_URL`: PostgreSQL connection string (default: `"postgres://postgres:postgres@localhost:5432/opspilot?sslmode=disable"`).
  - `STORAGE_DRIVER`: `"memory"` or `"postgres"` (default: `"memory"` for frictionless testing).
  - `LOG_LEVEL`: `"DEBUG"`, `"INFO"`, `"WARN"`, `"ERROR"` (default: `"INFO"`).
- **Logging**: Standard library `log/slog` writing JSON to `os.Stdout`. Logs include `method`, `path`, `status`, `duration_ms`, and sanitized error strings.

### 7. Dual-Runtime: Standalone HTTP Server vs AWS Lambda
- The domain router returns a standard `http.Handler` (`task.NewRouter(service)`).
- **Local HTTP**: Mounted on `http.Server` with graceful shutdown (`srv.Shutdown`) listening to `SIGINT`/`SIGTERM`.
- **AWS Lambda**: Mounted on `httpadapter.New(router)` in `cmd/lambda/main.go`. Handlers require zero modification between runtimes.

### 8. Testing Strategy
- **Unit Tests**: Domain entity validation (`task_test.go`), ID generator (`id_test.go`), response helper (`response_test.go`).
- **Handler Tests**: Table-driven tests using `net/http/httptest` covering all 2xx, 400, 404, and 405 paths.
- **Repository Tests**: Both `MemoryRepository` and `PostgresRepository` must pass identical test suites asserting CRUD operations, context cancellation, and concurrency safety.
- **Race Condition Testing**: Run `go test -v ./...` verifying defensive copies and mutex synchronization.

## Risks / Trade-offs

- **[Risk]** Database connectivity drops during runtime → **Mitigation**: Configure connection pool limits (`SetMaxOpenConns`, `SetMaxIdleConns`, `SetConnMaxLifetime`) and background ping health checks.
- **[Risk]** Divergence between in-memory and PostgreSQL repository behavior → **Mitigation**: Shared test suite validating identical behavior across both adapters.
- **[Risk]** Accidental secret leakage in connection strings → **Mitigation**: Never log full `DATABASE_URL`; mask passwords in connection logs.

## Migration Plan

1. Stage 1 (Current): Establish OpenSpec specifications, technical design, OpenAPI 3.1 contract, and task breakdown. Stop for human approval.
2. Stage 2 (Implementation): Incrementally implement `internal/platform/config`, `migrations/`, `internal/task/postgres_repo.go`, `internal/task/service.go`, and verify with `go test`.
