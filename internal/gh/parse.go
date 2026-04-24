package gh

import (
	"encoding/json"
	"fmt"
	"strings"
)

// prViewResponse mirrors the JSON returned by `gh pr view --json ...`.
type prViewResponse struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	BaseRefName string `json:"baseRefName"`
	HeadRefName string `json:"headRefName"`
	Additions   int    `json:"additions"`
	Deletions   int    `json:"deletions"`
	Labels      []struct {
		Name string `json:"name"`
	} `json:"labels"`
	Commits []struct {
		OID             string `json:"oid"`
		MessageHeadline string `json:"messageHeadline"`
	} `json:"commits"`
	Files []struct {
		Path      string `json:"path"`
		Additions int    `json:"additions"`
		Deletions int    `json:"deletions"`
	} `json:"files"`
}

// parsePRView parses the JSON output of `gh pr view --json ...` into a PullRequest.
func parsePRView(data []byte, owner, repo string) (*PullRequest, error) {
	var raw prViewResponse
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse pr view: %w", err)
	}

	pr := &PullRequest{
		Owner:     owner,
		Repo:      repo,
		Number:    raw.Number,
		Title:     raw.Title,
		Body:      raw.Body,
		BaseRef:   raw.BaseRefName,
		HeadRef:   raw.HeadRefName,
		Additions: raw.Additions,
		Deletions: raw.Deletions,
	}

	for _, l := range raw.Labels {
		pr.Labels = append(pr.Labels, l.Name)
	}

	for _, c := range raw.Commits {
		pr.Commits = append(pr.Commits, Commit{SHA: c.OID, Message: c.MessageHeadline})
	}

	for _, f := range raw.Files {
		pr.Files = append(pr.Files, ChangedFile{
			Path:      f.Path,
			Additions: f.Additions,
			Deletions: f.Deletions,
		})
	}

	return pr, nil
}

// attachPatches parses a unified diff and populates Patch fields on the PR's changed files.
func attachPatches(pr *PullRequest, diff string) {
	pr.Diff = diff
	patches := splitDiffByFile(diff)
	for i, f := range pr.Files {
		if patch, ok := patches[f.Path]; ok {
			pr.Files[i].Patch = patch
		}
	}
}

// splitDiffByFile splits a unified diff into per-file patches keyed by file path.
func splitDiffByFile(diff string) map[string]string {
	result := make(map[string]string)
	var currentFile string
	var buf strings.Builder

	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			if currentFile != "" {
				result[currentFile] = buf.String()
			}
			buf.Reset()
			// extract b-side path: "diff --git a/foo b/foo" → "foo"
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				currentFile = strings.TrimPrefix(parts[3], "b/")
			}
		}
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	if currentFile != "" {
		result[currentFile] = buf.String()
	}
	return result
}
