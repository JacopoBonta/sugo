package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"sync"

	"github.com/spf13/cobra"

	"github.com/jacopobonta/sugo/internal/agents"
	"github.com/jacopobonta/sugo/internal/agents/analysisgap"
	"github.com/jacopobonta/sugo/internal/agents/coverage"
	"github.com/jacopobonta/sugo/internal/agents/focus"
	"github.com/jacopobonta/sugo/internal/agents/lint"
	"github.com/jacopobonta/sugo/internal/agents/logic"
	"github.com/jacopobonta/sugo/internal/agents/rules"
	"github.com/jacopobonta/sugo/internal/agents/security"
	"github.com/jacopobonta/sugo/internal/config"
	"github.com/jacopobonta/sugo/internal/gh"
	"github.com/jacopobonta/sugo/internal/llm"
	"github.com/jacopobonta/sugo/internal/orchestrator"
	"github.com/jacopobonta/sugo/internal/renderer"
)

var (
	outputFormat         string
	rulesAgentPrompt     string
	lintAgentPrompt      string
	logicAgentPrompt     string
	focusAgentPrompt     string
	agapAgentPrompt      string
	securityAgentPrompt  string
	coverageAgentPrompt  string
	openHTML             bool
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

	verbose, err := cmd.Flags().GetBool("verbose")
	if err != nil {
		return fmt.Errorf("get verbose flag: %w", err)
	}
	logger := buildLogger(verbose)

	// Wrap logger if using spinner to buffer console logs
	useSpinner := isStderrTTY() && !verbose
	var bufHandler *bufferingHandler
	if useSpinner {
		bufHandler = &bufferingHandler{parent: logger.Handler()}
		logger = slog.New(bufHandler)
	}

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
		security.New(llmClient, logger, promptFor(securityAgentPrompt, cfg.Agents.Security.Prompt)),
		coverage.New(llmClient, logger, promptFor(coverageAgentPrompt, cfg.Agents.Coverage.Prompt)),
	}

	ghClient := gh.NewCLIClient()
	orch := orchestrator.New(ghClient, agentList, cfg, logger)

	report, err := orch.Run(ctx, owner, repo, number)
	if bufHandler != nil {
		bufHandler.Flush(ctx)
	}
	if err != nil {
		return err
	}

	if openHTML {
		tmpFile, err := os.CreateTemp("", "sugo_report_*.html")
		if err != nil {
			return fmt.Errorf("create temp file for HTML report: %w", err)
		}
		defer tmpFile.Close()

		if err := renderer.Render(tmpFile, report, "html"); err != nil {
			return fmt.Errorf("render HTML report: %w", err)
		}

		fmt.Printf("HTML report written to: %s\n", tmpFile.Name())
		if err := openBrowser(tmpFile.Name()); err != nil {
			return fmt.Errorf("open HTML report in browser: %w", err)
		}
		return nil
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
	n, err := strconv.Atoi(m[3])
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid PR number: %w", err)
	}
	return m[1], m[2], n, nil
}

func init() {
	rootCmd.AddCommand(reviewCmd)
	reviewCmd.Flags().StringVar(&outputFormat, "format", "terminal", "output format: terminal, json, markdown, or html")
	reviewCmd.Flags().StringVar(&rulesAgentPrompt, "rules-agent", "", "path to custom rules agent system prompt")
	reviewCmd.Flags().StringVar(&lintAgentPrompt, "lint-agent", "", "path to custom lint agent system prompt")
	reviewCmd.Flags().StringVar(&logicAgentPrompt, "logic-agent", "", "path to custom logic agent system prompt")
	reviewCmd.Flags().StringVar(&focusAgentPrompt, "focus-agent", "", "path to custom focus agent system prompt")
	reviewCmd.Flags().StringVar(&agapAgentPrompt, "analysisgap-agent", "", "path to custom analysisgap agent system prompt")
	reviewCmd.Flags().StringVar(&securityAgentPrompt, "security-agent", "", "path to custom security agent system prompt")
	reviewCmd.Flags().StringVar(&coverageAgentPrompt, "coverage-agent", "", "path to custom coverage agent system prompt")
	reviewCmd.Flags().BoolVar(&openHTML, "open", false, "automatically open HTML report in browser")
	rootCmd.PersistentFlags().Bool("verbose", false, "enable verbose logging")
}

func openBrowser(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	return exec.Command(cmd, args...).Start()
}

func isStderrTTY() bool {
	fi, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

type bufferingHandler struct {
	mu      sync.Mutex
	records []slog.Record
	parent  slog.Handler
}

func (h *bufferingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.parent.Enabled(ctx, level)
}

func (h *bufferingHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	h.records = append(h.records, r.Clone())
	h.mu.Unlock()
	return nil
}

func (h *bufferingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &bufferingHandler{
		parent:  h.parent.WithAttrs(attrs),
		records: h.records,
	}
}

func (h *bufferingHandler) WithGroup(name string) slog.Handler {
	return &bufferingHandler{
		parent:  h.parent.WithGroup(name),
		records: h.records,
	}
}

func (h *bufferingHandler) Flush(ctx context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, r := range h.records {
		err := h.parent.Handle(ctx, r)
		_ = err
	}
	h.records = nil
}
