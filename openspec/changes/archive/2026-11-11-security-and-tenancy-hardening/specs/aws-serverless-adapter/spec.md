# Spec Delta: aws-serverless-adapter

## MODIFIED Requirements

### Requirement: SAM Infrastructure as Code and Least-Privilege IAM
The system SHALL provide an AWS Serverless Application Model (SAM) configuration targeting the `provided.al2023` runtime, compiling a standalone Go `bootstrap` binary, routing all HTTP methods and paths via API Gateway HTTP API, restricting IAM permissions, and parameterizing CORS `AllowOrigins`.

#### Scenario: Least-privilege IAM policy definition
- **WHEN** the SAM template is validated
- **THEN** the function role grants only `AWSLambdaBasicExecutionRole` and VPC interface attachment permissions without wildcard administrative capabilities

#### Scenario: Configurable CORS origins
- **WHEN** the SAM template is deployed with specified authorized origins
- **THEN** API Gateway configures CORS AllowOrigins to match the configured parameter rather than an unrestricted wildcard
