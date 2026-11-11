# Tasks

## 1. Platform & Cross-Domain Decoupling

- [x] 1.1 Create `internal/platform/tenant` and test in `context_test.go`.
- [x] 1.2 Create `internal/platform/httputil` with `DecodeJSON` and test in `decode_test.go`.
- [x] 1.3 Decouple `internal/task` from `internal/auth` by migrating to `internal/platform/tenant`.

## 2. Domain & Transport Refactoring

- [x] 2.1 Refactor `internal/task` to separate transport DTOs and pure domain entities.
- [x] 2.2 Refactor `internal/auth` to decouple use cases from `*slog.Logger` and separate transport DTOs.
- [x] 2.3 Refactor `internal/task/handler.go` and `internal/auth/handler.go` to use `httputil.DecodeJSON`.

## 3. Composition Root & Main Entrypoints

- [x] 3.1 Implement `internal/platform/app` as shared composition root.
- [x] 3.2 Refactor `cmd/server/main.go` and `cmd/lambda/main.go` to delegate to `app.Build()`.

## 4. Verification

- [x] 4.1 Run full build, vet, and test compilation checks.
