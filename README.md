# SUGO — SuggestGreatOutput

Go CLI (Cobra) that helps human reviewers prepare for GitHub PR reviews. It fetches a PR, runs independent analysis agents in parallel, and produces a structured report of findings. It is **not** an automated reviewer — it does not post comments on PRs.

## Build & Run

```bash
go build -o sugo .                       # compile
go run . review owner/repo#42            # run without installing
go install .                             # install to $GOPATH/bin
```

## Test & Lint

```bash
go test ./...                            # all tests
go test ./internal/agents/... -v         # single package, verbose
go test -race ./...                      # race detector
golangci-lint run ./...                  # lint this project (sugo itself uses golangci-lint)
```

Always run `go test ./...` and `golangci-lint run` before considering a task done. Both must pass.

## Project Layout

```
cmd/                   # Cobra command definitions (root, review)
internal/
  orchestrator/        # Fetches PR via `gh` CLI, dispatches agents, aggregates findings
  agents/
    rules/             # Rules checker — deterministic + LLM enrichment
    lint/              # Lint checker — runs configurable linter (language-agnostic) + LLM enrichment
    logic/             # Logic agent — LLM per changed file
    focus/             # Focus agent — LLM on full diff summary
    analysisgap/       # Analysis-gap agent — Jira API + LLM validation
  finding/             # Shared Finding type and severity constants
  config/              # YAML config loader (.sugo.yaml)
  renderer/            # Formats the final report (terminal, JSON, markdown)
  llm/                 # LLM client abstraction (prompt building, API calls)
  gh/                  # Thin wrapper around the `gh` CLI for PR fetching
pkg/                   # Public library code (if any)
testdata/              # Golden files, fixture diffs, sample configs
```

## Architecture

```
sugo review owner/repo#42
        │
        ▼
  Orchestrator
  1. Fetch PR metadata + diff via `gh` CLI
  2. Dispatch agents concurrently (errgroup)
  3. Aggregate []Finding
  4. Render report
        │
        ├── Rules agent   → []Finding (type: Fix)
        ├── Lint agent    → []Finding (type: Fix)
        ├── Logic agent   → []Finding (type: AttentionPoint)
        ├── Focus agent   → []Finding (type: AttentionPoint)
        └── AnalysisGap   → []Finding (type: Fix)
```

All agents implement a shared `Agent` interface and run in parallel via `errgroup`. Each returns `[]Finding`; the orchestrator merges and deduplicates before rendering.

## Finding Schema

Every agent must output findings matching this JSON structure:

```jsonc
{
  "findings": [{
    "agent": "rules",               // agent name
    "severity": "high",             // high | medium | low
    "location": {
      "file": "path/to/file.go",
      "line_start": 12,
      "line_end": 24
    },
    "message": "human-readable",
    "fix": "suggested fix or null"  // null for AttentionPoint agents
  }]
}
```

Do not change this schema without updating all agents and the renderer.

## Agent Details

| Agent        | Input                                   | Deterministic Pass              | LLM Pass                                    | Output Type     |
|--------------|-----------------------------------------|---------------------------------|----------------------------------------------|-----------------|
| Rules        | PR metadata (branch, commits, title, labels, description) | Regex on branch, conventional commit parse, required labels | Suggest fixes for violations; fully delegate ambiguous rules | Fix             |
| Lint         | Changed files                           | Run linter defined in `.sugo.yaml` (e.g. `golangci-lint`, `eslint`, `ruff`) | Explain each finding, suggest fix            | Fix             |
| Logic        | Changed files (per file)                | —                               | Analyze each changed file for logic issues   | AttentionPoint  |
| Focus        | Full diff                               | —                               | Summarize diff, rank files by review priority | AttentionPoint  |
| AnalysisGap  | Jira ticket + diff                      | —                               | Validate diff covers task requirements       | Fix             |

## Code Conventions

- Go 1.25+. Use `errors.Join`, `slog` for structured logging, and generics where they simplify.
- Follow standard Go project layout. No `src/` directory.
- All exported types and functions need doc comments.
- Error handling: wrap with `fmt.Errorf("context: %w", err)`. Never silently swallow errors.
- Use `context.Context` as the first parameter on any function that does I/O or spawns goroutines.
- Table-driven tests with `t.Run` subtests. Use `testdata/` for golden files.
- No global mutable state. Pass dependencies explicitly (constructor injection or functional options).
- Cobra commands go in `cmd/`. Business logic stays in `internal/`.

## Configuration

The CLI reads `.sugo.yaml` from the repo root (overridable with `--config`). It defines:

- Branch name patterns (regex)
- Conventional commit rules
- Required PR labels
- Lint agent configuration: linter command, args, paths, and severity overrides (language-agnostic — supports `golangci-lint`, `eslint`, `ruff`, etc.)
- LLM provider settings: provider name, model, base URL (API key is **never** in config — see below)
- Custom rules-agent prompt (`--rules-agent=path.md`)

## Dependencies & Tooling

- **gh CLI** must be installed and authenticated (`gh auth status`).
- **Linter binary** matching the `.sugo.yaml` lint config must be on `$PATH` (e.g. `golangci-lint`, `eslint`).
- **Jira API** credentials (env vars or config) for the analysis-gap agent.
- **LLM**: provider, model, and base URL are read from `.sugo.yaml`. Only the API key comes from the environment (`SUGO_LLM_API_KEY`). Default provider for the first implementation is **mimir**.

## Workflow

- Run `go test ./... && <linter> run` locally before considering a task done. Both must pass.
- Do not use git commands (no commits, no branching). Just write and modify code directly.

## Common Tasks

```bash
# Add a new agent
# 1. Create internal/agents/<name>/<name>.go implementing the Agent interface
# 2. Register it in the orchestrator's agent list
# 3. Add tests in internal/agents/<name>/<name>_test.go
# 4. Add golden test fixtures in testdata/

# Run a single agent in isolation (useful for debugging)
go test ./internal/agents/rules/... -run TestRulesAgent -v
```

## Compaction Hint

When compacting, always preserve: the full Agent interface contract, the Finding schema, the list of all five agents and their output types, and any test commands.
