# Spec Delta: task-service

## MODIFIED Requirements

### Requirement: Task Entity Lifecycle Management
The system SHALL maintain task entities with an RFC 4122 v4 UUID, title, description, status enum, and UTC timestamps. The domain entity SHALL represent pure business state independent of external transport formats, while transport adapters manage DTO serialization.

#### Scenario: Default status on creation
- **WHEN** a task is created without specifying a status
- **THEN** the system assigns the status `pending` and sets created_at and updated_at to the current UTC timestamp

#### Scenario: Valid status transition
- **WHEN** a task status is updated from `pending` to `in_progress` or `completed`
- **THEN** the system records the new status and updates updated_at timestamp
