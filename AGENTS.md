# AGENTS.md — Shared Engineering Contract

This document defines the strict engineering contract, architectural boundaries, security policies, and AI agent workflows for all ERP and CRM applications developed within this repository.

---

## 1. Core Principles

### 1.1 Specification-First Development
- **OpenSpec is the Source of Truth**: No feature, architectural change, or non-trivial refactor begins in source code before an OpenSpec change proposal (`openspec/changes/<change-id>/`) is authored and reviewed.
- **Delta Specifications**: Every capability under `openspec/changes/<change-id>/specs/<capability>/spec.md` specifies requirements and acceptance scenarios (`Given-When-Then`).
- **Proportional Overhead**: Trivial typo corrections and non-functional comment fixes do not require full specification proposals. Use judgment: any change impacting API contracts, schema, domain logic, or security must go through OpenSpec.

### 1.2 Small, Reviewable Increments
- Decompose complex initiatives into small, atomic tasks in `tasks.md`.
- Implement and verify one task at a time.
- Review diffs before advancing to subsequent tasks.
- Keep commits focused and logically grouped.

### 1.3 Token-Efficient Codebase Exploration
- **Targeted Inspection**: Do not perform broad or repetitive scans across the entire codebase.
- **Direct Context Loading**: Read only files directly referenced by the current task or specification.
- **Reference Once**: Consult `docs/context.md` and `docs/architecture.md` for architectural overviews instead of re-reading raw directory trees.
- **Summarize Concisely**: Keep communication concise. State changed files, test results, and unresolved items directly.

### 1.4 No Premature Abstractions or Unnecessary Dependencies
- **Idiomatic Simplicity**: Prefer the Go standard library (`net/http`, `crypto/rand`, `sync`, `encoding/json`, `database/sql`, `context`).
- **No Speculative Frameworks**: Do not introduce third-party libraries (e.g. heavy ORMs, web frameworks) unless strictly necessary and approved in a formal Architecture Decision Record (ADR).
- **Zero Duplicate Logic**: Reuse existing domain entities and response envelopes (`internal/platform/response`).

---

## 2. Architectural Boundaries & Code Organization

The system enforces hexagonal/clean architecture with strict unidirectional dependencies:

```
OpsPilot/
├── cmd/                          # Application entrypoints (server, lambda, cli)
│   └── server/main.go
├── internal/
│   ├── platform/                 # Shared cross-cutting infrastructure (zero domain dependencies)
│   │   ├── response/             # Standard JSON API envelopes and error formatting
│   │   └── config/               # Environment & configuration loading
│   └── <domain>/                 # Self-contained business domains (e.g. task, contact, order)
│       ├── <domain>.go           # Domain entities, validation rules, status enums, DTOs
│       ├── <domain>_test.go      # Domain entity unit tests
│       ├── repository.go         # Storage interface definitions (ports)
│       ├── memory_repo.go        # Thread-safe in-memory adapter (for tests and local dev)
│       ├── memory_repo_test.go   # Repository test suite
│       ├── handler.go            # HTTP transport handlers (request decoding, status mapping)
│       ├── handler_test.go       # Table-driven HTTP handler tests using net/http/httptest
│       └── router.go             # Sub-router route registration returning standard http.Handler
```

### Layer Rules:
1. **Domain Logic (`internal/<domain>/<domain>.go`)**: Must have zero transport (`http`) or persistence dependencies. Pure business invariants only.
2. **Persistence Interfaces (`internal/<domain>/repository.go`)**: Domain defines its required persistence interface. Adapters implement it.
3. **Transport Handlers (`internal/<domain>/handler.go`)**: Handle HTTP serialization, enforce payload size limits (e.g. `http.MaxBytesReader`), reject unknown JSON fields (`DisallowUnknownFields`), and map domain errors to standard response codes.
4. **Decoupled Routers (`internal/<domain>/router.go`)**: Routers return standard library `http.Handler` instances, enabling both standard HTTP servers and serverless AWS Lambda adapters without changing domain logic.

---

## 3. Testing & Verification Standards

### 3.1 Strict Verification Honesty
- **No Speculative Success**: Never report or claim that a test, build, lint check, or command passed unless it was actually executed and exited with status code `0`.
- **Include Verifiable Evidence**: Report command output, test counts, execution time, and coverage figures in task completion summaries.

### 3.2 Testing Requirements
- **Unit Tests**: Required for all business logic, status transitions, and validation rules.
- **Handler Tests**: Table-driven tests using `net/http/httptest` covering:
  - 2xx Success payloads and headers.
  - 400 Bad Request (malformed JSON, unknown fields, missing required attributes, boundary violations).
  - 404 Not Found (non-existent IDs).
  - 405 Method Not Allowed.
  - Concurrency safety and race condition absence (`go test -race`).
- **Regression Protection**: Every bug fix must include a test reproducing the defect prior to resolving it.

---

## 4. ERP & CRM Domain Guardrails

Applications built on this foundation manage business-critical resources (contacts, companies, leads, opportunities, invoices, stock movements). The following guardrails are strictly mandatory:

### 4.1 Multi-Tenancy & Isolation
- **Explicit Tenancy Model**: Every request must carry an authenticated, cryptographically validated tenant context (`TenantID`).
- **Server-Side Enforcement**: Tenant isolation must be enforced on every repository query and domain operation.
- **Never Trust the Client**: Frontend filtering is purely cosmetic; server-side database access must filter strictly by `tenant_id`.
- **Mandatory Isolation Tests**: Every data access layer test must verify that Tenant A cannot read, mutate, or delete Tenant B's data, even when supplying a valid record ID.

### 4.2 Financial & Inventory Data Integrity
- **Auditable State Transitions**: Domain state changes (e.g. `draft` → `approved` → `invoiced` → `paid`) must follow strict, verifiable state machines.
- **No Silent Mutations**: Critical business records must not be quietly overwritten. Use append-only audit logs or immutable ledger entries for financial and inventory transactions.
- **Precision**: Monetary figures must never use floating-point types (`float64`). Use fixed-point representation (e.g. integers representing cents or microprecision structs).
- **Concurrency Safety**: State changes affecting balances, allocations, or stock levels must execute inside explicit transaction boundaries with optimistic locking (`version` field) or row locks.

---

## 5. Security & Secret Hygiene

- **Zero Secrets in Code or Git**: Never hardcode API keys, database passwords, OAuth secrets, or credentials in any file committed to version control.
- **Environment-Based Configuration**: All sensitive values must be injected via external environment variables (e.g. `.env` listed in `.gitignore`, loaded only at runtime).
- **Input Sanitization & Boundary Limits**:
  - Bound incoming request bodies with `http.MaxBytesReader` (default: 1 MB).
  - Use `DisallowUnknownFields()` during JSON decoding to prevent parameter pollution.
- **Safety Boundaries**: No automated agent is permitted to execute cloud deployments, run destructive database drops (`DROP TABLE`, `TRUNCATE`), or make destructive git modifications (`git push --force`) without explicit human instruction.

---

## 6. Multi-Tool AI Agent Workflow

| Tool | Primary Responsibility | Permissions & Bounds |
|---|---|---|
| **OpenSpec** | Requirements, proposal, delta specs, task planning, and spec sync | CLI commands: `openspec new`, `validate`, `status`, `archive`. |
| **OpenCode** | Primary implementation, automated Go tests, refactoring, linting | CLI commands; specialized roles via `.opencode/agents/`. |
| **Antigravity** | UI/UX implementation, browser verification, visual QA, pair-programming | Frontend engineering, full-stack review, interactive QA. |
| **OpenRouter** | Direct model access for OpenCode agents | Configured via environment variables; tiered model usage. |
| **9Router** | Optional local gateway for model routing & protocol translation | Local port `20128` if explicitly activated. |

### Role Handoff Lifecycle:
1. **Explore & Propose**: OpenSpec captures feature intent in `proposal.md` and delta specs.
2. **Architecture Check**: Architect Agent reviews entity boundaries, tenancy impacts, and ADRs.
3. **Implementation**: Backend/Frontend Agents implement tasks incrementally following `tasks.md`.
4. **Verification**: QA Agent verifies acceptance criteria with unit/integration tests.
5. **Security Audit**: Security Agent reviews authorization and input validation in review-only mode.
6. **Archive**: OpenSpec archives the change, promoting delta specs to the main specification tree.
