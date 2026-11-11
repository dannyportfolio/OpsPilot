# Proposal: Clean Code and Architecture Refactor

## Why

To establish sustainable, enterprise-grade hexagonal architecture as mandated by `AGENTS.md`:
1. Cross-domain leakage exists between `internal/task` and `internal/auth` due to direct context tenancy dependencies.
2. HTTP transport concerns (JSON tags, request payload decoding) are duplicated across domains rather than cleanly abstracted.
3. Domain services leak concrete infrastructure logging types (`*slog.Logger`).
4. `cmd/server` and `cmd/lambda` duplicate composition root assembly logic.

## What Changes

- **Platform Tenant Context**: Extract tenant context helpers into `internal/platform/tenant`, decoupling business domains from `auth`.
- **Centralized HTTP Utilities**: Create `internal/platform/httputil` to provide robust, bounded JSON request decoding (`DecodeJSON`).
- **Domain & DTO Separation**: Separate pure domain entities from transport DTOs in `internal/task` and `internal/auth`.
- **Infrastructure Logger Decoupling**: Make domain use cases pure by removing direct infrastructure logger dependencies.
- **Unified Composition Root**: Implement `internal/platform/app` providing a single `app.Build()` assembly function shared by standard HTTP servers and serverless AWS Lambda adapters.

## Capabilities

### Modified Capabilities
- `task-service`: Decouple task domain persistence and services from auth package, using platform tenant context.
- `auth-and-access-control`: Separate transport DTOs and pure domain logic, using platform tenant context and centralized request decoding.
- `aws-serverless-adapter`: Use unified application builder for consistent composition root parity with standard server.

## Impact

- Zero API contract breakage (all JSON request/response formats and status codes preserved).
- Substantial reduction in code duplication across handlers and entrypoints.
- Independent evolvability of individual business domains.
