# Design: Clean Code and Architecture Refactor

## Context
See `proposal.md` for motivation. The codebase follows hexagonal architecture with ports and adapters. This design solidifies domain purity, eliminates cross-domain coupling, removes duplication, and unifies composition roots.

## Goals / Non-Goals

**Goals:**
- Decouple `internal/task` from `internal/auth` by creating `internal/platform/tenant`.
- Centralize JSON decoding into `internal/platform/httputil` to eliminate duplicated logic across handlers.
- Decouple `authService` from concrete `*slog.Logger`.
- Separate domain entities from transport DTOs.
- Create a shared composition root in `internal/platform/app` for `cmd/server` and `cmd/lambda`.

**Non-Goals:**
- Changing existing public API endpoints or JSON contracts.
- Introducing external dependency injection frameworks (e.g. Wire, Dig). Go standard library struct assembly is used.

## Decisions

### 1. `internal/platform/tenant` as Cross-Cutting Context Boundary
- **Decision**: Put `WithTenant(ctx, tenantID)` and `FromContext(ctx)` in `internal/platform/tenant`.
- **Rationale**: Tenancy is a foundational platform concern, not an authentication feature. Subdomains should not import each other to read context tenant.

### 2. Centralized `httputil.DecodeJSON`
- **Decision**: Place bounded JSON decoding in `internal/platform/httputil`.
- **Rationale**: Both `task/handler.go` and `auth/handler.go` had identical 1MB limits, `DisallowUnknownFields()`, and extra token checks.

### 3. Unified Composition Root in `internal/platform/app`
- **Decision**: `app.Build(cfg *config.Config, log *slog.Logger) (*app.App, error)` builds the database pool, repositories, services, and returns the mounted `http.Handler`.
- **Rationale**: `cmd/server` and `cmd/lambda` currently duplicate 60+ lines of identical wiring.

## Risks / Trade-offs

- [Risk: Backward compatibility with tests] → Mitigation: Preserve type aliases or backwards-compatible function wrappers where needed.
