package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// MimirClient sends requests to the mimir LLM using the OpenAI-compatible API.
type MimirClient struct {
	baseURL string
	apiKey  string
	model   string
	http    *http.Client
}

// NewMimirClient creates a MimirClient for the given base URL and API key.
func NewMimirClient(baseURL, apiKey, model string) *MimirClient {
	return &MimirClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		http:    &http.Client{Timeout: 120 * time.Second},
	}
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// Complete sends a chat completion request to mimir.
func (c *MimirClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = c.model
	}

	msgs := make([]openAIMessage, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = openAIMessage{Role: m.Role, Content: m.Content}
	}

	body, err := json.Marshal(openAIRequest{
		Model:       model,
		Messages:    msgs,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("mimir marshal: %w", err)
	}

	return c.doWithRetry(ctx, body)
}

func (c *MimirClient) doWithRetry(ctx context.Context, body []byte) (*CompletionResponse, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(2 * time.Second):
			}
		}
		resp, err := c.doOnce(ctx, body)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (c *MimirClient) doOnce(ctx context.Context, body []byte) (*CompletionResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("mimir request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	httpResp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("mimir http: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode == http.StatusTooManyRequests || httpResp.StatusCode >= 500 {
		return nil, fmt.Errorf("mimir status %d", httpResp.StatusCode)
	}
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mimir status %d", httpResp.StatusCode)
	}

	var result openAIResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("mimir decode: %w", err)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("mimir: no choices in response")
	}

	return &CompletionResponse{
		Content: result.Choices[0].Message.Content,
		Usage: Usage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
		},
	}, nil
}
