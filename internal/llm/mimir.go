package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// MimirClient sends requests to the mimir LLM using the OpenAI-compatible API.
type MimirClient struct {
	baseURL     string
	apiKey      string
	model       string
	defaultTemp *float64
	defaultSeed *int
	defaultJSON *bool
	http        *http.Client
}

// NewMimirClient creates a MimirClient for the given base URL and API key.
func NewMimirClient(baseURL, apiKey, model string, defaultTemp *float64, defaultSeed *int, defaultJSON *bool) *MimirClient {
	return &MimirClient{
		baseURL:     baseURL,
		apiKey:      apiKey,
		model:       model,
		defaultTemp: defaultTemp,
		defaultSeed: defaultSeed,
		defaultJSON: defaultJSON,
		http:        &http.Client{Timeout: 120 * time.Second},
	}
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type openAIRequest struct {
	Model          string          `json:"model"`
	Messages       []openAIMessage `json:"messages"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	Temperature    float64         `json:"temperature"`
	Seed           *int            `json:"seed,omitempty"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type statusError struct {
	statusCode int
}

func (e *statusError) Error() string {
	return fmt.Sprintf("mimir status %d", e.statusCode)
}

func isStatus400(err error) bool {
	var se *statusError
	if errors.As(err, &se) {
		return se.statusCode == http.StatusBadRequest
	}
	return false
}

func intPtr(i int) *int {
	return &i
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
		msgs[i] = openAIMessage(m)
	}

	var temp float64
	if req.Temperature != nil {
		temp = *req.Temperature
	} else if c.defaultTemp != nil {
		temp = *c.defaultTemp
	} else {
		temp = 0.0
	}

	var seed *int
	if req.Seed != nil {
		seed = req.Seed
	} else if c.defaultSeed != nil {
		seed = c.defaultSeed
	} else {
		seed = intPtr(42)
	}

	var respFmt *responseFormat
	jsonEnabled := c.defaultJSON == nil || *c.defaultJSON
	if jsonEnabled && req.JSONMode {
		respFmt = &responseFormat{Type: "json_object"}
	}

	reqPayload := openAIRequest{
		Model:          model,
		Messages:       msgs,
		MaxTokens:      req.MaxTokens,
		Temperature:    temp,
		Seed:           seed,
		ResponseFormat: respFmt,
	}

	body, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("mimir marshal: %w", err)
	}

	resp, err := c.doWithRetry(ctx, body)
	if err != nil {
		if isStatus400(err) {
			slog.Warn("mimir HTTP call failed with status 400, retrying without Seed and ResponseFormat", "error", err)
			reqPayload.Seed = nil
			reqPayload.ResponseFormat = nil
			fallbackBody, fallbackErr := json.Marshal(reqPayload)
			if fallbackErr != nil {
				return nil, fmt.Errorf("mimir marshal fallback: %w", fallbackErr)
			}
			return c.doWithRetry(ctx, fallbackBody)
		}
		return nil, err
	}
	return resp, nil
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
		if isStatus400(err) {
			return nil, err
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

	if httpResp.StatusCode != http.StatusOK {
		return nil, &statusError{statusCode: httpResp.StatusCode}
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
