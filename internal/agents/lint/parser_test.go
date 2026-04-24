package lint

import (
	"os"
	"testing"
)

func TestGolangciParser(t *testing.T) {
	data, err := os.ReadFile("../../../testdata/agents/lint/golangci_output.json")
	if err != nil {
		t.Fatal(err)
	}
	p := golangciParser{}
	issues, err := p.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 2 {
		t.Errorf("got %d issues, want 2", len(issues))
	}
	if issues[0].Rule != "errcheck" {
		t.Errorf("rule = %q, want errcheck", issues[0].Rule)
	}
	if issues[0].Line != 12 {
		t.Errorf("line = %d, want 12", issues[0].Line)
	}
}

func TestEslintParser(t *testing.T) {
	data, err := os.ReadFile("../../../testdata/agents/lint/eslint_output.json")
	if err != nil {
		t.Fatal(err)
	}
	p := eslintParser{}
	issues, err := p.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 {
		t.Errorf("got %d issues, want 1", len(issues))
	}
	if issues[0].Rule != "no-unused-vars" {
		t.Errorf("rule = %q", issues[0].Rule)
	}
}

func TestGenericParser(t *testing.T) {
	data, err := os.ReadFile("../../../testdata/agents/lint/generic_output.txt")
	if err != nil {
		t.Fatal(err)
	}
	p := genericParser{}
	issues, err := p.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 3 {
		t.Errorf("got %d issues, want 3", len(issues))
	}
	if issues[0].Line != 10 {
		t.Errorf("line = %d, want 10", issues[0].Line)
	}
}

func TestSelectParser(t *testing.T) {
	tests := []struct {
		cmd      string
		wantType string
	}{
		{"golangci-lint", "golangciParser"},
		{"eslint", "eslintParser"},
		{"ruff", "genericParser"},
	}
	for _, tc := range tests {
		t.Run(tc.cmd, func(t *testing.T) {
			p := selectParser(tc.cmd)
			switch tc.wantType {
			case "golangciParser":
				if _, ok := p.(golangciParser); !ok {
					t.Errorf("want golangciParser, got %T", p)
				}
			case "eslintParser":
				if _, ok := p.(eslintParser); !ok {
					t.Errorf("want eslintParser, got %T", p)
				}
			case "genericParser":
				if _, ok := p.(genericParser); !ok {
					t.Errorf("want genericParser, got %T", p)
				}
			}
		})
	}
}
