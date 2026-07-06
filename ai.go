// AI adapter.
// A thin, provider-agnostic seam over "make me one message call".
// Right now the only implementation is Claude (see claude.go), but any
// provider can satisfy AIClient without the callers needing to care.
package main

import "context"

// AIMessageRequest is a single, one-shot message call.
// System is optional context/instructions; Prompt is the user turn.
type AIMessageRequest struct {
	System    string
	Prompt    string
	MaxTokens int64 // 0 => the implementation picks a sane default
}

// AIClient makes a single AI message call and returns the text response.
// Implementations should be safe for concurrent use.
type AIClient interface {
	Message(ctx context.Context, req AIMessageRequest) (string, error)
}
