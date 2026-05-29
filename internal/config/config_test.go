package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissing(t *testing.T) {
	cfg, err := Load("nonexistent.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLM.Provider != "mimir" {
		t.Errorf("default provider = %q, want mimir", cfg.LLM.Provider)
	}
}

func TestLoadEmpty(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Fatal("nil config")
	}
}

func TestLoadMinimal(t *testing.T) {
	cfg, err := Load("../../testdata/configs/minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLM.Model != "mimir-1" {
		t.Errorf("model = %q, want mimir-1", cfg.LLM.Model)
	}
}

func TestLoadFull(t *testing.T) {
	cfg, err := Load("../../testdata/configs/full.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Rules.BranchPatterns) != 2 {
		t.Errorf("branch_patterns len = %d, want 2", len(cfg.Rules.BranchPatterns))
	}
	if cfg.Jira.ProjectKey != "PROJ" {
		t.Errorf("jira project_key = %q, want PROJ", cfg.Jira.ProjectKey)
	}
	if cfg.Agents.Rules.Prompt != "prompts/rules.md" {
		t.Errorf("agents.rules.prompt = %q", cfg.Agents.Rules.Prompt)
	}
}

func TestLoadInvalidBranchRegex(t *testing.T) {
	_, err := Load("../../testdata/configs/invalid.yaml")
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
}

func TestValidateInvalidTicketPattern(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "sugo.yaml")
	if err := os.WriteFile(cfgPath, []byte("rules:\n  trailing_ticket_pattern: \"[invalid\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid trailing_ticket_pattern regex")
	}
}

func TestValidateMissingRulesSpecFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "sugo.yaml")
	if err := os.WriteFile(cfgPath, []byte("rules:\n  spec_files:\n    - /nonexistent/spec.md\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for missing rules spec_files path")
	}
}

func TestValidateMissingLintSpecFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "sugo.yaml")
	content := "lint:\n  spec_files:\n    - path: /nonexistent/spec.md\n      extensions: [\".go\"]\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for missing lint spec_files path")
	}
}

func TestValidateSecurityAndCoverage(t *testing.T) {
	dir := t.TempDir()

	// Invalid security regex pattern
	cfgPath1 := filepath.Join(dir, "sugo1.yaml")
	content1 := "security:\n  secret_patterns:\n    - \"[invalid\"\n"
	if err := os.WriteFile(cfgPath1, []byte(content1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(cfgPath1); err == nil {
		t.Fatal("expected error for invalid security secret_pattern regex")
	}

	// Invalid coverage regex pattern
	cfgPath2 := filepath.Join(dir, "sugo2.yaml")
	content2 := "coverage:\n  mappings:\n    \"[invalid\": \"_test.go\"\n"
	if err := os.WriteFile(cfgPath2, []byte(content2), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(cfgPath2); err == nil {
		t.Fatal("expected error for invalid coverage mapping regex key")
	}

	// Valid security and coverage config
	cfgPath3 := filepath.Join(dir, "sugo3.yaml")
	content3 := `
security:
  secret_patterns:
    - "AWS_KEY"
  command: "gosec"
  args: ["-fmt=json"]
coverage:
  mappings:
    '\.go$': '_test.go'
  exclude_paths:
    - "vendor/"
`
	if err := os.WriteFile(cfgPath3, []byte(content3), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(cfgPath3)
	if err != nil {
		t.Fatalf("unexpected error loading valid security/coverage config: %v", err)
	}
	if len(cfg.Security.SecretPatterns) != 1 || cfg.Security.SecretPatterns[0] != "AWS_KEY" {
		t.Errorf("unexpected security config parsed: %+v", cfg.Security)
	}
	if cfg.Coverage.Mappings[`\.go$`] != "_test.go" {
		t.Errorf("unexpected coverage config parsed: %+v", cfg.Coverage)
	}
}

