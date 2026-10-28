# Design: Authentication, RBAC, and Multi-Tenant Access Control

## Context
See `proposal.md` for business motivation and high-level requirements.
Currently, OpsPilot includes `internal/platform/` (config, logger, response, database) and `internal/task/` (entity, memory repository, postgres repository, service, handler, router).
To fulfill Stage 4, we must implement an `internal/auth` domain package adhering to Hexagonal Architecture, integrate authentication and RBAC middleware, and update the task domain to enforce strict tenant isolation.

## Goals / Non-Goals

**Goals:**
- Implement `internal/auth` domain with User entity, repository ports, in-memory & PostgreSQL adapters, password hashing, JWT token lifecycle, and authentication handlers.
- Implement server-side authentication middleware that validates JWT signatures, issuer, audience, and expiration, injecting tenant and user claims into `context.Context`.
- Enforce Role-Based Access Control (RBAC) middleware verifying caller role (`admin`, `member`, `viewer`) for protected operations.
- Enforce multi-tenancy: tasks are bound to `tenant_id`, and repository operations query strictly by `tenant_id` derived from the verified token context.
- Implement brute-force protection via an in-memory rate limiter on `/auth/login`.
- Provide token revocation capabilities via `/auth/logout` and a revocation registry.
- Provide comprehensive test suites covering valid login, invalid credentials, suspended accounts, expired tokens, tampered signatures, rate limiting, RBAC role boundaries, and cross-tenant data isolation.

**Non-Goals:**
- External OAuth2/OIDC SSO providers (Google, GitHub, Okta, SAML) — planned for future stages.
- Out-of-band email delivery / SMTP infrastructure for self-service password reset.
- Distributed token blacklisting across cluster nodes via Redis — memory-based port/adapter is provided with documented scaling path.

## Decisions

### 1. Zero Third-Party Frameworks & Idiomatic Standard Library
- **Decision**: Implement RFC 7519 compliant HMAC-SHA256 (HS256) JWT creation and verification using Go standard library (`crypto/hmac`, `crypto/sha256`, `encoding/base64`, `encoding/json`, `crypto/rand`).
- **Rationale**: Keeps the codebase minimal, auditable, and aligned with `AGENTS.md` ("Idiomatic Simplicity: Prefer Go standard library").
- **Alternative considered**: Third-party libraries like `golang-jwt/jwt`. Rejected to maintain zero unnecessary dependencies and strict control over claim parsing and constant-time signature comparison (`hmac.Equal`).

### 2. Password Hashing via bcrypt
- **Decision**: Use `golang.org/x/crypto/bcrypt` with a minimum cost factor of 12.
- **Rationale**: bcrypt is an industry-standard, adaptive hashing algorithm that automatically salts hashes and resists brute-force GPU attacks.
- **Alternative considered**: Argon2id. While state-of-the-art, `golang.org/x/crypto/bcrypt` is mature, standardized in Go ecosystem, and requires no CGO.

### 3. Tenant Context Propagation & Storage Boundary
- **Decision**: Define private context keys in `internal/auth/context.go` (`tenantKey`, `userKey`). `AuthMiddleware` verifies the Bearer token, populates the request context with `AuthUser{ID, TenantID, Role, Email}`, and handlers/services retrieve this context. In repositories (`MemoryRepository` and `PostgresRepository`), all queries filter strictly by `tenant_id`.
- **Rationale**: Guarantees that clients cannot spoof tenant IDs via HTTP headers or JSON bodies. The authenticated token is the sole authority for tenant scoping.
- **Alternative considered**: Passing tenant ID in URL path (`/tenants/{tenant_id}/tasks`). Rejected because client-supplied URL parameters can be manipulated, introducing broken object-level authorization (BOLA) risks if not strictly cross-checked.

### 4. RBAC Permission Hierarchy
- **Decision**: Define three explicit roles:
  - `admin`: Permitted for all task operations (`create`, `read`, `update`, `delete`).
  - `member`: Permitted for `read`, `create`, `update`; forbidden from `delete` (403 Forbidden).
  - `viewer`: Permitted for `read` only; forbidden from `create`, `update`, `delete` (403 Forbidden).
- **Rationale**: Simple, auditable, and maps directly to ERP/CRM operational boundaries.

### 5. In-Memory Token Revocation and Rate Limiting
- **Decision**: Implement thread-safe in-memory stores (`sync.RWMutex`) for revoked token IDs (`jti`) and login rate limits (sliding window tracking failed attempts per IP/email).
- **Rationale**: Lightweight, zero external infrastructure required for tests and local development. Designed behind Go interfaces (`RevocationStore`, `RateLimiter`) to allow plugging in Redis for multi-node deployments.

## Risks / Trade-offs

- **[Risk] Horizontal Scaling Limitation**: In-memory token revocation and rate-limiting stores are local to a single server instance.
  - **Mitigation**: Define clean interfaces (`RevocationStore`, `RateLimiter`) in `internal/auth`. Document the Redis adapter requirement in production readiness docs.
- **[Risk] JWT Expiration Window**: Access tokens remain valid until expiration unless revoked.
  - **Mitigation**: Use short token lifetimes (default: 15 minutes to 1 hour, configurable via `AUTH_TOKEN_TTL`), coupled with immediate revocation check on protected endpoints.
- **[Risk] Secret Management**: Hardcoded JWT secrets pose a severe vulnerability.
  - **Mitigation**: Enforce loading `AUTH_JWT_SECRET` from environment variables. If running in production mode and secret is missing or insecure, fail fast on server startup.

## Migration Plan

1. Create database migration `000002_create_users_and_tenant_tables.up.sql` adding `users` table and adding `tenant_id` column with index to `tasks`.
2. Update task repository interfaces and implementations to require tenant context / tenant scoping.
3. Register `/auth/login` and `/auth/logout` endpoints in the server router.
4. Wrap task endpoints with `AuthMiddleware` and `RequireRoleMiddleware`.
