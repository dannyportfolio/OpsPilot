# Spec Delta: auth-and-access-control

## MODIFIED Requirements

### Requirement: Authentication Rate-Limiting
The system SHALL apply rate-limiting to `POST /auth/login` to prevent brute force credential attacks, bounding consecutive attempts per client IP or target account. The rate limiter SHALL extract client IP from direct network address unless behind a configured trusted proxy, and SHALL evict expired attempt records to prevent unbounded memory growth.

#### Scenario: Rate limit exceeded on login
- **WHEN** requests from an IP or targeting an account exceed 5 failed attempts within 60 seconds
- **THEN** the server rejects further attempts with HTTP status 429 Too Many Requests and an appropriate error envelope

#### Scenario: Spoofed X-Forwarded-For ignored from untrusted client
- **WHEN** an untrusted client sends arbitrary `X-Forwarded-For` headers with repeated failed login attempts
- **THEN** the system tracks attempts by the actual remote IP address and enforces the rate limit threshold without bypass

#### Scenario: Expired rate limit entries evicted
- **WHEN** rate limiter sliding window and block duration expire for recorded keys
- **THEN** the system removes stale tracking entries from memory, bounding memory footprint

### Requirement: Token Issuance and Cryptographic Validation
The system SHALL issue HMAC-SHA256 signed JWT tokens containing `sub` (user ID), `tenant_id`, `role`, `email`, `iss` (`opspilot`), `aud` (`opspilot-api`), `jti` (unique token ID), and an `exp` expiration claim. The signing secret key SHALL be at least 32 bytes (256 bits) in length.

#### Scenario: Valid token authorization
- **WHEN** an authenticated client sends an API request with header `Authorization: Bearer <valid_token>`
- **THEN** the server validates signature, issuer, audience, and expiration, injecting user and tenant claims into the request context

#### Scenario: Expired token rejection
- **WHEN** a client submits a request with an expired access token
- **THEN** the server denies access with HTTP status 401 Unauthorized with `{"error": "invalid or expired token"}`

#### Scenario: Missing or malformed token rejection
- **WHEN** a client requests a protected endpoint without an Authorization header or with invalid token structure
- **THEN** the server denies access with HTTP status 401 Unauthorized

#### Scenario: Insecure secret key rejection at initialization
- **WHEN** the token service is initialized with a secret containing fewer than 32 bytes
- **THEN** initialization fails with an explicit error requiring at least 32 bytes

## ADDED Requirements

### Requirement: User Administration Endpoints
The system SHALL provide administrative endpoints for managing users within the authenticated tenant boundary, including `POST /auth/users` (restricted to `admin` role) and `GET /auth/me` (accessible to any authenticated user).

#### Scenario: Admin creates new tenant user
- **WHEN** an authenticated user with `admin` role sends `POST /auth/users` with valid email, password, and role
- **THEN** the system creates the user under the admin's tenant_id and returns HTTP status 201 Created with safe user details

#### Scenario: Non-admin denied user creation
- **WHEN** an authenticated user with `member` or `viewer` role calls `POST /auth/users`
- **THEN** the system rejects the request with HTTP status 403 Forbidden

#### Scenario: Current user profile retrieval
- **WHEN** an authenticated user sends `GET /auth/me` with a valid Bearer token
- **THEN** the system returns HTTP status 200 OK with the caller's user profile matching the token subject

### Requirement: Persistent PostgreSQL User Storage
When `STORAGE_DRIVER=postgres`, the system SHALL persist user records to the PostgreSQL `users` table via parameterized queries, enforcing tenant boundary isolation and unique email constraints.

#### Scenario: User persisted in PostgreSQL
- **WHEN** a user is created in PostgreSQL storage driver mode
- **THEN** the user record is stored durably in the `users` table with password hash, tenant_id, role, and status
