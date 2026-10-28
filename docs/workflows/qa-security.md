# QA & Security Review Workflow

This document details the rigorous testing, verification, and security audit procedures required for all changes in OpsPilot.

---

## 1. QA Verification Workflow

### 1.1 Acceptance Criteria Mapping
Every acceptance scenario defined in OpenSpec delta specs (`specs/<capability>/spec.md`) must map directly to at least one automated Go test function.

| Acceptance Scenario | Test Type | Implementation |
|---|---|---|
| Domain entity validation & enums | Unit test | `internal/<domain>/<domain>_test.go` |
| Repository CRUD & concurrency | Unit / Integration | `internal/<domain>/<repo>_test.go` |
| HTTP request decoding & errors | Handler test | `internal/<domain>/handler_test.go` via `httptest` |
| Multi-tenant cross-boundary isolation | Security / Isolation test | `internal/<domain>/isolation_test.go` |

### 1.2 Table-Driven Testing Standards
All HTTP handler tests must use table-driven tests verifying:
- **Valid Payloads**: Minimal required fields and maximal valid fields.
- **Malformed Payloads**: Invalid JSON syntax, trailing characters, empty body (`{}`).
- **Unknown Fields**: Rejection when extra unmapped fields are sent (`DisallowUnknownFields()`).
- **Boundary Violations**: Strings exceeding max lengths, negative numbers for financial amounts, invalid status enums.
- **HTTP Status Codes**: Exact verification of `200`, `201` (with `Location` header), `204`, `400`, `404`, and `405`.
- **Response Format**: Verification of standard envelope (`{"error": "...", "details": [...]}`).

### 1.3 Concurrency & Race Condition Verification
Execute the Go race detector against all packages:
```powershell
go test -v -race ./...
```
In-memory repositories must employ defensive copying (`*t = original`) within `sync.RWMutex` locks to guarantee that pointers returned to callers cannot mutate repository internal state concurrently.

---

## 2. Security Audit Workflow

The `security` agent runs in **strict review-only mode** (`edit: deny`) to evaluate pull requests and OpenSpec changes against the following security checklist:

### 2.1 Multi-Tenant Segregation Checklist
- [ ] **Context Propagation**: Does every handler extract `tenant_id` from a verified token/context rather than trusting unverified query parameters?
- [ ] **Repository Scoping**: Does every `SELECT`, `UPDATE`, and `DELETE` query explicitly filter by `tenant_id`?
- [ ] **Cross-Tenant Test**: Does the test suite assert that Tenant `A` receives a `404 Not Found` (or `403 Forbidden`) when requesting Tenant `B`'s record ID?
- [ ] **Zero Frontend-Only Security**: Is tenant isolation enforced on the server, not delegated to UI filtering?

### 2.2 Input Defense & DoS Checklist
- [ ] **Body Limit**: Is `http.MaxBytesReader(w, r.Body, 1048576)` invoked before reading any request body?
- [ ] **Disallow Unknown Fields**: Is `dec.DisallowUnknownFields()` enabled on the JSON decoder?
- [ ] **Length Bounds**: Are user-supplied strings (titles, descriptions, names) strictly capped in domain validation?

### 2.3 Secret Hygiene & Logging Checklist
- [ ] **No Hardcoded Secrets**: Are there any API keys, database credentials, JWT secrets, or private certificates committed?
- [ ] **Log Masking**: Are passwords, access tokens, and sensitive customer PII excluded from structured logs?
- [ ] **Environment Separation**: Are secrets loaded strictly from environment variables?

### 2.4 Security Review Commands
```powershell
# Run security review via OpenCode subagent:
opencode --agent security run "Review internal/ and openspec/ for tenant isolation and input validation"

# Scan git repository for accidental secrets:
git diff --staged | Select-String -Pattern "(?i)(api[_-]?key|secret|password|bearer|token)\s*[:=]"
```
