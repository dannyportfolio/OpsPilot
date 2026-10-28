# Tasks: Authentication, RBAC, and Multi-Tenant Access Control

## 1. Environment & Dependencies

- [x] 1.1 Add `golang.org/x/crypto` dependency to `go.mod` and add JWT/auth settings to `internal/platform/config`; verify with `go test ./internal/platform/config/...`.
- [x] 1.2 Create database migrations `000002_create_users_and_tenant_support.up.sql` and `down.sql` defining `users` table and adding `tenant_id` to `tasks`.

## 2. Auth Domain Core, Token Management & Protection

- [x] 2.1 Implement `internal/auth` domain models (`User`, `Role`, `Status`, `Claims`, validation) and bcrypt password hashing; verify with unit tests in `internal/auth/user_test.go`.
- [x] 2.2 Implement HMAC-SHA256 JWT token generation, signature validation, audience/issuer checks, and expiration validation; verify with tests in `internal/auth/token_test.go`.
- [x] 2.3 Implement thread-safe token revocation store and in-memory login rate limiter; verify with tests in `internal/auth/revocation_test.go` and `internal/auth/ratelimit_test.go`.

## 3. Auth Persistence & Service Layer

- [x] 3.1 Define `UserRepository` port and implement thread-safe `MemoryUserRepository`; verify with repository tests in `internal/auth/memory_repo_test.go`.
- [x] 3.2 Implement `AuthService` orchestrating credential validation, rate limit checking, token issuance, and logout revocation; verify with tests in `internal/auth/service_test.go`.

## 4. Auth Transport & Access Control Middleware

- [x] 4.1 Implement `internal/auth` HTTP handlers (`/auth/login`, `/auth/logout`) and router returning `http.Handler`; verify with table-driven tests in `internal/auth/handler_test.go`.
- [x] 4.2 Implement `AuthMiddleware` for JWT Bearer token authentication and `RequireRoleMiddleware` for RBAC authorization; verify with middleware tests in `internal/auth/middleware_test.go`.

## 5. Multi-Tenant Scoping, Integration & Verification Gate

- [x] 5.1 Update `internal/task` domain model, repository interface, and implementations to enforce `tenant_id` scoping on all queries; verify with negative cross-tenant tests in `internal/task/memory_repo_test.go`.
- [x] 5.2 Protect task routes with auth and RBAC middleware; verify valid login, invalid credentials, expired tokens, missing auth, insufficient permissions, and cross-tenant access rejection in `internal/task/handler_test.go`.
- [x] 5.3 Integrate auth and protected task routes in `cmd/server/main.go`, verify audit logging sanitization, and execute full verification suite (`go test -race ./...` and `go vet ./...`).
