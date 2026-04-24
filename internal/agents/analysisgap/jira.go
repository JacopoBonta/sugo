package analysisgap

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// JiraIssue holds the fields relevant to analysis-gap checking.
type JiraIssue struct {
	Key     string
	Summary string
	Description string
}

// JiraClient fetches Jira issue data.
type JiraClient interface {
	GetIssue(ctx context.Context, key string) (*JiraIssue, error)
}

// httpJiraClient fetches issues via the Jira REST API v2.
type httpJiraClient struct {
	baseURL string
	user    string
	token   string
	http    *http.Client
}

// newHTTPJiraClient creates a Jira client using basic auth.
func newHTTPJiraClient(baseURL, user, token string) *httpJiraClient {
	return &httpJiraClient{
		baseURL: baseURL,
		user:    user,
		token:   token,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *httpJiraClient) GetIssue(ctx context.Context, key string) (*JiraIssue, error) {
	url := fmt.Sprintf("%s/rest/api/2/issue/%s", c.baseURL, key)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("jira request: %w", err)
	}
	req.SetBasicAuth(c.user, c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jira http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jira status %d for issue %s", resp.StatusCode, key)
	}

	var raw struct {
		Key    string `json:"key"`
		Fields struct {
			Summary     string `json:"summary"`
			Description string `json:"description"`
		} `json:"fields"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("jira decode: %w", err)
	}

	return &JiraIssue{
		Key:         raw.Key,
		Summary:     raw.Fields.Summary,
		Description: raw.Fields.Description,
	}, nil
}
