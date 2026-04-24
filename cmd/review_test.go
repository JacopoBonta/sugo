package cmd

import "testing"

func TestParseReviewArg(t *testing.T) {
	tests := []struct {
		arg     string
		owner   string
		repo    string
		number  int
		wantErr bool
	}{
		{"owner/repo#42", "owner", "repo", 42, false},
		{"my-org/my-repo#1", "my-org", "my-repo", 1, false},
		{"foo/bar.baz#999", "foo", "bar.baz", 999, false},
		{"badformat", "", "", 0, true},
		{"owner#42", "", "", 0, true},
		{"owner/repo", "", "", 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.arg, func(t *testing.T) {
			owner, repo, number, err := parseReviewArg(tc.arg)
			if (err != nil) != tc.wantErr {
				t.Fatalf("wantErr=%v, got err=%v", tc.wantErr, err)
			}
			if !tc.wantErr {
				if owner != tc.owner || repo != tc.repo || number != tc.number {
					t.Errorf("got (%s, %s, %d), want (%s, %s, %d)", owner, repo, number, tc.owner, tc.repo, tc.number)
				}
			}
		})
	}
}
