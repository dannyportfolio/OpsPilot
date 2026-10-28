# Proposal: Authentication, RBAC, and Multi-Tenant Access Control

## Why
OpsPilot is designed as a reusable foundation for multi-tenant ERP and CRM business applications. Currently, the system lacks authentication and authorization boundaries; all endpoints are publicly accessible, and domain entities do not isolate company tenant boundaries. To safeguard business data and meet the security requirements of Stage 4, OpsPilot requires cryptographic user authentication, role-based authorization (RBAC), and server-enforced multi-tenant isolation.

## What Changes
- **User Identity & Lifecycle**: Introduce User entity with email, role (`admin`, `member`, `viewer`), tenant ID, and status (`active`, `suspended`).
- **Secure Password Storage**: Hash user passwords with `bcrypt` (cost 12); strictly prohibit storing or logging plaintext passwords.
- **Token-Based Authentication**: Issue signed HMAC-SHA256 JWT access tokens upon successful authentication containing subject (`user_id`), `tenant_id`, `role`, standard claims (`iss`, `aud`, `exp`, `iat`), and support server-side token revocation/invalidation.
- **Login, Logout & Rate-Limiting**: Expose `/auth/login` and `/auth/logout` endpoints with in-memory IP/account rate-limiting against brute force attacks and uniform, non-enumerating error responses.
- **Strict Server-Side Authorization**: Implement auth middleware enforcing default-deny semantics. Validate token signatures, claims, and tenant bounds on protected endpoints.
- **Multi-Tenant Scoping**: Update task persistence and service layers to associate tasks with `tenant_id`. Every repository query filters strictly by `tenant_id` from the authenticated token context. Prevent arbitrary tenant overriding.
- **RBAC Matrix**: Enforce granular role permissions:
  - `admin`: Full read/write/delete access within their tenant.
  - `member`: Read/write access within their tenant; cannot delete tasks.
  - `viewer`: Read-only access within their tenant.
- **Audit Logging & Sensitive Data Protection**: Structured logging with `slog` omitting passwords, tokens, and authorization headers, recording security-relevant audit events (login success, login failure, access denial).

## Capabilities

### New Capabilities
- `auth-and-access-control`: User identity lifecycle, password hashing, JWT issuance and validation, token revocation, rate-limiting, role-based access control (RBAC), and tenant isolation middleware for OpsPilot resources.

### Modified Capabilities
*(None; previous foundation capabilities will enforce auth via middleware and tenant scoping).*

## Impact
- **Architecture**: New `internal/auth` domain module following Hexagonal Architecture (entities, repository port, in-memory & SQL storage adapters, service, handler, router, middleware).
- **Existing Endpoints**: Protected endpoints (`/tasks`, `/tasks/{id}`) now require a valid Bearer token and enforce tenant isolation and role permissions. Health check (`/health`) remains public.
- **Dependencies**: Uses `golang.org/x/crypto/bcrypt` for secure hashing and standard library / cryptographic HMAC-SHA256 for signed token validation without third-party frameworks.
- **Configuration**: New configuration variables for `JWT_SECRET`, token TTL, and rate-limiting thresholds with safe development defaults.
- **Security Boundaries**: Default-deny access policy; cross-tenant requests return 404 or 403; no secret leakage in logs or error messages.
