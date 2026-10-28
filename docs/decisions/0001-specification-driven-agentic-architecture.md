# ADR 0001: Specification-Driven Agentic Engineering Architecture

- **Status**: Accepted
- **Date**: 2026-10-27
- **Deciders**: Senior Software Architect, AI Engineering Workflow Engineer
- **Consulted**: Backend Engineer, QA Engineer, Security Reviewer

---

## Context and Problem Statement

OpsPilot requires a reliable, repeatable, and cost-effective multi-agent workflow for implementing enterprise ERP and CRM applications in Go. In typical unstructured AI engineering, agents hallucinate missing requirements, write untested business logic, perform unneeded full-repo re-scans consuming excessive tokens, and inadvertently break tenant isolation boundaries.

## Decision Drivers

- Need strict, verifiable requirements before writing code.
- Prevent regressions in domain logic and multi-tenancy rules.
- Reduce token consumption through targeted context loading and token-efficient agents.
- Clear separation of responsibilities between planning, coding, QA, and security review.
- Support both direct OpenRouter model routing and optional local gateways (9Router).

## Considered Options

1. **Ad-hoc Chat / Monolithic Coding Prompt**: Single prompt executing implementation without structured planning or specs.
2. **Traditional Issue Trackers / GitHub Issues Only**: External tracking without machine-readable delta specifications or local agent tooling.
3. **OpenSpec + OpenCode + Antigravity Multi-Agent Architecture**: Specification-first with OpenSpec for change proposals and delta specs; OpenCode for CLI engineering tasks; Antigravity for UI/UX and browser visual QA; review-only security agent.

## Decision Outcome

Chosen option: **Option 3: OpenSpec + OpenCode + Antigravity Multi-Agent Architecture**.

### Architecture Overview:
- **OpenSpec**: Acts as the project specification engine (`openspec/changes/`). All non-trivial changes require a proposal, delta spec, and atomic tasks.
- **OpenCode**: Acts as the headless and TUI implementation engine with specialized project-level agents (`.opencode/agents/`):
  - `architect` (review/plan mode)
  - `backend` (Go API & domain logic)
  - `frontend` (UI components & accessibility)
  - `database` (migrations & data integrity)
  - `qa` (automated test authoring & execution)
  - `security` (review-only mode: `edit: deny`)
- **Antigravity**: Coordinates browser verification, visual testing, interactive pair-programming, and artifact presentation.
- **AGENTS.md**: Acts as the shared engineering contract binding all agents and human developers.

### Positive Consequences

- All code changes are verified against explicit Given-When-Then acceptance scenarios.
- The security agent operates in review-only mode (`edit: deny`), preventing premature auto-merging of vulnerabilities.
- Domain logic maintains zero framework dependencies and decoupled standard library `http.Handler` routing.
- Context documents (`docs/context.md`) prevent wasteful full-codebase token dumps.

### Negative Consequences / Trade-offs

- Requires authoring OpenSpec proposals for non-trivial features.
- *Mitigation*: Proportional workflow: small typo or comment fixes can skip full OpenSpec ceremonies.

## Validation & Verification

- `openspec doctor` and `openspec validate --all` confirm specification health.
- `opencode agent list` confirms agent registration and permission configurations.
- `go test -v ./...` and `go vet ./...` confirm code compilation and test suite passage.
