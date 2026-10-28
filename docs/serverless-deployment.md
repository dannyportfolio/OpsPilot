# AWS Serverless Deployment Architecture & Operational Runbook

This document describes the serverless architecture, runtime requirements, compilation workflow, least-privilege IAM policies, and operational safety controls for deploying OpsPilot on AWS Lambda and Amazon API Gateway.

---

## 1. Architecture Overview

OpsPilot uses a decoupled Hexagonal Architecture where business domains (`internal/auth`, `internal/task`) return standard Go `http.Handler` instances. The serverless adapter ([`internal/platform/lambda`](file:///C:/Users/Lenovo%20X13/Documents/OpsPilot/internal/platform/lambda/adapter.go)) bridges incoming Amazon API Gateway HTTP API v2 events directly into the secured HTTP router without altering domain logic or transport defenses.

```
Internet (HTTPS)
       │
       ▼
Amazon API Gateway (HTTP API v2, $default stage)
       │
       ▼ (APIGatewayV2HTTPRequest)
AWS Lambda (provided.al2023 / ARM64 Graviton)
       │
       ├─► internal/platform/lambda (Bidirectional Event Bridge)
       ├─► internal/auth/middleware (JWT Signature & Claims Validation)
       ├─► internal/task (Tenant Scoping & Domain Invariants)
       │
       ▼ (Database Connection Pool: MaxOpenConns=2)
Private Subnet (VPC)
       │
       ▼
AWS RDS Proxy / PostgreSQL (Multi-Tenant Relational Storage)
```

---

## 2. Runtime & Compilation Standards

### 2.1 Supported AWS Lambda Runtime
- **Runtime**: `provided.al2023` (Amazon Linux 2023 custom runtime).
- **Architecture**: `arm64` (AWS Graviton2/3, providing ~20% lower cost and higher performance) or `x86_64`.
- **Executable Name**: Must be named exactly `bootstrap` inside the root of the deployment zip archive.

> [!IMPORTANT]
> The legacy `go1.x` runtime was formally deprecated by AWS on December 31, 2023. All modern Go serverless functions must target `provided.al2023` or `provided.al2`.

### 2.2 Compilation Command
```bash
# Clean previous builds
rm -f bootstrap bootstrap.zip

# Cross-compile for Linux ARM64 with static linking (zero CGO)
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags lambda.norpc -ldflags="-s -w" -o bootstrap ./cmd/lambda

# Package executable for deployment
zip bootstrap.zip bootstrap
```

---

## 3. Serverless Database Connection Architecture

### 3.1 Concurrency & Connection Pool Sizing
In traditional containerized environments (`cmd/server`), 1 process handles hundreds of concurrent HTTP requests with a pool of 25 connections.

In AWS Lambda, scaling occurs horizontally across independent execution environments. If 50 concurrent Lambda containers open 25 connections each, they will attempt to open 1,250 connections, quickly exhausting PostgreSQL's `max_connections` (typically 100 on standard RDS instances).

To protect database stability:
1. **Per-Container Pool Limits**: Configured in [`cmd/lambda/main.go`](file:///C:/Users/Lenovo%20X13/Documents/OpsPilot/cmd/lambda/main.go#L44-L48):
   - `db.SetMaxOpenConns(2)`
   - `db.SetMaxIdleConns(2)`
   - `db.SetConnMaxLifetime(5 * time.Minute)`
   - `db.SetConnMaxIdleTime(2 * time.Minute)`
2. **Cold-Start Reuse**: Database connection pools and JWT token services are initialized in global/package scope (`buildApp()`), enabling warm connection reuse across subsequent invocations without repeated TCP handshakes.
3. **Production Recommendation**: Deploy **AWS RDS Proxy** between the Lambda function and PostgreSQL. RDS Proxy pools thousands of concurrent Lambda connections down to a fixed set of database connections, multiplexing transactions safely.

---

## 4. Least-Privilege IAM Execution Policy

The Lambda execution role ([`OpsPilotFunctionRole`](file:///C:/Users/Lenovo%20X13/Documents/OpsPilot/template.yaml#L66-L91)) follows strict least-privilege principles:

### Standard Execution Policy:
- `arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole`
  - `logs:CreateLogGroup`
  - `logs:CreateLogStream`
  - `logs:PutLogEvents`

### Private VPC Attachment Policy (when connected to RDS):
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:CreateNetworkInterface",
        "ec2:DescribeNetworkInterfaces",
        "ec2:DeleteNetworkInterface",
        "ec2:AssignPrivateIpAddresses",
        "ec2:UnassignPrivateIpAddresses"
      ],
      "Resource": "*"
    }
  ]
}
```
*No S3, DynamoDB, IAM manipulation, or wildcard administrative policies are granted.*

---

## 5. Local Testing & Verification Workflow

### 5.1 Automated Unit & Parity Tests
Run the serverless adapter test suite and HTTP parity suite:
```bash
# Verify Lambda adapter and HTTP parity
go test -v ./internal/platform/lambda/... ./cmd/lambda/...
```

### 5.2 Local Emulation via AWS SAM CLI
Ensure [AWS SAM CLI](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/install-sam-cli.html) is installed:
```bash
# 1. Build the SAM application
sam build

# 2. Start local API Gateway on port 8080
sam local start-api --port 8080

# 3. Test health check probe
curl -i http://localhost:8080/health

# 4. Test login
curl -i -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@opspilot.internal","password":"AdminSecure123!"}'
```

---

## 6. Cost & Operational Risk Checklist

| Risk Area | Severity | Impact | Mitigation Strategy |
|---|---|---|---|
| **Cold Starts** | Low | First invocation of an idle container incurs ~100-200ms startup latency. | Pure Go compiled on `provided.al2023` has minimal runtime overhead. Global initialization reuses database and auth resources. |
| **Database Connection Exhaustion** | High | Burst concurrency can exhaust PostgreSQL connection limits. | Per-container pool capped at `MaxOpenConns: 2`. Deploy AWS RDS Proxy for high-traffic environments. |
| **Payload Size Exceeded** | Medium | API Gateway rejects payloads > 10 MB (6 MB for Lambda synchronous invocations). | OpsPilot enforces inbound payload bounding of 1 MB via `http.MaxBytesReader` across all endpoints. |
| **Execution Timeouts** | Medium | Long-running queries could hold Lambda containers open until function timeout (20s). | Application contexts pass timeout deadlines down to `sql.DB.QueryRowContext`, cancelling database queries when timeout approaches. |
| **Unbounded Cloud Spend** | High | Traffic spikes or DDoS could generate high invocation charges. | Set API Gateway throttling limits (e.g. 100 RPS burst) and configure AWS Budgets with billing alarms at $50/month. |

---

## 7. Deployment Approval Commands

> [!CAUTION]
> Do not execute cloud deployments without explicit human authorization. All deployment commands require valid AWS credentials configured in the environment.

When authorized, execute the deployment following these exact steps:

```bash
# Step 1: Validate SAM template
sam validate --lint

# Step 2: Build serverless artifacts
sam build

# Step 3: Interactive guided deployment (prompts for Stack Name, Region, Parameters)
sam deploy --guided \
  --stack-name opspilot-staging \
  --parameter-overrides AppEnvironment=staging DatabaseURL="<RDS_POSTGRES_URL>" JWTSecret="<32_BYTE_SECRET>"
```
