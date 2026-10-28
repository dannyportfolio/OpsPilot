# Proposal: Reusable Backend Architecture Foundation

## Why

OpsPilot requires a production-grade, reusable backend foundation for enterprise ERP and CRM applications. The current reference implementation features clean in-memory Go REST handlers, but lacks persistent relational storage, structured configuration, database migrations, application service orchestration, and multi-tenant preparation necessary for business-critical operations. Establishing this foundation now provides an immutable architectural standard for future ERP/CRM domains while keeping the reference Task API fully operational.

## What Changes

- **Application Services Layer**: Introduce explicit application service use cases separating HTTP transport handling from domain logic.
- **Relational Persistence Adapter**: Implement a PostgreSQL repository conforming to the domain's storage interface, supporting both in-memory and SQL stores.
- **SQL Migration Discipline**: Provide explicit, versioned, reversible forward (`up`) and rollback (`down`) migration scripts with non-destructive guarantees.
- **Structured Logging & Environment Configuration**: Standardize structured JSON logging (`slog`) and 12-factor environment configuration with safe defaults.
- **OpenAPI 3.1 Specification**: Document the complete HTTP API contract in `docs/openapi.yaml`.
- **Local Development Environment**: Provide Docker Compose orchestration for PostgreSQL 16 local testing.
- **Serverless & Container Dual-Runtime**: Preserve decoupled `http.Handler` routing for zero-refactor deployment to both AWS Lambda and standalone HTTP servers.

## Capabilities

### New Capabilities
- `task-service`: Covers the reusable backend service foundation, Task domain entity lifecycle, PostgreSQL persistence adapter, SQL migrations, structured logging, and HTTP transport.

### Modified Capabilities
*(None - this establishes the initial system capability baseline)*

## Impact

- **Affected Code**: `internal/task`, `internal/platform`, `cmd/server`.
- **New Infrastructure Files**: `docker-compose.yml`, `migrations/`, `docs/openapi.yaml`.
- **Dependencies**: PostgreSQL driver (`github.com/jackc/pgx/v5` with stdlib `database/sql` compatibility).
- **APIs**: Existing REST contracts (`GET /health`, `POST /tasks`, `GET /tasks`, `GET /tasks/{id}`, `PUT /tasks/{id}`, `DELETE /tasks/{id}`) remain backward-compatible with enhanced error formatting and RFC 3339 timestamps.
- **Multi-Tenancy Readiness**: Adds structural preparation for `tenant_id` columns and request context propagation across repositories and services.
