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
	Temperature float64
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
