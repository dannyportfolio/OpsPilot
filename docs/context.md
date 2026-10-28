# Project Context & Agent Quick-Reference

This document is the fast-start reference for AI agents working in OpsPilot. Consult this file before inspecting code to avoid broad repository scans.

---

## 1. Quick Facts

- **Repository**: `OpsPilot`
- **Language**: Go 1.22+ (Idiomatic standard library, zero external framework dependencies)
- **Primary Domain**: Serverless-ready Task API foundation expanding into reusable ERP/CRM systems.
- **Specification Engine**: OpenSpec (`openspec/`)
- **Coding Agents**: OpenCode (`.opencode/agents/`)
- **UI & Browser QA**: Antigravity (Visual review, browser verification)
- **Tenancy Model**: Server-side multi-tenancy enforcement on all database queries and repositories.

---

## 2. Directory Layout at a Glance

```
OpsPilot/
├── cmd/server/main.go            # HTTP server entrypoint with graceful shutdown
├── internal/
│   ├── platform/response/        # Standard JSON envelopes (JSON(), Error())
│   └── task/                     # Task domain: entity, repository interface, in-memory store, REST handlers
├── openspec/                     # OpenSpec root (changes, specs, config.yaml)
├── .opencode/agents/             # Specialized OpenCode agents (architect, backend, frontend, etc.)
├── docs/                         # Architecture, decisions (ADRs), workflows, and model routing
├── AGENTS.md                     # Non-negotiable engineering contract
└── README.md                     # Project overview and runbook
```

---

## 3. Standard Commands

| Task | Command |
|---|---|
| Run Unit/Integration Tests | `go test -v ./...` |
| Run Race Detector | `go test -race ./...` |
| Run Static Analysis | `go vet ./...` |
| Format Check | `gofmt -l .` |
| OpenSpec Health Check | `openspec doctor` |
| Validate Specs & Changes | `openspec validate --all` |
| List OpenCode Agents | `opencode agent list` |

---

## 4. Token-Efficiency Rules for Agents

1. **Do Not Re-scan Codebase**: If you need to understand how an endpoint or entity works, read only `internal/<domain>/<domain>.go`, `handler.go`, or `repository.go`.
2. **Follow OpenSpec Changes**: Active changes live under `openspec/changes/<change-id>/`. That is your specification contract.
3. **Keep Summaries Direct**: Report files modified, tests executed with exit code, and remaining items. Do not repeat architectural boilerplate.
