# Spec Delta: aws-serverless-adapter

## MODIFIED Requirements

### Requirement: AWS Lambda API Gateway HTTP Proxy Event Translation
The system SHALL translate Amazon API Gateway HTTP API proxy events into standard library `http.Request` instances, route them through the secured application `http.Handler` assembled via the shared application composition root, and map HTTP responses into API Gateway response structures preserving status codes, headers, and response bodies.

#### Scenario: Successful GET request translation
- **WHEN** API Gateway triggers the Lambda handler with an HTTP GET request event
- **THEN** the adapter maps path, method, headers, and query parameters to an http.Request and returns an API Gateway response with HTTP 200 and the JSON payload

#### Scenario: Request with Authorization Bearer header
- **WHEN** API Gateway passes a request with an `Authorization: Bearer <token>` header
- **THEN** the adapter preserves the header and passes it to the AuthMiddleware, correctly establishing user and tenant context

#### Scenario: Validation error response mapping
- **WHEN** a client submits an invalid request payload to API Gateway
- **THEN** the adapter captures the HTTP 400 Bad Request error response and maps it to an API Gateway response with status 400 and the standard error envelope
