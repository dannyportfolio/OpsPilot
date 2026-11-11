# Proposal: Security and Tenancy Hardening

## Why

A comprehensive security and architectural audit identified critical vulnerabilities and operational risks in authentication and multi-tenancy:
1. The login rate-limiter blindly trusts `X-Forwarded-For`, permitting brute-force protection bypass.
2. In-memory rate limiting lacks key eviction, causing unbounded memory growth (CWE-400 / DoS).
3. The tenancy model exhibits collisions during login across multi-tenant user stores.
4. When configured with `STORAGE_DRIVER=postgres`, user records, sessions, and revocations remain in-memory only.
5. Insecure fallback JWT secrets and plaintext seed admin credentials exist in server entrypoints.
6. The task domain silently defaults to a `"default"` tenant on unauthenticated operations.

Hardening these boundaries is required before deploying OpsPilot in production multi-tenant environments.

## What Changes

- **Rate-Limiting & IP Sanitization**: Restrict `X-Forwarded-For` inspection to verified trusted proxies only; fallback to `r.RemoteAddr`. Add active sliding-window eviction to prevent memory leaks in `MemoryRateLimiter`.
- **Cryptographic Key Enforcement**: Enforce minimum 32-byte (256-bit) signing secrets for HMAC-SHA256 tokens per RFC 7518.
- **Persistent User Management**: Implement `PostgresUserRepository` to persist user accounts, roles, and status to PostgreSQL when `STORAGE_DRIVER=postgres`.
- **Tenancy-Aware Authentication**: Accept optional `tenant_id` on `POST /auth/login`. Support unique resolution by email or disambiguation across tenants.
- **Strict Multi-Tenancy**: Remove silent fallback to `"default"` tenant in task services; require explicit cryptographically validated tenant context.
- **Credential & Config Hygiene**: Gate developer seed users behind `SEED_DEV_DATA=true`; fail fast in production if default JWT secrets are detected.
- **User Management HTTP Endpoints**: Expose `POST /auth/users` (admin only) and `GET /auth/me` for authenticated user profile retrieval.
- **Serverless CORS Hardening**: Restrict CORS `AllowOrigins` in SAM template via configurable parameters.

## Capabilities

### Modified Capabilities
- `auth-and-access-control`: Add hardened IP extraction, memory bounded rate limiting, 32-byte secret validation, persistent PostgreSQL user repository, and user management endpoints.
- `task-service`: Enforce strict tenant boundary check without silent fallback to `"default"`.
- `aws-serverless-adapter`: Parameterize CORS origin configuration and wire persistent user repository in serverless initialization.

## Impact

- **APIs**: `POST /auth/login` accepts optional `tenant_id`. New routes `POST /auth/users` and `GET /auth/me` are registered.
- **Storage**: `users` table in PostgreSQL is actively queried and mutated when `STORAGE_DRIVER=postgres`.
- **Security**: Complete mitigation of CWE-290 (IP spoofing) and CWE-400 (unbounded memory growth).
