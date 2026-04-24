package llm

import (
	"fmt"

	"github.com/jacopobonta/sugo/internal/config"
)

// NewClient creates an LLM Client from config and an API key.
// Returns an error if the provider is unknown.
func NewClient(cfg config.LLMConfig, apiKey string) (Client, error) {
	switch cfg.Provider {
	case "mimir", "":
		return NewMimirClient(cfg.BaseURL, apiKey, cfg.Model), nil
	default:
		return nil, fmt.Errorf("unknown LLM provider %q", cfg.Provider)
	}
}
