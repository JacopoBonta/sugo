package coverage

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
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

var defaultMappings = map[string]string{
	`\.go$`:        `_test.go`,
	`\.(js|ts)$`:   `.test.$1`,
	`\.(jsx|tsx)$`: `.test.$1`,
	`\.py$`:        `_test.py`,
}

type Agent struct {
	llm    llm.Client // may be nil
	logger *slog.Logger
	prompt string
}

func New(llmClient llm.Client, logger *slog.Logger, promptOverride string) *Agent {
	prompt, err := promptutil.Load(defaultPrompt, promptOverride)
	if err != nil && logger != nil {
		logger.Warn("coverage: failed to load prompt override", "error", err)
		prompt = defaultPrompt
	}
	return &Agent{
		llm:    llmClient,
		logger: logger,
		prompt: prompt,
	}
}

func (a *Agent) Name() string { return "coverage" }

func (a *Agent) Available(_ *config.Config) bool { return true }

func (a *Agent) Analyze(ctx context.Context, input *agents.AnalysisInput) ([]finding.Finding, error) {
	var findings []finding.Finding

	// 1. Run deterministic checks
	deterministicFindings := a.runDeterministicChecks(input)
	findings = append(findings, deterministicFindings...)

	// 2. LLM Pass
	if a.llm != nil {
		llmFindings, err := a.runLLMPass(ctx, input)
		if err != nil {
			if a.logger != nil {
				a.logger.Warn("coverage LLM pass failed", "error", err)
			}
		} else {
			findings = append(findings, llmFindings...)
		}
	}

	return findings, nil
}

func (a *Agent) runDeterministicChecks(input *agents.AnalysisInput) []finding.Finding {
	var findings []finding.Finding

	// Check if any doc files are updated in the PR
	docUpdated := false
	for _, f := range input.PR.Files {
		ext := strings.ToLower(filepath.Ext(f.Path))
		if ext == ".md" || ext == ".rst" || strings.Contains(f.Path, "/docs/") || strings.Contains(f.Path, "/doc/") {
			docUpdated = true
			break
		}
	}

	// Prepare mappings
	mappings := defaultMappings
	if len(input.Config.Coverage.Mappings) > 0 {
		mappings = input.Config.Coverage.Mappings
	}

	// Helper to check if file is changed in PR
	isChangedInPR := func(path string) bool {
		for _, f := range input.PR.Files {
			if f.Path == path {
				return true
			}
		}
		return false
	}

	// Process each changed file
	for _, f := range input.PR.Files {
		if isExcluded(f.Path, input.Config.Coverage.ExcludePaths) {
			continue
		}

		// Skip test files themselves to avoid infinite loop
		if isTestFile(f.Path) {
			continue
		}

		// A. Check for test files
		var expectedTestPath string
		for pat, replacement := range mappings {
			re, err := regexp.Compile(pat)
			if err != nil {
				continue
			}
			if re.MatchString(f.Path) {
				expectedTestPath = re.ReplaceAllString(f.Path, replacement)
				break
			}
		}

		if expectedTestPath != "" && !isChangedInPR(expectedTestPath) {
			findings = append(findings, finding.Finding{
				Agent:    a.Name(),
				Severity: finding.SeverityMedium,
				Type:     finding.TypeAttentionPoint,
				Location: finding.Location{
					File: f.Path,
				},
				Message: fmt.Sprintf("Functional file modified, but corresponding test file %q was not updated in this PR.", expectedTestPath),
			})
		}

		// B. Check for API/docs update
		if f.Patch != "" {
			publicAPI, commentsAdded := inspectPatch(f.Path, f.Patch)
			if publicAPI && !docUpdated && !commentsAdded {
				findings = append(findings, finding.Finding{
					Agent:    a.Name(),
					Severity: finding.SeverityLow,
					Type:     finding.TypeAttentionPoint,
					Location: finding.Location{
						File: f.Path,
					},
					Message: "Exported/public symbols were modified, but no updates to documentation files (e.g. README.md, /docs) or inline comments were detected.",
				})
			}
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
		if f.Patch == "" || isExcluded(f.Path, input.Config.Coverage.ExcludePaths) {
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
					firstErr = fmt.Errorf("coverage LLM call for %s: %w", path, err)
				}
				mu.Unlock()
				return
			}

			findings, err := llmparse.ParseFindings(resp.Content, a.Name(), finding.TypeAttentionPoint)
			if err != nil {
				if a.logger != nil {
					a.logger.Warn("failed to parse coverage LLM findings", "file", path, "error", err)
				}
				return
			}

			// Post-process type based on whether a fix is available
			for i := range findings {
				if findings[i].Fix != nil && *findings[i].Fix != "" {
					findings[i].Type = finding.TypeFix
				} else {
					findings[i].Type = finding.TypeAttentionPoint
				}
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

func isExcluded(path string, excludes []string) bool {
	for _, pattern := range excludes {
		if strings.Contains(path, pattern) {
			return true
		}
		if matched, err := filepath.Match(pattern, path); err == nil && matched {
			return true
		}
	}
	return false
}

func isTestFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains(lower, "_test.go") ||
		strings.Contains(lower, ".test.") ||
		strings.Contains(lower, ".spec.") ||
		strings.Contains(lower, "test_")
}

// inspectPatch scans the diff for added exported symbols and added comments.
func inspectPatch(path, patch string) (publicAPI, commentsAdded bool) {
	ext := strings.ToLower(filepath.Ext(path))
	lines := strings.Split(patch, "\n")

	for _, line := range lines {
		if !strings.HasPrefix(line, "+") || strings.HasPrefix(line, "+++") {
			continue
		}
		content := line[1:]

		// Check for inline comments
		if containsComments(content, ext) {
			commentsAdded = true
		}

		// Check for exported symbols based on extension
		switch ext {
		case ".go":
			// Matches: func ExportedName, type ExportedName, const ExportedName, var ExportedName
			if reGoExport.MatchString(content) {
				publicAPI = true
			}
		case ".ts", ".js", ".tsx", ".jsx":
			// Matches: export const Name, export class Name, export function Name
			if strings.Contains(content, "export ") {
				publicAPI = true
			}
		case ".py":
			// Matches: def exported_name, class ExportedName (must not start with _)
			if rePyExport.MatchString(content) {
				publicAPI = true
			}
		}
	}
	return
}

var (
	reGoExport = regexp.MustCompile(`^\s*(func|type|const|var)\s+[A-Z]`)
	rePyExport = regexp.MustCompile(`^\s*(def|class)\s+[^_]`)
)

func containsComments(s, ext string) bool {
	s = strings.TrimSpace(s)
	switch ext {
	case ".py":
		return strings.HasPrefix(s, "#") || strings.Contains(s, `"""`) || strings.Contains(s, `'''`)
	default:
		return strings.HasPrefix(s, "//") || strings.HasPrefix(s, "/*") || strings.HasPrefix(s, "*") || strings.Contains(s, "/*")
	}
}
