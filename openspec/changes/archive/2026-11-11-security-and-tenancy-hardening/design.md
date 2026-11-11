# Design: Security and Tenancy Hardening

## Context
See `proposal.md` for background. The system is written in idiomatic Go (1.22+) using hexagonal architecture with decoupled HTTP routers returning `http.Handler`. Storage includes in-memory mocks for testing and PostgreSQL for production.

## Goals / Non-Goals

**Goals:**
- Eliminate CWE-290 (IP spoofing) by strictly validating client IP extraction.
- Eliminate CWE-400 (unbounded memory growth) in the in-memory rate limiter with sliding window pruning.
- Implement `PostgresUserRepository` satisfying `auth.UserRepository` with parameterized SQL.
- Prevent cross-tenant collisions in user lookups.
- Remove hardcoded credentials from startup routines.
- Remove silent `"default"` tenant fallback from domain operations.
- Expose administrative endpoints `POST /auth/users` and `GET /auth/me`.

**Non-Goals:**
- External Redis or DynamoDB cluster deployment (out of scope for single database + in-memory parity).
- Implementing full OAuth 2.0 / OIDC provider flows (existing HS256 JWT architecture is preserved).

## Decisions

### 1. Trusted Proxy Verification in IP Extraction
- **Decision**: In `internal/auth/handler.go`, inspect `X-Forwarded-For` only if `r.RemoteAddr` matches a configured trusted proxy list or loopback. Otherwise, use `r.RemoteAddr`.
- **Alternative considered**: Always stripping `X-Forwarded-For`. Rejected because legitimate reverse proxies (e.g. AWS ALB / API Gateway) pass the real client IP in `X-Forwarded-For` or `RequestContext.HTTP.SourceIP`.

### 2. Eviction Strategy for In-Memory Rate Limiter
- **Decision**: Run periodic cleanup on an interval or prune expired entries on mutation when `now.After(info.blockedTo)` and all recorded failure timestamps are older than `rl.window`.
- **Alternative considered**: Third-party LRU cache library. Rejected to adhere to Go standard library minimalism (`sync.Mutex` with map pruning).

### 3. PostgreSQL Persistence for User Records
- **Decision**: Create `internal/auth/postgres_repo.go` using `database/sql` queries against the existing `users` table created by migration `000002_create_users_and_tenant_support.up.sql`.
- **Alternative considered**: GORM or heavy ORM. Rejected in accordance with `AGENTS.md` Rule 1.4.

### 4. Tenancy Disambiguation on Login
- **Decision**: Update `LoginRequest` to accept an optional `tenant_id`. If `tenant_id` is supplied, query strictly by `(tenant_id, email)`. If omitted and email is unique across the system, authenticate successfully; if ambiguous across multiple tenants, reject with requirement to specify `tenant_id`.

## Risks / Trade-offs

- [Risk: Cold start latency in serverless] → Mitigation: Connection pool limits tuned to 2 conns in Lambda.
- [Risk: In-memory rate limiting not synchronized across multiple Lambda instances] → Mitigation: Standard for serverless without Redis/ElastiCache; API Gateway throttles can be configured at ingress level.
