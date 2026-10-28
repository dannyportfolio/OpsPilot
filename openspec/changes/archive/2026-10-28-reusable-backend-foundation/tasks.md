# Tasks: Reusable Backend Architecture Foundation

## 1. Environment Configuration & Structured Logging

- [x] 1.1 Implement environment configuration loader in `internal/platform/config/config.go` with safe defaults (`PORT`, `DATABASE_URL`, `STORAGE_DRIVER`, `LOG_LEVEL`) and verify with `config_test.go`
- [x] 1.2 Implement structured JSON logger setup in `internal/platform/logger/logger.go` using standard library `log/slog` and verify logging output format

## 2. Relational Schema & Migration Infrastructure

- [x] 2.1 Author SQL migration scripts `migrations/000001_create_tasks_table.up.sql` and `migrations/000001_create_tasks_table.down.sql` with `tenant_id` and composite index
- [x] 2.2 Create `docker-compose.yml` defining PostgreSQL 16 container service with health check and environment variables for local development
- [x] 2.3 Implement database connection pool initializer in `internal/platform/database/postgres.go` with connection pooling limits and ping verification

## 3. Persistence Adapters & Application Service

- [x] 3.1 Implement PostgreSQL storage adapter in `internal/task/postgres_repo.go` conforming to `task.Repository` interface
- [x] 3.2 Author integration tests for `postgres_repo.go` verifying CRUD operations, context cancellation, and SQL error mapping
- [x] 3.3 Implement application service layer in `internal/task/service.go` encapsulating domain invariants, validation, and repository orchestration
- [x] 3.4 Author unit tests in `internal/task/service_test.go` verifying service business logic with both mock/in-memory and real repository adapters

## 4. HTTP Transport & Server Integration

- [x] 4.1 Update `internal/task/handler.go` and `router.go` to delegate to `task.Service` while preserving standard `http.Handler` routing
- [x] 4.2 Verify HTTP handler test suite in `internal/task/handler_test.go` covers all 2xx, 400, 404, and 405 scenarios with table-driven tests
- [x] 4.3 Update `cmd/server/main.go` to initialize configuration, database pool, logger, and graceful server shutdown

## 5. Verification & Specification Sync

- [x] 5.1 Execute full test suite `go test -v -cover ./...` and `go vet ./...` reporting test execution evidence and coverage
- [x] 5.2 Validate OpenSpec artifacts with `openspec validate --all` and ensure zero errors
- [x] 5.3 Review code changes with `opencode --agent security` in review-only mode to audit tenant scoping and input validation

## Workflow follow-up

- Archive the change (`openspec archive reusable-backend-foundation`) after Stage 3 implementation is reviewed and approved.
- Promote delta specifications to main `openspec/specs/task-service/spec.md`.
