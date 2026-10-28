# Design: AWS Serverless Adapter

## Context
See `proposal.md` for architectural motivation.
OpsPilot's domain layer and transport routers (`internal/task/router.go`, `internal/auth/router.go`) return standard library `http.Handler` instances.
To make the application serverless-ready on AWS Lambda and Amazon API Gateway without altering domain invariants or security middleware, we add a decoupled serverless adapter layer and SAM deployment blueprint.

## Goals / Non-Goals

**Goals:**
- Target official modern AWS Lambda runtime: `provided.al2023` running a standalone Go executable named `bootstrap`.
- Implement high-performance bidirectional event conversion for API Gateway HTTP API v2.0 (`APIGatewayV2HTTPRequest` ↔ `http.Handler` ↔ `APIGatewayV2HTTPResponse`).
- Forward Lambda context deadlines and execution timeouts into `http.Request` context.
- Support warm-container connection reuse with serverless-tuned connection pool limits (`maxOpenConns: 2`).
- Provide an AWS SAM template (`template.yaml`) with least-privilege IAM policies, environment variables, and VPC configuration parameters.
- Provide unit and integration tests verifying HTTP handler execution via the Lambda adapter.

**Non-Goals:**
- Automatic cloud provisioning or running `sam deploy` without user authorization.
- Multi-region active-active database replication.
- Direct AWS SDK calls inside business domains (all AWS-specific code stays strictly in `cmd/lambda` and `internal/platform/serverless`).

## Decisions

### 1. Modern Runtime: `provided.al2023` (Custom Runtime)
- **Decision**: Target AWS Lambda `provided.al2023` on Linux ARM64 (`arm64` Graviton) or x86_64 (`amd64`). The binary is compiled with `GOOS=linux CGO_ENABLED=0` and named `bootstrap`.
- **Rationale**: AWS deprecated the native `go1.x` runtime on December 31, 2023. All modern Go serverless functions on AWS must run on `provided.al2023` or `provided.al2`. Graviton (ARM64) offers up to 20% better price/performance.
- **Alternative considered**: Container image deployment (`PackageType: Image`). Rejected as default because zip deployment with `provided.al2023` has significantly faster cold-start times (sub-100ms) and simpler CI/CD pipelines.

### 2. Event Bridge: Dedicated `serverless` Package
- **Decision**: Implement an idiomatic, dependency-light adapter in `internal/platform/serverless` using `github.com/aws/aws-lambda-go/events`. The adapter converts API Gateway HTTP API v2 requests into `http.Request`, executes the root `http.Handler`, and serializes the recorded response into `events.APIGatewayV2HTTPResponse`.
- **Rationale**: Keeps `cmd/lambda/main.go` clean and testable, while keeping domain packages 100% agnostic to AWS Lambda.
- **Alternative considered**: Third-party framework adapters like `awslabs/aws-lambda-go-api-proxy`. Implementing our own targeted v2 adapter avoids heavy third-party transitive dependencies and adheres to our zero unnecessary dependency principle.

### 3. Serverless Database Connection Pool Tuning
- **Decision**: In serverless mode, set `db.SetMaxOpenConns(2)`, `db.SetMaxIdleConns(2)`, and `db.SetConnMaxLifetime(5 * time.Minute)`.
- **Rationale**: Unlike persistent containers where 1 process handles hundreds of requests, Lambda scales horizontally by provisioning separate execution containers. If 50 Lambda containers open 25 connections each, they will overwhelm PostgreSQL's connection limits. For production workloads with high concurrency, AWS RDS Proxy is recommended.

### 4. Least-Privilege IAM & Security
- **Decision**: Function execution policy includes only:
  - `AWSLambdaBasicExecutionRole` (`logs:CreateLogGroup`, `logs:CreateLogStream`, `logs:PutLogEvents`).
  - `AWSLambdaVPCAccessExecutionRole` (if database lives inside private VPC).
  - No wildcard `*` permissions or IAM privilege escalation.

## Risks / Trade-offs

- **[Risk] Cold Starts**: First invocation of a new Lambda container requires binary startup and database connection initialization.
  - **Mitigation**: Pure Go binaries on `provided.al2023` have low cold start latency (~100-200ms). Connection initialization is non-blocking or uses short connect deadlines.
- **[Risk] Database Connection Exhaustion**: High spike in Lambda concurrency can saturate PostgreSQL max connections.
  - **Mitigation**: Bounded pool per container (2 connections) + documentation recommending AWS RDS Proxy for bursty workloads.
- **[Risk] API Gateway 10 MB Payload Limit**: API Gateway rejects request and response payloads greater than 10 MB (6 MB for Lambda synchronous invocations).
  - **Mitigation**: Our application already enforces an inbound 1 MB limit (`http.MaxBytesReader`), safely below API Gateway limits.

## Deployment Architecture

```
Internet ──▶ API Gateway HTTP API ($default) ──▶ AWS Lambda (provided.al2023 / bootstrap)
                                                         │
                                                  (Private Subnet)
                                                         ▼
                                            PostgreSQL RDS / RDS Proxy
```
