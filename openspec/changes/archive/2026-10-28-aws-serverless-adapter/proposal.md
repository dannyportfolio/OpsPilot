# Proposal: AWS Serverless Adapter

## Why
OpsPilot is architected using Hexagonal Architecture so that core domain logic, repositories, and HTTP routers remain completely decoupled from the underlying deployment runtime. While `cmd/server/main.go` provides the containerized standalone HTTP server, deploying OpsPilot on AWS requires a dedicated serverless entrypoint compatible with AWS Lambda and Amazon API Gateway without altering domain invariants or multi-tenant security controls.

## What Changes
- **Lambda Runtime Adapter**: Add `cmd/lambda/main.go` targeting the official AWS Lambda runtime `provided.al2023` (standalone Go executable compiled as `bootstrap`).
- **Event Translation**: Implement bidirectional proxy event translation for Amazon API Gateway HTTP API (payload format v2.0) and REST API (v1.0), translating incoming gateway events to standard `http.Request` and capturing `http.ResponseWriter` outputs.
- **Context Deadline & Cancellation**: Forward AWS Lambda context deadlines, cancellation signals, and trace headers (`X-Amzn-Trace-Id`) into the request context.
- **Serverless Database Connection Management**: Configure global database pool caching across warm container invocations with conservative connection limits to prevent exhausting database connection limits during concurrency bursts.
- **Infrastructure as Code (IaC) & Least-Privilege IAM**: Provide an AWS SAM template (`template.yaml`) defining the Lambda function, HTTP API Gateway, environment configuration, VPC attachment parameters, and least-privilege IAM execution roles.
- **Local Serverless Testing Harness**: Provide unit and local integration tests verifying API Gateway event translation, response fidelity, error semantics, and cold-start initialization.

## Capabilities

### New Capabilities
- `aws-serverless-adapter`: Serverless execution adapter mapping AWS Lambda and API Gateway events to standard HTTP handlers, context deadline forwarding, connection management, and SAM deployment definitions.

### Modified Capabilities
*(None; core domain models and secured routers remain unchanged).*

## Impact
- **Entrypoints**: Adds `cmd/lambda/main.go` as a parallel binary entrypoint alongside `cmd/server/main.go`.
- **Dependencies**: Introduces `github.com/aws/aws-lambda-go` for standard Lambda lifecycle primitives.
- **Cloud Infrastructure**: SAM template and documentation provided; no cloud resources created automatically.
