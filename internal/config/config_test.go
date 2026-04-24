package config

import (
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
