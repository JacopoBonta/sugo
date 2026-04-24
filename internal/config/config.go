// Package config loads and validates the .sugo.yaml configuration file.
package config

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// AgentPromptConfig holds per-agent prompt override paths.
type AgentPromptConfig struct {
	Prompt string `yaml:"prompt"`
}

// AgentsConfig holds prompt override paths for each agent.
type AgentsConfig struct {
	Rules       AgentPromptConfig `yaml:"rules"`
	Lint        AgentPromptConfig `yaml:"lint"`
	Logic       AgentPromptConfig `yaml:"logic"`
	Focus       AgentPromptConfig `yaml:"focus"`
	AnalysisGap AgentPromptConfig `yaml:"analysisgap"`
}

// RulesConfig configures the rules agent.
type RulesConfig struct {
	BranchPatterns     []string `yaml:"branch_patterns"`
	ConventionalCommit bool     `yaml:"conventional_commit"`
	RequiredLabels     []string `yaml:"required_labels"`
}

// LintConfig configures the lint agent.
type LintConfig struct {
	Command           string            `yaml:"command"`
	Args              []string          `yaml:"args"`
	Paths             []string          `yaml:"paths"`
	SeverityOverrides map[string]string `yaml:"severity_overrides"`
}

// LLMConfig configures the LLM provider.
type LLMConfig struct {
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
	BaseURL  string `yaml:"base_url"`
}

// JiraConfig configures the Jira integration.
type JiraConfig struct {
	BaseURL    string `yaml:"base_url"`
	ProjectKey string `yaml:"project_key"`
}

// Config is the top-level configuration structure.
type Config struct {
	Rules  RulesConfig  `yaml:"rules"`
	Lint   LintConfig   `yaml:"lint"`
	LLM    LLMConfig    `yaml:"llm"`
	Jira   JiraConfig   `yaml:"jira"`
	Agents AgentsConfig `yaml:"agents"`
}

// Load reads and parses a .sugo.yaml config file, applying defaults and validating.
// If path is empty or the file does not exist, a default config is returned.
func Load(path string) (*Config, error) {
	cfg := defaultConfig()

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	applyDefaults(cfg)

	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		Rules: RulesConfig{
			ConventionalCommit: true,
		},
		LLM: LLMConfig{
			Provider: "mimir",
		},
	}
}

func applyDefaults(cfg *Config) {
	if cfg.LLM.Provider == "" {
		cfg.LLM.Provider = "mimir"
	}
}

func validate(cfg *Config) error {
	for _, pattern := range cfg.Rules.BranchPatterns {
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("branch pattern %q: %w", pattern, err)
		}
	}
	return nil
}
