// Package promptutil provides helpers for loading agent system prompts.
package promptutil

import (
	"fmt"
	"os"
	"strings"
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

// Compose builds a system prompt by appending non-empty extras to base,
// each separated by a horizontal rule divider.
func Compose(base string, extras ...string) string {
	parts := []string{base}
	for _, e := range extras {
		if e != "" {
			parts = append(parts, e)
		}
	}
	return strings.Join(parts, "\n\n---\n\n")
}
