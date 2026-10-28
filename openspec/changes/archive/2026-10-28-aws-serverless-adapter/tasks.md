# Tasks: AWS Serverless Adapter

## 1. Serverless Adapter & Event Translation

- [x] 1.1 Add `github.com/aws/aws-lambda-go` to `go.mod` and implement `internal/platform/serverless` mapping API Gateway HTTP API v2 requests and responses to standard `http.Handler`; verify with tests in `internal/platform/serverless/adapter_test.go`.
- [x] 1.2 Implement context deadline and cancellation forwarding, trace ID propagation, and cookie/query parameter mapping; verify with tests in `internal/platform/serverless/context_test.go`.

## 2. Lambda Entrypoint & Concurrency Architecture

- [x] 2.1 Implement `cmd/lambda/main.go` targeting `provided.al2023`, configuring cold-start resource caching, serverless database pool limits (`MaxOpenConns: 2`), and routing; verify with tests in `cmd/lambda/main_test.go`.
- [x] 2.2 Add representative parity tests comparing response bodies, status codes, and error envelopes between local HTTP server and Lambda adapter in `internal/platform/serverless/parity_test.go`.

## 3. Infrastructure as Code & Deployment Blueprint

- [x] 3.1 Create AWS SAM template `template.yaml` defining HTTP API Gateway, Lambda function (`provided.al2023`), least-privilege IAM policies, and VPC configuration parameters.
- [x] 3.2 Author `docs/serverless-deployment.md` documenting compilation (`GOOS=linux go build -o bootstrap`), local SAM testing, cost and operational risks, and deployment approval commands.
