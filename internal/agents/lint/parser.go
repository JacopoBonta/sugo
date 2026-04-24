package lint

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/jacopobonta/sugo/internal/finding"
)

// lintIssue is a normalized linter finding before conversion to finding.Finding.
type lintIssue struct {
	File    string
	Line    int
	Message string
	Rule    string
}

// Parser converts raw linter output to a slice of lintIssues.
type Parser interface {
	Parse(output []byte) ([]lintIssue, error)
}

// golangciParser parses golangci-lint JSON output.
type golangciParser struct{}

func (golangciParser) Parse(output []byte) ([]lintIssue, error) {
	var raw struct {
		Issues []struct {
			FromLinter string `json:"FromLinter"`
			Text       string `json:"Text"`
			Pos        struct {
				Filename string `json:"Filename"`
				Line     int    `json:"Line"`
			} `json:"Pos"`
		} `json:"Issues"`
	}
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("golangci parse: %w", err)
	}
	issues := make([]lintIssue, 0, len(raw.Issues))
	for _, i := range raw.Issues {
		issues = append(issues, lintIssue{
			File:    i.Pos.Filename,
			Line:    i.Pos.Line,
			Message: i.Text,
			Rule:    i.FromLinter,
		})
	}
	return issues, nil
}

// eslintParser parses ESLint JSON output.
type eslintParser struct{}

func (eslintParser) Parse(output []byte) ([]lintIssue, error) {
	var raw []struct {
		FilePath string `json:"filePath"`
		Messages []struct {
			RuleID  string `json:"ruleId"`
			Message string `json:"message"`
			Line    int    `json:"line"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("eslint parse: %w", err)
	}
	var issues []lintIssue
	for _, f := range raw {
		for _, m := range f.Messages {
			issues = append(issues, lintIssue{
				File:    f.FilePath,
				Line:    m.Line,
				Message: m.Message,
				Rule:    m.RuleID,
			})
		}
	}
	return issues, nil
}

// genericParser parses "file:line:col: message" text output (ruff, pyflakes, etc.).
type genericParser struct{}

var genericLineRE = regexp.MustCompile(`^([^:]+):(\d+)(?::\d+)?: (.+)$`)

func (genericParser) Parse(output []byte) ([]lintIssue, error) {
	var issues []lintIssue
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := genericLineRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		lineNum, _ := strconv.Atoi(m[2])
		issues = append(issues, lintIssue{
			File:    m[1],
			Line:    lineNum,
			Message: m[3],
		})
	}
	return issues, nil
}

// selectParser picks the right parser based on the linter command name.
func selectParser(command string) Parser {
	cmd := strings.ToLower(command)
	switch {
	case strings.Contains(cmd, "golangci"):
		return golangciParser{}
	case strings.Contains(cmd, "eslint"):
		return eslintParser{}
	default:
		return genericParser{}
	}
}

// toFindings converts lintIssues to findings, applying severity overrides from config.
func toFindings(issues []lintIssue, agentName string, severityOverrides map[string]string) []finding.Finding {
	results := make([]finding.Finding, 0, len(issues))
	for _, issue := range issues {
		sev := finding.SeverityMedium
		if override, ok := severityOverrides[issue.Rule]; ok {
			switch strings.ToLower(override) {
			case "high":
				sev = finding.SeverityHigh
			case "low":
				sev = finding.SeverityLow
			}
		}
		msg := issue.Message
		if issue.Rule != "" {
			msg = fmt.Sprintf("[%s] %s", issue.Rule, issue.Message)
		}
		results = append(results, finding.Finding{
			Agent:    agentName,
			Severity: sev,
			Type:     finding.TypeFix,
			Location: finding.Location{File: issue.File, LineStart: issue.Line, LineEnd: issue.Line},
			Message:  msg,
		})
	}
	return results
}
