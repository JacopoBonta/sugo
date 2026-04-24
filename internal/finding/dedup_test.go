package finding

import (
	"testing"
)

func ptr(s string) *string { return &s }

func TestDeduplicate(t *testing.T) {
	loc := Location{File: "a.go", LineStart: 1, LineEnd: 5}

	tests := []struct {
		name     string
		input    []Finding
		wantLen  int
		wantSev  Severity // for single-finding dedup cases
	}{
		{
			name:    "empty",
			input:   nil,
			wantLen: 0,
		},
		{
			name: "no duplicates",
			input: []Finding{
				{Agent: "rules", Severity: SeverityHigh, Location: loc},
				{Agent: "lint", Severity: SeverityHigh, Location: loc}, // different agent — kept
			},
			wantLen: 2,
		},
		{
			name: "same agent same location keeps higher severity",
			input: []Finding{
				{Agent: "rules", Severity: SeverityLow, Location: loc},
				{Agent: "rules", Severity: SeverityHigh, Location: loc},
			},
			wantLen: 1,
			wantSev: SeverityHigh,
		},
		{
			name: "same agent different locations both kept",
			input: []Finding{
				{Agent: "rules", Severity: SeverityLow, Location: Location{File: "a.go", LineStart: 1, LineEnd: 5}},
				{Agent: "rules", Severity: SeverityLow, Location: Location{File: "a.go", LineStart: 6, LineEnd: 10}},
			},
			wantLen: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Deduplicate(tc.input)
			if len(got) != tc.wantLen {
				t.Errorf("len=%d, want %d", len(got), tc.wantLen)
			}
			if tc.wantSev != "" && len(got) == 1 && got[0].Severity != tc.wantSev {
				t.Errorf("severity=%s, want %s", got[0].Severity, tc.wantSev)
			}
		})
	}
}
