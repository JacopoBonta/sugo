# SUGO User Guide & Customization Manual

SUGO (SuggestGreatOutput) is a parallel-agent, LLM-enriched code review assistant CLI for developers and code reviewers. It runs security checks, rule validations, lint commands, and logical analysis across code changes, aggregating findings to give reviewers a head start before reading pull requests manually.

This guide details the purpose, inner workings, and configuration/customization options for every analysis agent in SUGO.

---

## Table of Contents
1. [Core Configuration](#core-configuration)
2. [Rules Agent](#1-rules-agent)
3. [Lint Agent](#2-lint-agent)
4. [Security Agent](#3-security-agent)
5. [Coverage Agent](#4-coverage-agent)
6. [Logic Agent](#5-logic-agent)
7. [Focus Agent](#6-focus-agent)
8. [Analysis-Gap Agent](#7-analysis-gap-agent)

---

## Core Configuration

SUGO is configured via a YAML file (defaults to looking for `.sugo.yaml` in the repository root, which can be overridden using the `--config` CLI flag).

### LLM Setup
Most agents leverage a LLM client to parse findings, suggest fixes, or review logic. Configure the LLM provider in `.sugo.yaml`:

```yaml
llm:
  provider: mimir                 # LLM provider (default: mimir)
  model: vllm/mimir               # Target model
  base_url: "https://mimir.dev"   # Provider API URL
```

> [!IMPORTANT]
> The API key is loaded from the environment variable `SUGO_LLM_API_KEY` for security and is never stored in the configuration file.

---

## 1. Rules Agent

Checks pull request metadata to ensure consistency with branch naming conventions, conventional commit messages, and repository-specific labeling rules.

### How it Works
1. **Deterministic Scan**: Checks the branch name matching `branch_patterns`. If `conventional_commit` is enabled, checks all PR commits. If `trailing_ticket_pattern` is defined, ensures branch and commit messages contain issue tracker tickets at the end.
2. **LLM Enrichment**: If violations occur, the agent feeds them to the LLM alongside loaded standard specification markdown files to explain the violation clearly and suggest corrective steps.

### Customization Options

Set rules configuration under `rules:` and the optional prompt override under `agents.rules.prompt`:

```yaml
rules:
  branch_patterns:
    - "^(feat|fix|hotfix|chore)/[a-z0-9-]+-[a-z]+-[0-9]+$"
    - "^release/v[0-9]+\\.[0-9]+\\.[0-9]+$"
  conventional_commit: true
  trailing_ticket_pattern: "[a-z]+-[0-9]+$"
  required_labels:
    - "needs-review"
  spec_files:
    - specs/rules.composer.md

agents:
  rules:
    prompt: "prompts/custom_rules.md" # Path to custom rules system prompt
```

- **`branch_patterns`**: List of regular expressions matching allowed branch name patterns.
- **`conventional_commit`**: Set to `true` to require conventional commits (e.g. `feat: description`).
- **`trailing_ticket_pattern`**: Regex to match and require ticket suffixes on branch and commit messages.
- **`required_labels`**: String list of labels that must be present on the pull request.
- **`spec_files`**: Paths to project specification markdown rules to append to the system prompt.
- **`prompt`**: Path to a markdown file containing a custom system prompt to override the default rules agent prompt.

---

## 2. Lint Agent

Integrates existing development linters (e.g. `golangci-lint`, `eslint`, `ruff`) directly into the review report and explains warnings contextually.

### How it Works
1. Runs the specified shell `command` with `args`.
2. Automatically parses stdout based on the linter type (currently handles `golangci-lint` JSON formats, standard checkstyles, or generic output matchers).
3. If an LLM is configured, it sends parsed lint issues to the LLM to write exact code patches/fixes based on selected style guides matching file extensions.

### Customization Options

Set lint parameters under `lint:` and prompt overrides under `agents.lint.prompt`:

```yaml
lint:
  command: golangci-lint
  args: [ "run", "--output.json.path", "stdout", "--show-stats=false", "--timeout", "5m" ]
  paths: []
  severity_overrides:
    godot: low
    gochecknoglobals: low
  spec_files:
    - path: specs/codestyle.go.md
      extensions: [ ".go" ]
    - path: specs/codestyle.ts.md
      extensions: [ ".ts", ".tsx" ]

agents:
  lint:
    prompt: "prompts/custom_lint.md"
```

- **`command`**: The linter command to execute (must be available on the shell `$PATH`).
- **`args`**: Command-line arguments passed to the linter.
- **`severity_overrides`**: A key-value mapping of specific linter checks to custom report severities (`high`, `medium`, `low`).
- **`spec_files`**: Defines which coding standard markdown files are loaded and sent to the LLM depending on the extensions of the changed files in the PR.

---

## 3. Security Agent

Identifies vulnerabilities, security policy violations, and hardcoded secrets within code changes.

### How it Works
1. **CLI Scanner**: Runs a configured CLI security scan tool (e.g., `gosec`, `semgrep`, etc.) on the project and parses vulnerability results.
2. **Secret Scan**: Uses high-confidence regexes to look for hardcoded API keys, private keys, Slack API tokens, and credentials in the PR diff.
3. **LLM Scan**: Passes changed file diffs to the LLM with instructions to inspect them for access control flaws, input validation issues (SQLi, XSS), cryptographic mistakes, and unsafe library calls.

### Customization Options

Set security options under `security:` and the prompt override under `agents.security.prompt`:

```yaml
security:
  command: gosec
  args: [ "-fmt=json", "./..." ]
  secret_patterns:
    - '(?i)(?:aws_access_key_id|aws_secret_access_key|api_key|secret_key)\s*[:=]\s*[''"]([a-zA-Z0-9_\-\.\~\+\/]{16,})[''"]'
    - 'xox[bapr]-[0-9]{12}-[a-zA-Z0-9]{24}'

agents:
  security:
    prompt: "prompts/custom_security.md"
```

- **`command`**: The external security command to execute (e.g., `gosec`, `semgrep`).
- **`args`**: CLI arguments for the tool.
- **`secret_patterns`**: Custom regex pattern overrides for hardcoded credential matching.
- **`prompt`**: Path to a markdown file to override the default security system prompt.

---

## 4. Coverage Agent

Maintains test coverage discipline and checks that functional logic modifications are matched with appropriate testing and documentation updates.

### How it Works
1. **Deterministic Test File Match**: Matches modified files against a mapping regex (e.g., matching a functional `.go` file with its expected `_test.go` counterpart). If the test counterpart is missing from the PR, it flags a warning.
2. **Deterministic Doc/API Match**: Checks for exported/public symbols (such as exported Go types, TypeScript `export` keywords, Python public `class` or `def` definitions) in the diff. If new symbols are added but no inline comments or markdown files (in `/docs` or `README.md`) are changed, it logs a documentation coverage warning.
3. **LLM Test Planner**: Passes functional code patches to the LLM to inspect logic completeness and draft suggested unit/integration test specifications.

### Customization Options

Configure coverage criteria under `coverage:` and the prompt override under `agents.coverage.prompt`:

```yaml
coverage:
  mappings:
    `\.go$`: `_test.go`
    `\.(js|ts)$`: `.test.$1`
    `\.(jsx|tsx)$`: `.test.$1`
    `\.py$`: `_test.py`
  exclude_paths:
    - "**/vendor/**"
    - "**/mocks/**"
    - "testdata/**"

agents:
  coverage:
    prompt: "prompts/custom_coverage.md"
```

- **`mappings`**: Key-value mapping of functional file pattern regexes to their expected test file name formats.
- **`exclude_paths`**: Globs or subpaths to exclude functional files from test coverage enforcement.
- **`prompt`**: Path to a markdown file to override the default coverage agent prompt.

---

## 5. Logic Agent

Inspects changed code for logical bugs, edge cases, error-handling bugs, and concurrent programming flaws.

### How it Works
- Sends the patch diff of each changed file concurrently to the LLM.
- Analyzes logic flow, checks for off-by-one errors, bounds checks, dereference risks, unhandled errors, and race conditions.

### Customization Options

Customization is done primarily through system prompt tuning under `agents.logic.prompt`:

```yaml
agents:
  logic:
    prompt: "prompts/custom_logic.md" # Path to a custom logic inspection prompt
```

- **`prompt`**: Custom system prompt defining specific anti-patterns, performance standards, or architectural guidelines the logic agent should inspect.

---

## 6. Focus Agent

Synthesizes a high-level review summary and recommends prioritizing review time on high-complexity or high-risk files.

### How it Works
- Feeds the full aggregated pull request diff along with the title to the LLM.
- Explains the architectural impact of the change and ranks files by review priority.

### Customization Options

Customization is done through system prompt tuning under `agents.focus.prompt`:

```yaml
agents:
  focus:
    prompt: "prompts/custom_focus.md"
```

- **`prompt`**: Override prompt to direct the focus agent on specific prioritization priorities (e.g., focusing on database migrations, API contracts, or critical business rule engines).

---

## 7. Analysis-Gap Agent

Validates that code changes in a pull request align and cover the requirements outlined in their corresponding Jira issue ticket.

### How it Works
1. Extracts a Jira issue key (e.g. `PROJ-123`) from the branch name using regular expressions.
2. Queries the Jira API using the instance configured under `jira.base_url`.
3. Sends both the retrieved ticket details and the PR diff to the LLM to inspect whether any requirements are missing, incomplete, or incorrectly implemented.

### Setup and Customization

Configure Jira parameters under `jira:` and the prompt override under `agents.analysisgap.prompt`:

```yaml
jira:
  base_url: "https://your-company.atlassian.net"
  project_key: "PROJ"

agents:
  analysisgap:
    prompt: "prompts/custom_gap.md"
```

In addition to `.sugo.yaml` settings, you must export credentials to your shell environment:
- **`SUGO_JIRA_USER`**: Your Jira username or email address.
- **`SUGO_JIRA_TOKEN`**: A Jira API token generated from Atlassian console.
