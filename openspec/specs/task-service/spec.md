# task-service Specification

## Purpose
Provides a reusable backend service foundation for ERP and CRM tasks, managing task lifecycle state transitions, robust input validation, relational PostgreSQL persistence, and serverless-ready HTTP endpoints.

## Requirements

### Requirement: Task Entity Lifecycle Management
The system SHALL maintain task entities with an RFC 4122 v4 UUID, title, description, status enum, and UTC timestamps. The status enum SHALL only accept `pending`, `in_progress`, and `completed`.

#### Scenario: Default status on creation
- **WHEN** a task is created without specifying a status
- **THEN** the system assigns the status `pending` and sets created_at and updated_at to the current UTC timestamp

#### Scenario: Valid status transition
- **WHEN** a task status is updated from `pending` to `in_progress` or `completed`
- **THEN** the system records the new status and updates updated_at timestamp

### Requirement: Task Creation Endpoint
The system SHALL expose `POST /tasks` accepting a JSON payload with a required `title` (1-200 characters), optional `description` (0-2000 characters), and optional `status`.

#### Scenario: Successful task creation
- **WHEN** a client sends a valid `POST /tasks` request
- **THEN** the system returns HTTP status 201 Created with a `Location: /tasks/{id}` header and the created task JSON payload

#### Scenario: Unknown field rejection
- **WHEN** a client sends a request body containing unmapped JSON fields
- **THEN** the system rejects the request with HTTP status 400 Bad Request

### Requirement: Task Retrieval Endpoints
The system SHALL expose `GET /tasks` to list tasks ordered newest first, and `GET /tasks/{id}` to retrieve an individual task by its UUID.

#### Scenario: List all tasks
- **WHEN** a client sends a `GET /tasks` request
- **THEN** the system returns HTTP status 200 OK with a JSON array of tasks sorted by created_at in descending order

#### Scenario: Task not found by ID
- **WHEN** a client sends a `GET /tasks/{id}` request with a non-existent UUID
- **THEN** the system returns HTTP status 404 Not Found with a structured error envelope

### Requirement: Task Mutation and Deletion Endpoints
The system SHALL expose `PUT /tasks/{id}` to update an existing task and `DELETE /tasks/{id}` to remove a task.

#### Scenario: Successful task update
- **WHEN** a client sends a valid `PUT /tasks/{id}` request with updated title, description, and status
- **THEN** the system updates the record, increments the updated_at timestamp, and returns HTTP status 200 OK

#### Scenario: Successful task deletion
- **WHEN** a client sends a `DELETE /tasks/{id}` request for an existing task
- **THEN** the system removes the record from storage and returns HTTP status 204 No Content

#### Scenario: Deletion of non-existent task
- **WHEN** a client sends a `DELETE /tasks/{id}` request for a non-existent task with a valid UUID
- **THEN** the system returns HTTP status 404 Not Found

#### Scenario: Deletion with invalid identifier format
- **WHEN** a client sends a `DELETE /tasks/{id}` request with an invalid UUID format
- **THEN** the system returns HTTP status 400 Bad Request

### Requirement: Input Defense and Payload Bounding
The system SHALL bound incoming request bodies to a maximum of 1 MB (1,048,576 bytes) and validate string length boundaries.

#### Scenario: Payload exceeding limit
- **WHEN** a client sends a request payload exceeding 1 MB
- **THEN** the system terminates reading and returns HTTP status 400 Bad Request

#### Scenario: Title validation failure
- **WHEN** a client submits an empty title or a title exceeding 200 characters
- **THEN** the system returns HTTP status 400 Bad Request detailing the field constraint violation

### Requirement: Standard Error Response Format
The system SHALL return all operational errors in a consistent JSON envelope containing an `error` message string and an optional `details` array of field errors.

#### Scenario: Validation error formatting
- **WHEN** a client submits a payload failing validation rules
- **THEN** the system returns HTTP status 400 Bad Request with JSON body `{"error": "validation failed", "details": [{"field": "...", "message": "..."}]}`

### Requirement: Service Health Check
The system SHALL expose `GET /health` to report service operational health for container orchestrators and load balancers.

#### Scenario: Health check probe
- **WHEN** an orchestrator sends `GET /health`
- **THEN** the system returns HTTP status 200 OK with body `{"status": "ok"}`
