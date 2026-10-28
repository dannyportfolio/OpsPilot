# Model Routing & Cost Control Guide

This guide describes how to configure model access across OpenCode, Antigravity, OpenRouter, and the optional 9Router local gateway, including cost-effective model tiering.

---

## 1. Model Access Architecture

OpsPilot supports flexible, portable model routing without committing credentials or environment-specific configs to version control:

```
┌────────────────────────────────────────────────────────┐
│                        Clients                         │
│   OpenCode CLI (Agents)          Antigravity (IDE/QA)  │
└───────────────┬───────────────────────────┬────────────┘
                │                           │
                │ Direct or Routed          │ Direct Built-in
                ▼                           ▼
┌───────────────────────────────┐   ┌────────────────────┐
│      OpenRouter Direct        │   │    Antigravity     │
│   (https://openrouter.ai/api) │   │ Provider / Model   │
└───────────────▲───────────────┘   └────────────────────┘
                │ Optional Gateway
┌───────────────┴───────────────┐
│        9Router Gateway        │
│    (http://127.0.0.1:20128)   │
└───────────────────────────────┘
```

### 1.1 Direct OpenRouter Access in OpenCode
When accessing OpenRouter directly:
1. Export the API key in your local shell (external to git, never committed):
   ```powershell
   $env:OPENROUTER_API_KEY = "your-key-here"
   ```
2. OpenCode connects natively to OpenRouter using standard model naming (e.g., `openrouter/<provider>/<model>` or as configured via `opencode providers login`).

### 1.2 Optional 9Router Gateway
`9router` is a local multiplexing gateway running on `http://127.0.0.1:20128`:
- Can translate endpoints, manage rate limits, and provide a unified proxy for multiple CLI tools.
- **Verification Rule**: Never assume two routers (e.g. 9Router → OpenRouter) can be chained safely without testing:
  - Streaming SSE compatibility.
  - Function / Tool calling support.
  - Canonical model ID mapping.
  - Authentication header propagation.
- To inspect local 9Router status without exposing secrets:
  ```powershell
  9router show
  ```

### 1.3 Antigravity Model Access
Antigravity manages its model selection through its native CLI/IDE settings (e.g., configured Gemini, Claude, or custom endpoints). Antigravity operates independently of OpenCode's local model configurations, ensuring robust browser verification and interactive pair-programming.

---

## 2. Cost Control & Model Tiering Strategy

To minimize token usage and operating costs while preserving high engineering quality, follow a two-tier strategy:

| Tier | Purpose | Recommended Model Classes | Usage Context |
|---|---|---|---|
| **Tier 1: Fast & Cost-Effective** | Exploration, documentation, quick task updates, routine unit tests, code formatting | Fast, lightweight models (e.g. Flash-class, Haiku-class, Mini-class) | - `openspec` planning exploration<br/>- `docs/` updates<br/>- Routine table-driven test boilerplate<br/>- Refactoring simple functions |
| **Tier 2: Advanced Reasoning & Audit** | Complex architecture, domain modeling, concurrency design, security auditing | High-reasoning coding models (e.g. Sonnet-class, Pro-class, Codex-class) | - `architect` agent proposals<br/>- Multi-tenancy isolation design<br/>- `security` agent audit reviews<br/>- Complex concurrency race debugging |

> [!TIP]
> Keep model identifiers configurable per developer environment rather than hardcoding specific vendor strings in project files. Let `opencode.json` inherit defaults or developer-level environment settings.

---

## 3. Configuration Safety & Portability

1. **Keep Secrets External**: Always inject API keys via environment variables (`OPENROUTER_API_KEY`, etc.).
2. **Never Edit Global Settings from Agents**: Avoid modifying `~/.config/opencode/` or user-level credentials automatically. All project configuration is scoped locally in `opencode.json` and `.opencode/`.
3. **Verify Before Chaining**: If configuring a proxy, test a simple ping or health check before running autonomous agent loops.
