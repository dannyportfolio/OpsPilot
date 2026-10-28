# Spec Delta: Authentication, RBAC, and Multi-Tenant Access Control

## Purpose

Defines user authentication, password hashing, signed JWT token management, rate limiting, role-based access control, and strict multi-tenant boundary enforcement for OpsPilot APIs.

## ADDED Requirements

### Requirement: User Identity and Password Storage
The system SHALL manage user accounts with unique email, bcrypt-hashed passwords (cost >= 10), role assignment (`admin`, `member`, `viewer`), tenant identifier, and active status. The system SHALL never store or return plaintext passwords.

#### Scenario: User entity validation
- **WHEN** a user account is registered or created with valid attributes
- **THEN** the system generates a UUID v4 identifier, computes a bcrypt hash of the password, and stores the user under the specified tenant_id

#### Scenario: Plaintext password redaction
- **WHEN** user entities or responses are serialized or queried
- **THEN** the password hash is excluded from API responses and plain passwords are zeroed from memory buffers

### Requirement: Authentication and Login Endpoint
The system SHALL expose `POST /auth/login` accepting `email` and `password`. The endpoint SHALL verify credentials against stored bcrypt hashes and return a signed Bearer token with UTC expiration.

#### Scenario: Successful login
- **WHEN** a client submits valid email and password credentials for an active account
- **THEN** the system returns HTTP status 200 OK with a signed JWT access token, expiration timestamp, and user profile information

#### Scenario: Invalid credentials rejection without account enumeration
- **WHEN** a client submits a non-existent email or an incorrect password
- **THEN** the system returns HTTP status 401 Unauthorized with a generic error `{"error": "invalid credentials"}` without disclosing whether the email exists

#### Scenario: Suspended account login rejection
- **WHEN** a client attempts login with valid credentials for an account marked suspended
- **THEN** the system rejects authentication with HTTP status 403 Forbidden with `{"error": "account suspended"}`

### Requirement: Token Issuance and Cryptographic Validation
The system SHALL issue HMAC-SHA256 signed JWT tokens containing `sub` (user ID), `tenant_id`, `role`, `email`, `iss` (`opspilot`), `aud` (`opspilot-api`), `jti` (unique token ID), and an `exp` expiration claim.

#### Scenario: Valid token authorization
- **WHEN** an authenticated client sends an API request with header `Authorization: Bearer <valid_token>`
- **THEN** the server validates signature, issuer, audience, and expiration, injecting user and tenant claims into the request context

#### Scenario: Expired token rejection
- **WHEN** a client submits a request with an expired access token
- **THEN** the server denies access with HTTP status 401 Unauthorized with `{"error": "invalid or expired token"}`

#### Scenario: Missing or malformed token rejection
- **WHEN** a client requests a protected endpoint without an Authorization header or with invalid token structure
- **THEN** the server denies access with HTTP status 401 Unauthorized

### Requirement: Token Revocation and Logout
The system SHALL expose `POST /auth/logout` requiring an authenticated Bearer token. Upon invocation, the token's `jti` SHALL be recorded in a revocation registry until expiration, preventing subsequent use.

#### Scenario: Successful token revocation
- **WHEN** an authenticated user calls `POST /auth/logout`
- **THEN** the system records the token identifier in the revocation store and returns HTTP status 200 OK

#### Scenario: Subsequent request using revoked token
- **WHEN** a client attempts to access a protected endpoint using a revoked token
- **THEN** the server denies the request with HTTP status 401 Unauthorized with `{"error": "token revoked"}`

### Requirement: Authentication Rate-Limiting
The system SHALL apply rate-limiting to `POST /auth/login` to prevent brute force credential attacks, bounding consecutive attempts per client IP or target account.

#### Scenario: Rate limit exceeded on login
- **WHEN** requests from an IP or targeting an account exceed 5 failed attempts within 60 seconds
- **THEN** the server rejects further attempts with HTTP status 429 Too Many Requests and an appropriate error envelope

### Requirement: Multi-Tenant Data Isolation
The system SHALL enforce tenant scoping on all protected domain entities. Tenant context SHALL be derived strictly from verified authentication tokens and cannot be overridden by client request parameters or headers.

#### Scenario: Scoped list and retrieval by tenant
- **WHEN** an authenticated user belonging to Tenant A requests task lists or records
- **THEN** the system returns only records matching Tenant A's tenant_id

#### Scenario: Rejection of cross-tenant access
- **WHEN** an authenticated user belonging to Tenant A attempts to access or mutate a resource belonging to Tenant B by ID
- **THEN** the system rejects the request with HTTP status 404 Not Found to prevent leaking the existence of other tenants' records

#### Scenario: Client attempt to spoof tenant ID
- **WHEN** a client provides a custom tenant header or parameter differing from their authenticated token context
- **THEN** the server ignores or rejects the spoofed identifier and enforces the cryptographically verified token tenant_id

### Requirement: Role-Based Access Control (RBAC)
The system SHALL enforce permissions based on user roles: `admin` (full create, read, update, delete within tenant), `member` (create, read, update within tenant; delete forbidden), and `viewer` (read-only within tenant).

#### Scenario: Admin role permissions
- **WHEN** a user with `admin` role performs any CRUD operation on resources within their tenant
- **THEN** the system permits the operation and processes the request

#### Scenario: Member role forbidden from deletion
- **WHEN** a user with `member` role attempts `DELETE /tasks/{id}`
- **THEN** the server rejects the request with HTTP status 403 Forbidden with `{"error": "insufficient permissions"}`

#### Scenario: Viewer role forbidden from mutations
- **WHEN** a user with `viewer` role attempts `POST /tasks` or `PUT /tasks/{id}`
- **THEN** the server rejects the request with HTTP status 403 Forbidden with `{"error": "insufficient permissions"}`

### Requirement: Security Audit Logging and Secret Sanitization
The system SHALL produce structured audit logs for security events including login successes, login failures, rate-limit triggers, and access denials. The system SHALL redact passwords, credentials, and full authorization tokens from all log output.

#### Scenario: Audit log on failed login
- **WHEN** a login attempt fails due to invalid credentials
- **THEN** the system emits an audit log event with action `auth.login_failed`, recording client IP and email, but omitting the submitted password

#### Scenario: Audit log on permission denial
- **WHEN** a user is denied access due to insufficient role permissions or cross-tenant access attempt
- **THEN** the system emits an audit warning event recording tenant_id, user_id, requested URI, and attempted action
