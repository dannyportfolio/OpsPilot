# Spec Delta: auth-and-access-control

## MODIFIED Requirements

### Requirement: User Identity and Password Storage
The system SHALL manage user accounts with unique email, bcrypt-hashed passwords (cost >= 10), role assignment (`admin`, `member`, `viewer`), tenant identifier, and active status. Domain entities SHALL remain free of transport concerns while dedicated DTOs manage API request decoding and response serialization.

#### Scenario: User entity validation
- **WHEN** a user account is registered or created with valid attributes
- **THEN** the system generates a UUID v4 identifier, computes a bcrypt hash of the password, and stores the user under the specified tenant_id

#### Scenario: Plaintext password redaction
- **WHEN** user entities or responses are serialized or queried
- **THEN** the password hash is excluded from API responses and plain passwords are zeroed from memory buffers
