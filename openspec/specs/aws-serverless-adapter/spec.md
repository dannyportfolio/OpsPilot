# aws-serverless-adapter Specification

## Purpose
Enables running the OpsPilot backend on AWS Lambda and Amazon API Gateway, translating serverless events to standard HTTP handlers without modifying business logic.

## Requirements

### Requirement: AWS Lambda API Gateway HTTP Proxy Event Translation
The system SHALL translate Amazon API Gateway HTTP API (payload format v2.0) and REST API proxy events into standard library `http.Request` instances, route them through the secured application `http.Handler`, and map HTTP responses into API Gateway response structures preserving status codes, headers, and response bodies.

#### Scenario: Successful GET request translation
- **WHEN** API Gateway triggers the Lambda handler with an HTTP GET request event
- **THEN** the adapter maps path, method, headers, and query parameters to an http.Request and returns an API Gateway response with HTTP 200 and the JSON payload

#### Scenario: Request with Authorization Bearer header
- **WHEN** API Gateway passes a request with an `Authorization: Bearer <token>` header
- **THEN** the adapter preserves the header and passes it to the AuthMiddleware, correctly establishing user and tenant context

#### Scenario: Validation error response mapping
- **WHEN** a client submits an invalid request payload to API Gateway
- **THEN** the adapter captures the HTTP 400 Bad Request error response and maps it to an API Gateway response with status 400 and the standard error envelope

### Requirement: Context Deadline and Execution Trace Propagation
The system SHALL propagate the AWS Lambda execution context deadline, cancellation signals, and request trace headers into the standard `context.Context` supplied to downstream domain and persistence layers.

#### Scenario: Lambda execution timeout approaching
- **WHEN** a database query or service operation executes while Lambda runtime nears its timeout
- **THEN** the request context receives cancellation, terminating outstanding database and transport operations

#### Scenario: AWS X-Ray trace header propagation
- **WHEN** an incoming API Gateway event includes `X-Amzn-Trace-Id`
- **THEN** the adapter injects the trace identifier into structured logs and downstream context

### Requirement: Serverless Database Connection Pool Management
The system SHALL initialize the database connection pool in package or global initialization scope to enable warm connection reuse across sequential Lambda invocations while bounding maximum open connections to prevent connection exhaustion under burst concurrency.

#### Scenario: Warm container invocation reuse
- **WHEN** a Lambda function processes consecutive requests on the same execution environment
- **THEN** the adapter reuses the existing database connection pool without establishing new physical TCP connections

### Requirement: SAM Infrastructure as Code and Least-Privilege IAM
The system SHALL provide an AWS Serverless Application Model (SAM) configuration targeting the `provided.al2023` runtime, compiling a standalone Go `bootstrap` binary, routing all HTTP methods and paths via API Gateway HTTP API, and restricting IAM permissions to VPC network interface creation and structured CloudWatch log writing.

#### Scenario: Least-privilege IAM policy definition
- **WHEN** the SAM template is validated
- **THEN** the function role grants only `AWSLambdaBasicExecutionRole` and VPC interface attachment permissions without wildcard administrative capabilities
