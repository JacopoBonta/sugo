package llm

import (
	"fmt"

	"github.com/jacopobonta/sugo/internal/config"
)

// NewClient creates an LLM Client from config and an API key.
// Returns an error if the provider is unknown.
func NewClient(cfg config.LLMConfig, apiKey string) (Client, error) {
	var client Client
	switch cfg.Provider {
	case "mimir", "":
		client = NewMimirClient(cfg.BaseURL, apiKey, cfg.Model, cfg.Temperature, cfg.Seed, cfg.JSONMode)
	default:
		return nil, fmt.Errorf("unknown LLM provider %q", cfg.Provider)
	}

	if cfg.Cache != nil && *cfg.Cache {
		cacheDir := cfg.CacheDir
		if cacheDir == "" {
			cacheDir = ".sugo/cache"
		}
		cachingClient, err := NewCachingClient(client, cacheDir)
		if err != nil {
			return nil, err
		}
		return cachingClient, nil
	}

	return client, nil
}
