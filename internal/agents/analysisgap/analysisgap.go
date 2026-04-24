// Package analysisgap implements the analysis-gap agent.
// It fetches a Jira ticket and checks whether the PR diff addresses all requirements.
package analysisgap

import (
	_ "embed"
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"

	"github.com/jacopobonta/sugo/internal/agents"
	"github.com/jacopobonta/sugo/internal/agents/llmparse"
	"github.com/jacopobonta/sugo/internal/config"
	"github.com/jacopobonta/sugo/internal/finding"
	"github.com/jacopobonta/sugo/internal/llm"
	"github.com/jacopobonta/sugo/internal/promptutil"
)

//go:embed AGENT.md
var defaultPrompt string

// jiraKeyRE extracts a Jira key (e.g. PROJ-123) from a branch name.
var jiraKeyRE = regexp.MustCompile(`[A-Z]+-\d+`)

// Agent fetches a Jira ticket and validates the PR diff covers the requirements.
type Agent struct {
	jira   JiraClient
	llm    llm.Client
	logger *slog.Logger
	prompt string
}

// New creates an analysisgap Agent. Pass nil for jira to use the real HTTP client.
func New(llmClient llm.Client, jira JiraClient, logger *slog.Logger, promptOverride string) *Agent {
	prompt, err := promptutil.Load(defaultPrompt, promptOverride)
	if err != nil && logger != nil {
		logger.Warn("analysisgap: failed to load prompt override", "error", err)
		prompt = defaultPrompt
	}
	return &Agent{jira: jira, llm: llmClient, logger: logger, prompt: prompt}
}

// NewDefault creates an analysisgap Agent using env-var credentials.
func NewDefault(llmClient llm.Client, logger *slog.Logger, cfg *config.Config, promptOverride string) *Agent {
	var jira JiraClient
	if cfg.Jira.BaseURL != "" {
		jira = newHTTPJiraClient(
			cfg.Jira.BaseURL,
			os.Getenv("SUGO_JIRA_USER"),
			os.Getenv("SUGO_JIRA_TOKEN"),
		)
	}
	return New(llmClient, jira, logger, promptOverride)
}

// Name returns the agent identifier.
func (a *Agent) Name() string { return "analysisgap" }

// Available checks that Jira credentials and LLM are configured.
func (a *Agent) Available(cfg *config.Config) bool {
	if a.llm == nil {
		return false
	}
	if cfg.Jira.BaseURL == "" {
		return false
	}
	return os.Getenv("SUGO_JIRA_TOKEN") != "" && os.Getenv("SUGO_JIRA_USER") != ""
}

// Analyze fetches the Jira ticket and asks the LLM to identify coverage gaps.
func (a *Agent) Analyze(ctx context.Context, input *agents.AnalysisInput) ([]finding.Finding, error) {
	key := jiraKeyRE.FindString(input.PR.HeadRef)
	if key == "" {
		if a.logger != nil {
			a.logger.Info("analysisgap: no Jira key found in branch name", "branch", input.PR.HeadRef)
		}
		return nil, nil
	}

	issue, err := a.jira.GetIssue(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("analysisgap fetch jira %s: %w", key, err)
	}

	userMsg := fmt.Sprintf(
		"Jira ticket %s: %s\n\nDescription:\n%s\n\nPR diff:\n%s",
		issue.Key, issue.Summary, issue.Description, input.PR.Diff,
	)

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: a.prompt},
			{Role: "user", Content: userMsg},
		},
		MaxTokens:   2048,
		Temperature: 0.2,
	})
	if err != nil {
		return nil, fmt.Errorf("analysisgap LLM call: %w", err)
	}

	return llmparse.ParseFindings(resp.Content, a.Name(), finding.TypeFix)
}
