// AI adapter.
// A thin, provider-agnostic seam over "make me one message call".
// Right now the only implementation is Claude (see claude.go), but any
// provider can satisfy AIClient without the callers needing to care.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
)

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

// newAIClient builds the AIClient for the configured provider. Callers depend on
// the interface, not the provider, so swapping models/vendors is a config change
// here rather than a code change at the call sites.
//
// Select with AI_PROVIDER (default "claude"). Add a case per new provider.
func newAIClient() (AIClient, error) {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("AI_PROVIDER")))
	switch provider {
	case "", "claude", "anthropic":
		return newClaudeClient()
	default:
		return nil, fmt.Errorf("unknown AI_PROVIDER %q (supported: claude)", provider)
	}
}
