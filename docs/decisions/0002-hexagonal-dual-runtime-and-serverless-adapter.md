# ADR 0002: Hexagonal Dual-Runtime Architecture and AWS Serverless Adapter

- **Status**: Accepted
- **Date**: 2026-10-28
- **Deciders**: Senior Go Serverless Architect, Application Lead
- **Consulted**: Backend Engineer, QA Engineer, DevOps Lead

---

## Context and Problem Statement

OpsPilot requires deployment flexibility: running locally or in traditional containerized environments (Docker / ECS / K8s) as a long-lived HTTP server, while also operating seamlessly as an on-demand serverless function (AWS Lambda behind API Gateway HTTP API v2) to achieve near-zero idle compute cost. Core domain and application logic must remain pure Go, free from vendor lock-in or cloud-specific framework bloat.

## Decision Drivers

- **Zero Domain Pollution**: Core business rules and HTTP handlers must not depend on AWS SDK or Lambda event definitions (`events.APIGatewayV2HTTPRequest`).
- **Idiomatic Go Standard Library**: All routers must yield standard `http.Handler` instances (`net/http`).
- **Exact Contract Parity**: HTTP responses, status codes, error payloads, and header semantics must be identical between local standalone HTTP servers and AWS Lambda functions.
- **Serverless Database Safety**: Transient Lambda concurrency spikes must not exhaust PostgreSQL database connection pools.

## Considered Options

1. **Monolithic Cloud Framework (e.g., Gin with vendor Lambda bridge)**: Fast setup, but couples code to external router dependencies and breaks the standard library contract.
2. **Dual-Codebase / Separate Lambda Handlers**: Maintaining distinct handlers for HTTP and serverless leads to drift, code duplication, and regression risks.
3. **Hexagonal Architecture with Bidirectional In-Memory Adapter**: Decoupled domain and `http.Handler` routes wrapped by an in-memory event translator (`internal/platform/lambda`).

## Decision Outcome

Chosen option: **Option 3: Hexagonal Architecture with Bidirectional In-Memory Adapter**.

### Architectural Design:
- **`cmd/server/main.go`**: Mounts `taskRouter` and `authRouter` into standard `http.Server` with graceful SIGINT/SIGTERM shutdown.
- **`cmd/lambda/main.go`**: Initializes the identical domain handlers, wrapping the root `http.Handler` with `lambda.New(handler)`.
- **`internal/platform/lambda`**: Translates incoming `events.APIGatewayV2HTTPRequest` into `http.Request`, executes `handler.ServeHTTP(rec, req)`, and captures `httptest.ResponseRecorder` output into `events.APIGatewayV2HTTPResponse`.
- **Database Tuning**: Serverless entrypoints cap pool limits to `MaxOpenConns: 2` and `MaxIdleConns: 2` to protect RDS instances from connection exhaustion under high concurrency.
- **Header Propagation**: Propagates execution deadlines (`ctx.Deadline()`) and AWS X-Ray trace headers (`X-Amzn-Trace-Id`).

### Positive Consequences:
- 100% of domain logic and tests run without AWS infrastructure or credentials.
- Exact behavioral parity verified by automated table-driven tests (`internal/platform/lambda/parity_test.go`).
- ARM64 Graviton runtime (`provided.al2023`) provides lowest execution cost and sub-15ms cold starts.

### Trade-offs & Mitigations:
- In-memory event translation incurs negligible microsecond CPU overhead per request.
- Unpooled Lambda instances require external connection pooling (e.g. AWS RDS Proxy) for enterprise workloads exceeding hundreds of concurrent executions.
