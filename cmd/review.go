package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/jacopobonta/sugo/internal/agents"
	"github.com/jacopobonta/sugo/internal/agents/analysisgap"
	"github.com/jacopobonta/sugo/internal/agents/focus"
	"github.com/jacopobonta/sugo/internal/agents/lint"
	"github.com/jacopobonta/sugo/internal/agents/logic"
	"github.com/jacopobonta/sugo/internal/agents/rules"
	"github.com/jacopobonta/sugo/internal/config"
	"github.com/jacopobonta/sugo/internal/gh"
	"github.com/jacopobonta/sugo/internal/llm"
	"github.com/jacopobonta/sugo/internal/orchestrator"
	"github.com/jacopobonta/sugo/internal/renderer"
)

var (
	outputFormat      string
	rulesAgentPrompt  string
	lintAgentPrompt   string
	logicAgentPrompt  string
	focusAgentPrompt  string
	agapAgentPrompt   string
)

var reviewCmd = &cobra.Command{
	Use:   "review owner/repo#number",
	Short: "Analyze a GitHub PR and produce a review report",
	Args:  cobra.ExactArgs(1),
	RunE:  runReview,
}

func runReview(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	owner, repo, number, err := parseReviewArg(args[0])
	if err != nil {
		return fmt.Errorf("invalid argument %q: %w", args[0], err)
	}

	verbose, _ := cmd.Flags().GetBool("verbose")
	logger := buildLogger(verbose)

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Build LLM client if API key is set.
	var llmClient llm.Client
	if apiKey := os.Getenv("SUGO_LLM_API_KEY"); apiKey != "" {
		llmClient, err = llm.NewClient(cfg.LLM, apiKey)
		if err != nil {
			return fmt.Errorf("create LLM client: %w", err)
		}
	} else {
		logger.Warn("SUGO_LLM_API_KEY not set — LLM-dependent agents will be skipped")
	}

	// Resolve per-agent prompt overrides: CLI flag > config > embedded default.
	promptFor := func(flag, cfgPath string) string {
		if flag != "" {
			return flag
		}
		return cfgPath
	}

	agentList := []agents.Agent{
		rules.New(llmClient, logger, promptFor(rulesAgentPrompt, cfg.Agents.Rules.Prompt), cfg.Rules),
		lint.New(llmClient, logger, promptFor(lintAgentPrompt, cfg.Agents.Lint.Prompt), cfg.Lint),
		logic.New(llmClient, logger, promptFor(logicAgentPrompt, cfg.Agents.Logic.Prompt)),
		focus.New(llmClient, logger, promptFor(focusAgentPrompt, cfg.Agents.Focus.Prompt)),
		analysisgap.NewDefault(llmClient, logger, cfg, promptFor(agapAgentPrompt, cfg.Agents.AnalysisGap.Prompt)),
	}

	ghClient := gh.NewCLIClient()
	orch := orchestrator.New(ghClient, agentList, cfg, logger)

	report, err := orch.Run(ctx, owner, repo, number)
	if err != nil {
		return err
	}

	return renderer.Render(os.Stdout, report, outputFormat)
}

func buildLogger(verbose bool) *slog.Logger {
	level := slog.LevelWarn
	if verbose {
		level = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}

var prRefPattern = regexp.MustCompile(`^([^/]+)/([^#]+)#(\d+)$`)

func parseReviewArg(arg string) (owner, repo string, number int, err error) {
	m := prRefPattern.FindStringSubmatch(arg)
	if m == nil {
		return "", "", 0, fmt.Errorf("expected format owner/repo#number")
	}
	n, _ := strconv.Atoi(m[3])
	return m[1], m[2], n, nil
}

func init() {
	rootCmd.AddCommand(reviewCmd)
	reviewCmd.Flags().StringVar(&outputFormat, "format", "terminal", "output format: terminal, json, or markdown")
	reviewCmd.Flags().StringVar(&rulesAgentPrompt, "rules-agent", "", "path to custom rules agent system prompt")
	reviewCmd.Flags().StringVar(&lintAgentPrompt, "lint-agent", "", "path to custom lint agent system prompt")
	reviewCmd.Flags().StringVar(&logicAgentPrompt, "logic-agent", "", "path to custom logic agent system prompt")
	reviewCmd.Flags().StringVar(&focusAgentPrompt, "focus-agent", "", "path to custom focus agent system prompt")
	reviewCmd.Flags().StringVar(&agapAgentPrompt, "analysisgap-agent", "", "path to custom analysisgap agent system prompt")
	rootCmd.PersistentFlags().Bool("verbose", false, "enable verbose logging")
}
