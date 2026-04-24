# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build -buildvcs=false -o sugo .                          # compile (repo has no commits yet; -buildvcs=false avoids VCS stamp error)
go test ./...                                               # all tests (must pass before any task is done)
go test ./internal/agents/rules/... -run TestRulesAgent -v  # single package/test
go test -race ./...                                         # race detector
golangci-lint run ./...                                     # lint (must pass before any task is done)
```

Always run both `go test ./...` and `golangci-lint run ./...` before considering any task done.

## Architecture

`sugo review owner/repo#42` flows through the orchestrator:

1. **Fetch** — `internal/gh/` wraps the `gh` CLI to get PR metadata + diff
2. **Dispatch** — `internal/orchestrator/` fans out to all agents concurrently via `sync.WaitGroup` (one failing agent does NOT cancel others)
3. **Aggregate** — merges and deduplicates `[]Finding`
4. **Render** — `internal/renderer/` formats the report (terminal / JSON / markdown)

Five agents, all running in parallel:

| Agent       | Input                     | Output type    |
|-------------|---------------------------|----------------|
| rules       | PR metadata               | Fix            |
| lint        | Changed files             | Fix            |
| logic       | Changed files (per file)  | AttentionPoint |
| focus       | Full diff                 | AttentionPoint |
| analysisgap | Jira ticket + diff        | Fix            |

Each agent has a deterministic pass (regex, linter, Jira API) optionally followed by an LLM enrichment pass. The LLM client abstraction lives in `internal/llm/`.

## Finding Schema

All agents must return findings matching this structure — do not change it without updating all agents and the renderer:

```jsonc
{
  "findings": [{
    "agent": "rules",
    "severity": "high",          // high | medium | low
    "location": { "file": "path/to/file.go", "line_start": 12, "line_end": 24 },
    "message": "human-readable",
    "fix": "suggested fix or null"  // null for AttentionPoint agents
  }]
}
```

## Agent Prompt System

Every LLM-connected agent has a hardcoded default system prompt embedded in the binary via `//go:embed AGENT.md`. Prompts can be overridden at three levels (highest wins):
1. CLI flag: `--<name>-agent=path.md` (e.g. `--rules-agent=custom.md`)
2. `.sugo.yaml` under `agents.<name>.prompt: path/to/custom.md`
3. Embedded `AGENT.md` in the agent's package directory

## Adding a New Agent

1. Create `internal/agents/<name>/<name>.go` implementing the `Agent` interface (`Name`, `Available`, `Analyze`)
2. Create `internal/agents/<name>/AGENT.md` with the default LLM system prompt
3. Register it in `cmd/review.go`'s `agentList`
4. Add tests in `internal/agents/<name>/<name>_test.go`
5. Add golden fixtures in `testdata/`

## Code Conventions

- Go 1.25+. Use `errors.Join`, `slog`, and generics where they simplify.
- Error handling: `fmt.Errorf("context: %w", err)` — never silently swallow.
- `context.Context` as the first parameter on any function doing I/O or spawning goroutines.
- No global mutable state — pass dependencies explicitly (constructor injection or functional options).
- Table-driven tests with `t.Run` subtests; golden files in `testdata/`.
- Cobra commands in `cmd/`. Business logic in `internal/`.
- All exported types and functions need doc comments.

## Configuration & Credentials

`.sugo.yaml` in the repo root (overridable with `--config`) controls branch patterns, conventional commit rules, required labels, lint command, and LLM provider settings. The LLM API key is **never** in config — it comes only from `SUGO_LLM_API_KEY`. The default LLM provider is **mimir**.

## External Dependencies

- `gh` CLI must be installed and authenticated (`gh auth status`)
- Linter binary matching `.sugo.yaml` lint config must be on `$PATH`
- Jira API credentials (env vars) for the analysisgap agent
