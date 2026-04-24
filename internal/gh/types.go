// Package gh provides types and a client for fetching GitHub PR data via the gh CLI.
package gh

// PullRequest holds all data about a GitHub pull request needed for analysis.
type PullRequest struct {
	Owner      string
	Repo       string
	Number     int
	Title      string
	Body       string
	BaseRef    string
	HeadRef    string
	Labels     []string
	Commits    []Commit
	Files      []ChangedFile
	Diff       string
	Additions  int
	Deletions  int
}

// Commit represents a single commit in the PR.
type Commit struct {
	SHA     string
	Message string
}

// ChangedFile represents a file changed in the PR.
type ChangedFile struct {
	Path      string
	Additions int
	Deletions int
	Patch     string // per-file unified diff
}
