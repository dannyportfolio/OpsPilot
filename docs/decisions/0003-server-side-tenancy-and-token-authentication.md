# ADR 0003: Server-Side Multi-Tenancy and Token Authentication

- **Status**: Accepted
- **Date**: 2026-10-28
- **Deciders**: Senior Application Security Engineer, Backend Lead
- **Consulted**: Database Engineer, QA Engineer

---

## Context and Problem Statement

OpsPilot is designed as an ERP/CRM foundation hosting multiple distinct organizations. To satisfy strict multi-tenancy requirements defined in `AGENTS.md`, all stored records must be strictly isolated by `tenant_id`. Furthermore, the API must authenticate users securely, enforce role-based access control (RBAC), and prevent unauthorized cross-tenant data access.

## Decision Drivers

- **Zero Trust in Client Input**: Clients must never be allowed to dictate `tenant_id` via request body or query parameters. Tenant identity must be derived exclusively from cryptographically signed server-issued tokens.
- **Strict Tenant Scoping on Queries**: Every database query and memory repository lookup must include `tenant_id = $1`.
- **Stateless Authentication with Immediate Revocation**: Use HS256-signed JWTs for low-latency validation, combined with an in-memory/cache token revocation registry for instant logout.
- **Brute-Force & Rate-Limiting Protection**: Authentication endpoints (`/auth/login`) must be rate-limited per IP to prevent dictionary attacks.

## Considered Options

1. **Client-Supplied Tenant Headers (e.g. `X-Tenant-ID`)**: High risk of spoofing and lateral data exposure if headers are manipulated.
2. **Database-Per-Tenant**: Maximum physical isolation, but high operational cost and operational complexity for hundreds of small tenants.
3. **Shared Database with Discriminator Column (`tenant_id`) & Token-Derived Context**: Standard SaaS pattern where every table includes `tenant_id`, indexes are composite `(tenant_id, created_at DESC)`, and repositories filter strictly by the token's authenticated tenant.

## Decision Outcome

Chosen option: **Option 3: Shared Database with Discriminator Column (`tenant_id`) & Token-Derived Context**.

### Security & Tenancy Rules:
- **Token Claims**: Access tokens carry `sub` (User ID), `tid` (Tenant ID), `rol` (Role), `iss`, `aud`, and `exp`.
- **Context Injection**: `auth.Authenticate` middleware validates the signature, issuer, audience, and revocation status, then injects `auth.UserContext` into `r.Context()`.
- **Repository Isolation**: `task.Repository` methods (`Create`, `GetByID`, `List`, `Update`, `Delete`) accept `tenantID` as an explicit first parameter. SQL queries bind `WHERE tenant_id = $1`.
- **RBAC Boundaries**:
  - `admin`: Full administrative access (create, list, get, update, delete).
  - `member`: Standard access (create, list, get, update).
  - `viewer`: Read-only access (list, get). Mutation attempts return `403 Forbidden`.
- **Negative Cross-Tenant Tests**: Unit and integration test suites include explicit cross-tenant penetration tests verifying that Tenant A cannot access Tenant B's data even with known valid IDs.

### Positive Consequences:
- Cryptographic proof of tenant identity with zero chance of tenant spoofing via request bodies.
- High database performance via composite indexes (`(tenant_id, created_at DESC)` and `(tenant_id, email)`).
- Complete isolation verified by automated unit and integration tests.

### Trade-offs & Mitigations:
- In-memory revocation and rate-limiting stores are local to the running process or Lambda container.
- *Mitigation for Production*: Recommended future migration to distributed Redis / ElastiCache for multi-node deployments.
