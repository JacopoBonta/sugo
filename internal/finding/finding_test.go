package finding

import (
	"encoding/json"
	"testing"
)

func TestSeverityRank(t *testing.T) {
	if SeverityHigh.Rank() <= SeverityMedium.Rank() {
		t.Error("high must outrank medium")
	}
	if SeverityMedium.Rank() <= SeverityLow.Rank() {
		t.Error("medium must outrank low")
	}
}

func TestFindingJSONMarshal(t *testing.T) {
	fix := "rename it"
	f := Finding{
		Agent:    "rules",
		Severity: SeverityHigh,
		Type:     TypeFix,
		Location: Location{File: "main.go", LineStart: 1, LineEnd: 5},
		Message:  "bad branch",
		Fix:      &fix,
	}
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	var got Finding
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Fix == nil || *got.Fix != fix {
		t.Errorf("Fix round-trip failed: got %v", got.Fix)
	}
}

func TestFindingNullFix(t *testing.T) {
	f := Finding{Agent: "logic", Severity: SeverityLow, Type: TypeAttentionPoint}
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "" {
		t.Fatal("empty marshal")
	}
	// Fix must be null in JSON
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	fixVal, ok := raw["fix"]
	if !ok {
		t.Fatal("fix key missing")
	}
	if fixVal != nil {
		t.Errorf("fix should be null, got %v", fixVal)
	}
}
