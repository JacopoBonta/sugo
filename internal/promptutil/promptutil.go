// Package promptutil provides helpers for loading agent system prompts.
package promptutil

import (
	"fmt"
	"os"
)

// Load returns the system prompt for an agent.
// If overridePath is non-empty, it reads from that file.
// Otherwise it returns the embedded default.
func Load(embedded string, overridePath string) (string, error) {
	if overridePath == "" {
		return embedded, nil
	}
	data, err := os.ReadFile(overridePath)
	if err != nil {
		return "", fmt.Errorf("read prompt override %s: %w", overridePath, err)
	}
	return string(data), nil
}
