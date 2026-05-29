// Package llm provides an abstraction for LLM API communication.
package llm

import "context"

// Message is a single turn in a conversation.
type Message struct {
	Role    string // "system", "user", or "assistant"
	Content string
}

// CompletionRequest is sent to the LLM.
type CompletionRequest struct {
	Model       string
	Messages    []Message
	MaxTokens   int
	Temperature *float64
	Seed        *int
	JSONMode    bool
}

// Float64 returns a pointer to the given float64 value.
func Float64(v float64) *float64 {
	return &v
}

// Int returns a pointer to the given int value.
func Int(v int) *int {
	return &v
}

// CompletionResponse contains the LLM's reply.
type CompletionResponse struct {
	Content string
	Usage   Usage
}

// Usage reports token consumption.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
}

// Client sends prompts to an LLM provider and returns completions.
type Client interface {
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
}
