package finding

// dedupKey uniquely identifies a finding by agent + location.
type dedupKey struct {
	agent     string
	file      string
	lineStart int
	lineEnd   int
}

// Deduplicate removes duplicate findings from the same agent at the same location,
// keeping the one with the highest severity. Findings from different agents at the
// same location are kept intentionally.
func Deduplicate(findings []Finding) []Finding {
	best := make(map[dedupKey]Finding, len(findings))
	for _, f := range findings {
		k := dedupKey{f.Agent, f.Location.File, f.Location.LineStart, f.Location.LineEnd}
		if existing, ok := best[k]; !ok || f.Severity.Rank() > existing.Severity.Rank() {
			best[k] = f
		}
	}
	result := make([]Finding, 0, len(best))
	for _, f := range best {
		result = append(result, f)
	}
	return result
}
