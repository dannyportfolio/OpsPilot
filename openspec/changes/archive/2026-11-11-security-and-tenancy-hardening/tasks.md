# Tasks

## 1. Security & Rate-Limiter Hardening

- [x] 1.1 Harden `extractClientIP` in `internal/auth/handler.go` against spoofed `X-Forwarded-For`; verify with unit tests in `internal/auth/handler_test.go`.
- [x] 1.2 Implement sliding window cleanup in `MemoryRateLimiter` (`internal/auth/ratelimit.go`) to prevent memory leaks; verify with pruning test in `internal/auth/ratelimit_test.go`.
- [x] 1.3 Update `NewTokenService` in `internal/auth/token.go` to enforce a 32-byte secret minimum; verify with unit test in `internal/auth/token_test.go`.

## 2. Tenancy & Storage Adapter Implementation

- [x] 2.1 Update `LoginRequest` and `authService.Login` to support optional `tenant_id` and resolve ambiguous email collisions; verify with tests in `internal/auth/service_test.go`.
- [x] 2.2 Implement `PostgresUserRepository` in `internal/auth/postgres_repo.go` implementing `UserRepository`; verify with unit/integration tests in `internal/auth/postgres_repo_test.go`.
- [x] 2.3 Wire `PostgresUserRepository` into `cmd/server/main.go` and `cmd/lambda/main.go` when `storage_driver=postgres`.
- [x] 2.4 Eliminate silent fallback to `"default"` tenant in `internal/task/service.go` and `internal/task/postgres_repo.go`; verify with tests in `internal/task/service_test.go`.

## 3. Credential Hygiene, User Endpoints & Infrastructure

- [x] 3.1 Gate development seed user creation behind `SEED_DEV_DATA=true` in `cmd/server/main.go` and `cmd/lambda/main.go`; enforce production secret checks in `internal/platform/config/config.go`.
- [x] 3.2 Add `POST /auth/users` and `GET /auth/me` endpoints in `internal/auth/handler.go` and register in `internal/auth/router.go`; verify with handler tests.
- [x] 3.3 Update `template.yaml` to parameterize CORS `AllowOrigins`.

## 4. Full Verification

- [x] 4.1 Run full test suite and `go vet ./...` to verify all components pass cleanly.
