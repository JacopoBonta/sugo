// Package specs provides helpers for loading spec files and extracting
// file-extension information from unified diffs.
package specs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Load reads a spec file and returns its trimmed content.
func Load(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("load spec %s: %w", path, err)
	}
	return strings.TrimSpace(string(data)), nil
}

// LoadAll loads multiple spec files and returns their trimmed contents in order.
func LoadAll(paths []string) ([]string, error) {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		content, err := Load(p)
		if err != nil {
			return nil, err
		}
		out = append(out, content)
	}
	return out, nil
}

// ExtensionsFromDiff parses a unified diff and returns the set of file
// extensions present among the changed files.
func ExtensionsFromDiff(diff string) map[string]struct{} {
	exts := make(map[string]struct{})
	for line := range strings.SplitSeq(diff, "\n") {
		if !strings.HasPrefix(line, "diff --git ") {
			continue
		}
		// format: "diff --git a/<path> b/<path>"
		parts := strings.Fields(line)
		if len(parts) < 4 {
			continue
		}
		bPath := strings.TrimPrefix(parts[3], "b/")
		if ext := filepath.Ext(bPath); ext != "" {
			exts[ext] = struct{}{}
		}
	}
	return exts
}
