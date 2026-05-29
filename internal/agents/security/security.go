package security

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/jacopobonta/sugo/internal/agents"
	"github.com/jacopobonta/sugo/internal/agents/llmparse"
	"github.com/jacopobonta/sugo/internal/config"
	"github.com/jacopobonta/sugo/internal/finding"
	"github.com/jacopobonta/sugo/internal/llm"
	"github.com/jacopobonta/sugo/internal/promptutil"
)

//go:embed AGENT.md
var defaultPrompt string

var defaultSecretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(?:aws_access_key_id|aws_secret_access_key|aws_key|aws_token|secret_key|api_key|apikey|secret|password|passwd|private_key|privatekey|token|auth)\s*[:=]\s*['"]([a-zA-Z0-9_\-\.\~\+\/]{16,})['"]`),
	regexp.MustCompile(`xox[bapr]-[0-9]{12}-[a-zA-Z0-9]{24}`),
	regexp.MustCompile(`gh[opr]_[a-zA-Z0-9]{36}`),
	regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`),
}

type runFn func(ctx context.Context, command string, args []string) ([]byte, error)

type Agent struct {
	run    runFn
	llm    llm.Client // may be nil
	logger *slog.Logger
	prompt string
}

func New(llmClient llm.Client, logger *slog.Logger, promptOverride string) *Agent {
	prompt, err := promptutil.Load(defaultPrompt, promptOverride)
	if err != nil && logger != nil {
		logger.Warn("security: failed to load prompt override", "error", err)
		prompt = defaultPrompt
	}
	return &Agent{
		run:    securityRun,
		llm:    llmClient,
		logger: logger,
		prompt: prompt,
	}
}

func NewWithRun(fn runFn, llmClient llm.Client, logger *slog.Logger) *Agent {
	return &Agent{
		run:    fn,
		llm:    llmClient,
		logger: logger,
		prompt: defaultPrompt,
	}
}

func securityRun(ctx context.Context, command string, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		// External tool may return non-zero when vulnerabilities are found
		_ = err.Error()
	}
	return out.Bytes(), nil
}

func (a *Agent) Name() string { return "security" }

func (a *Agent) Available(_ *config.Config) bool { return true }

func (a *Agent) Analyze(ctx context.Context, input *agents.AnalysisInput) ([]finding.Finding, error) {
	var findings []finding.Finding

	// 1. Run local CLI security tool if configured
	if input.Config.Security.Command != "" {
		cliFindings, err := a.runCLIScanner(ctx, input.Config.Security.Command, input.Config.Security.Args)
		if err != nil {
			if a.logger != nil {
				a.logger.Warn("security CLI scanner run failed", "error", err)
			}
		} else {
			findings = append(findings, cliFindings...)
		}
	}

	// 2. Local regex secret scan on patch diffs
	regexFindings := a.runRegexSecretScan(input)
	findings = append(findings, regexFindings...)

	// 3. LLM Pass
	if a.llm != nil {
		llmFindings, err := a.runLLMPass(ctx, input)
		if err != nil {
			if a.logger != nil {
				a.logger.Warn("security LLM pass failed", "error", err)
			}
		} else {
			findings = append(findings, llmFindings...)
		}
	}

	return findings, nil
}

func (a *Agent) runCLIScanner(ctx context.Context, command string, args []string) ([]finding.Finding, error) {
	output, err := a.run(ctx, command, args)
	if err != nil {
		return nil, err
	}

	cmdLower := strings.ToLower(command)
	if strings.Contains(cmdLower, "gosec") {
		return parseGosec(output)
	} else if strings.Contains(cmdLower, "semgrep") {
		return parseSemgrep(output)
	}

	return parseGenericSecurity(output)
}

func (a *Agent) runRegexSecretScan(input *agents.AnalysisInput) []finding.Finding {
	var findings []finding.Finding
	patterns := defaultSecretPatterns

	if len(input.Config.Security.SecretPatterns) > 0 {
		patterns = make([]*regexp.Regexp, 0, len(input.Config.Security.SecretPatterns))
		for _, pat := range input.Config.Security.SecretPatterns {
			re, err := regexp.Compile(pat)
			if err == nil {
				patterns = append(patterns, re)
			}
		}
	}

	for _, file := range input.PR.Files {
		if file.Patch == "" {
			continue
		}
		secretOccurrences := scanPatchForSecrets(file.Patch, patterns)
		for _, o := range secretOccurrences {
			findings = append(findings, finding.Finding{
				Agent:    a.Name(),
				Severity: finding.SeverityHigh,
				Type:     finding.TypeFix,
				Location: finding.Location{
					File:      file.Path,
					LineStart: o.Line,
					LineEnd:   o.Line,
				},
				Message: o.Message,
			})
		}
	}

	return findings
}

func (a *Agent) runLLMPass(ctx context.Context, input *agents.AnalysisInput) ([]finding.Finding, error) {
	var (
		mu       sync.Mutex
		all      []finding.Finding
		firstErr error
	)

	const maxConcurrent = 5
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, f := range input.PR.Files {
		if f.Patch == "" {
			continue
		}

		wg.Add(1)
		go func(path, patch string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			userMsg := fmt.Sprintf("File: %s\n\n%s", path, patch)
			resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
				Messages: []llm.Message{
					{Role: "system", Content: a.prompt},
					{Role: "user", Content: userMsg},
				},
				MaxTokens:   2048,
				Temperature: llm.Float64(0.0),
				JSONMode:    true,
			})
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("security LLM call for %s: %w", path, err)
				}
				mu.Unlock()
				return
			}

			findings, err := llmparse.ParseFindings(resp.Content, a.Name(), finding.TypeFix)
			if err != nil {
				if a.logger != nil {
					a.logger.Warn("failed to parse security LLM findings", "file", path, "error", err)
				}
				return
			}

			mu.Lock()
			all = append(all, findings...)
			mu.Unlock()
		}(f.Path, f.Patch)
	}
	wg.Wait()

	if firstErr != nil && len(all) == 0 {
		return nil, firstErr
	}

	return all, nil
}

type secretFinding struct {
	Line    int
	Message string
}

func scanPatchForSecrets(patch string, patterns []*regexp.Regexp) []secretFinding {
	var findings []secretFinding
	lines := strings.Split(patch, "\n")
	var currentLine int
	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			parts := strings.Split(line, " ")
			if len(parts) >= 3 {
				newRange := parts[2]
				newRange = strings.TrimPrefix(newRange, "+")
				subparts := strings.Split(newRange, ",")
				start, err := strconv.Atoi(subparts[0])
				if err == nil {
					currentLine = start - 1
				}
			}
			continue
		}
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			currentLine++
			content := line[1:]
			for _, re := range patterns {
				if re.MatchString(content) {
					findings = append(findings, secretFinding{
						Line:    currentLine,
						Message: fmt.Sprintf("Potential hardcoded secret or API key detected by regex: %s", re.String()),
					})
					break
				}
			}
		} else if !strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			currentLine++
		}
	}
	return findings
}

// Parsers for CLI scanner outputs

func parseGosec(output []byte) ([]finding.Finding, error) {
	var raw struct {
		Issues []struct {
			Severity   string      `json:"severity"`
			Confidence string      `json:"confidence"`
			RuleID     string      `json:"rule_id"`
			Details    string      `json:"details"`
			File       string      `json:"file"`
			Line       json.Number `json:"line"`
		} `json:"Issues"`
	}
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("gosec parse: %w", err)
	}

	findings := make([]finding.Finding, 0, len(raw.Issues))
	for _, i := range raw.Issues {
		lineNum, err := i.Line.Int64()
		if err != nil {
			lineNum = 0
		}
		severity := finding.SeverityMedium
		switch strings.ToUpper(i.Severity) {
		case "HIGH":
			severity = finding.SeverityHigh
		case "LOW":
			severity = finding.SeverityLow
		}

		findings = append(findings, finding.Finding{
			Agent:    "security",
			Severity: severity,
			Type:     finding.TypeFix,
			Location: finding.Location{
				File:      i.File,
				LineStart: int(lineNum),
				LineEnd:   int(lineNum),
			},
			Message: fmt.Sprintf("[%s] %s (Confidence: %s)", i.RuleID, i.Details, i.Confidence),
		})
	}
	return findings, nil
}

func parseSemgrep(output []byte) ([]finding.Finding, error) {
	var raw struct {
		Results []struct {
			Path  string `json:"path"`
			Start struct {
				Line int `json:"line"`
			} `json:"start"`
			Extra struct {
				Message  string `json:"message"`
				Severity string `json:"severity"`
				Metadata struct {
					RuleID string `json:"rule_id"`
				} `json:"metadata"`
			} `json:"extra"`
		} `json:"results"`
	}
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("semgrep parse: %w", err)
	}

	findings := make([]finding.Finding, 0, len(raw.Results))
	for _, r := range raw.Results {
		severity := finding.SeverityMedium
		switch strings.ToUpper(r.Extra.Severity) {
		case "ERROR":
			severity = finding.SeverityHigh
		case "INFO":
			severity = finding.SeverityLow
		}

		ruleID := r.Extra.Metadata.RuleID
		if ruleID == "" {
			ruleID = "semgrep-rule"
		}

		findings = append(findings, finding.Finding{
			Agent:    "security",
			Severity: severity,
			Type:     finding.TypeFix,
			Location: finding.Location{
				File:      r.Path,
				LineStart: r.Start.Line,
				LineEnd:   r.Start.Line,
			},
			Message: fmt.Sprintf("[%s] %s", ruleID, r.Extra.Message),
		})
	}
	return findings, nil
}

var genericSecurityRE = regexp.MustCompile(`(?i)(?:^|\[)([^:\[\]\n\r]+):(\d+)(?::\d+)?\]?\s*-?\s*(.*)$`)

func parseGenericSecurity(output []byte) ([]finding.Finding, error) {
	var findings []finding.Finding
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := genericSecurityRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		lineNum, err := strconv.Atoi(m[2])
		if err != nil {
			lineNum = 0
		}
		findings = append(findings, finding.Finding{
			Agent:    "security",
			Severity: finding.SeverityMedium,
			Type:     finding.TypeFix,
			Location: finding.Location{
				File:      m[1],
				LineStart: lineNum,
				LineEnd:   lineNum,
			},
			Message: m[3],
		})
	}
	return findings, nil
}
