# Spec Delta: task-service

## ADDED Requirements

### Requirement: Strict Tenant Context Requirement
The system SHALL require an authenticated tenant context for all task mutations and queries, rejecting requests without a validated tenant rather than falling back to default shared storage.

#### Scenario: Missing tenant context rejected
- **WHEN** a task creation, retrieval, update, or deletion is attempted without a valid tenant context
- **THEN** the system rejects the operation with an explicit error rather than defaulting to "default"

#### Scenario: Tenant isolation during task queries
- **WHEN** an authenticated user creates or queries tasks
- **THEN** all storage operations strictly filter by the authenticated tenant_id, preventing any cross-tenant data leakage
