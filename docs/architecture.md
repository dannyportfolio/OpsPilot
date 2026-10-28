# Architecture & Engineering Standards

This document describes the architectural foundations of OpsPilot, designed as a reusable system for building professional ERP and CRM applications.

---

## 1. Architectural Philosophy

OpsPilot follows Hexagonal Architecture (Ports and Adapters) with clean package boundaries and zero external dependencies for core modules:

```
          ┌────────────────────────────────────────┐
          │               Transport                │
          │  HTTP Handlers (net/http), Lambda      │
          └───────────────────┬────────────────────┘
                              │ Calls
                              ▼
          ┌────────────────────────────────────────┐
          │              Domain Core               │
          │  Entities, Invariants, Enums, DTOs     │
          │  Repository Interfaces (Ports)         │
          └───────────────────▲────────────────────┘
                              │ Implemented by
                              │
          ┌───────────────────┴────────────────────┐
          │              Persistence               │
          │  MemoryRepository, PostgreSQL, Dynamo  │
          └────────────────────────────────────────┘
```

### Key Rules:
1. **Unidirectional Dependency**: Transport and Persistence depend on Domain Core. Domain Core never imports Transport or Persistence.
2. **Decoupled Handlers**: Handlers register on `http.ServeMux` and return standard library `http.Handler` instances. Handlers do not manage server lifecycle.
3. **Payload Defense**: Every incoming body is bounded with `http.MaxBytesReader(w, r.Body, 1048576)` and decoded with `dec.DisallowUnknownFields()`.
4. **Consistent Envelope**: All endpoints return uniform JSON formats via `internal/platform/response`:
   - Success: Direct domain resource or array.
   - Error: `{"error": "message", "details": [{"field": "name", "message": "reason"}]}`.

---

## 2. ERP & CRM Domain Foundations

Future modules in OpsPilot expand into:
- **CRM**: Contacts, Accounts/Companies, Leads, Opportunities, Activities, Pipelines.
- **ERP**: Products/SKUs, Inventory Warehouses, Purchase Orders, Sales Orders, Invoices, General Ledger.
- **Platform**: Multi-company tenancy, User Authentication, RBAC Permissions, Audit Trails.

### 2.1 Tenancy Architecture
Every database schema and entity must support one of the following tenancy models:

| Tenancy Model | Description | Isolation Guarantee | Best Used For |
|---|---|---|---|
| **Shared Database, Discriminator Column (`tenant_id`)** | All tenants share database tables. Queries strictly filter `WHERE tenant_id = ?`. Compound indexes include `tenant_id` as the leading column. | High (enforced in repository queries + tests) | Standard multi-tenant SaaS CRM/ERP. |
| **Schema-per-Tenant** | Each tenant gets an isolated PostgreSQL schema (`tenant_uuid.*`). | Strict physical schema boundary. | Enterprise customers requiring regulatory data separation. |
| **Database-per-Tenant** | Independent database connection pools per tenant. | Complete infrastructure separation. | High-compliance, on-premise, or sovereign cloud deployments. |

> [!IMPORTANT]
> Regardless of the tenancy model, authorization MUST be validated server-side. Never rely on frontend filtering to isolate data.

### 2.2 Financial & Stock Movement Integrity
- **No Floating Point**: Amounts and quantities use integers (e.g. cents, basis points, milli-units) or fixed-precision structures.
- **Append-Only Ledgers**: Never mutate past financial balances or stock transactions directly. Use double-entry ledger entries or append-only transaction logs.
- **Explicit Transactions**: Complex balance updates must occur within database transactions (`sql.Tx`) with optimistic locking (`version` field) or row-level locking (`SELECT ... FOR UPDATE`).

---

## 3. Architecture Decision Records (ADRs)

We document significant architectural decisions using lightweight ADRs in `docs/decisions/`.

### ADR Lifecycle:
1. **Proposed**: Under discussion via OpenSpec change proposal.
2. **Accepted**: Approved by team/agent consensus and merged.
3. **Superseded**: Replaced by a newer numbered ADR.

### Decision File Naming:
`docs/decisions/<number>-<short-slug>.md` (e.g., `docs/decisions/0001-specification-driven-agentic-architecture.md`).
A standard template is provided at `docs/decisions/0000-template.md`.

---

## 4. Reusable Backend Foundation (Reference Implementation)

The Task API serves as the reference architecture for all business domains in OpsPilot:

### 4.1 Layer Responsibilities

```
cmd/server/main.go  ──▶  internal/platform/config  (Loads env & defaults)
                         internal/platform/logger  (Configures log/slog)
                         internal/platform/database (Connection pool & ping)
                                    │
                                    ▼
                         internal/task/service     (Application use cases & domain orchestration)
                                    │
                        ┌───────────┴───────────┐
                        ▼                       ▼
            internal/task/repository   internal/task/handler
            - MemoryRepository         - Standard net/http routing
            - PostgresRepository       - Request size limit (1 MB)
                                       - DisallowUnknownFields()
```

### 4.2 Migration Strategy
- Migrations reside in `migrations/` as versioned paired files: `*.up.sql` and `*.down.sql`.
- Every migration must be idempotent and non-destructive.
- The base `tasks` table includes an explicit `tenant_id VARCHAR(64)` and a composite index `idx_tasks_tenant_created` on `(tenant_id, created_at DESC)`.

### 4.3 Dual-Runtime Portability
Because `task.NewRouter(service)` yields a pure `http.Handler`, the application runs identically:
1. In local development or Docker containers via `cmd/server/main.go` on `http.Server` with graceful shutdown.
2. In AWS Lambda via `cmd/lambda/main.go` with API Gateway proxy adapters without modifying transport handlers or business logic.

### 4.4 OpenAPI Specification
The formal HTTP contract is maintained in [docs/openapi.yaml](docs/openapi.yaml) (OpenAPI 3.1.0).

