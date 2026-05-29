package llmparse

import (
	"testing"
)

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Regular JSON",
			input:    `{"findings": []}`,
			expected: `{"findings": []}`,
		},
		{
			name:     "JSON wrapped in markdown",
			input:    "```json\n{\"findings\": []}\n```",
			expected: `{"findings": []}`,
		},
		{
			name:     "JSON with leading/trailing text",
			input:    `Here is the JSON: {"findings": []} hope this helps!`,
			expected: `{"findings": []}`,
		},
		{
			name:     "JSON with BOM characters",
			input:    "\xef\xbb\xbf" + `{"findings": []}`,
			expected: `{"findings": []}`,
		},
		{
			name:     "JSON with BOM characters and leading/trailing text",
			input:    "\xef\xbb\xbf" + `Here is the JSON: {"findings": []} hope this helps!`,
			expected: `{"findings": []}`,
		},
		{
			name:     "Missing braces behavior - fallback to standard markdown fence stripping",
			input:    "```json\nno_braces_here\n```",
			expected: "no_braces_here",
		},
		{
			name:     "Malformed braces first > last",
			input:    `} malformed {`,
			expected: `} malformed {`,
		},
		{
			name:     "No braces and no markdown code fences",
			input:    "plain text",
			expected: "plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := extractJSON(tt.input)
			if actual != tt.expected {
				t.Errorf("extractJSON(%q) = %q; want %q", tt.input, actual, tt.expected)
			}
		})
	}
}
