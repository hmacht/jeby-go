// Claude client — the Anthropic implementation of AIClient.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// We default to the most capable Opus model. Adaptive thinking is on so Claude
// decides how much to reason per request.
const claudeModel = anthropic.ModelClaudeOpus4_8

// claudeClient wraps the Anthropic SDK and satisfies AIClient.
type claudeClient struct {
	client anthropic.Client
	model  anthropic.Model
}

// newClaudeClient reads ANTHROPIC_API_KEY from the environment.
func newClaudeClient() (*claudeClient, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, errors.New("ANTHROPIC_API_KEY not set")
	}
	return &claudeClient{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:  claudeModel,
	}, nil
}

// Message sends a single-turn message and returns the concatenated text.
func (c *claudeClient) Message(ctx context.Context, req AIMessageRequest) (string, error) {
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	adaptive := anthropic.ThinkingConfigAdaptiveParam{}
	params := anthropic.MessageNewParams{
		Model:     c.model,
		MaxTokens: maxTokens,
		Thinking:  anthropic.ThinkingConfigParamUnion{OfAdaptive: &adaptive},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(req.Prompt)),
		},
	}
	if req.System != "" {
		params.System = []anthropic.TextBlockParam{{Text: req.System}}
	}

	resp, err := c.client.Messages.New(ctx, params)
	if err != nil {
		return "", fmt.Errorf("claude message: %w", err)
	}

	// A refusal comes back as a normal 200 with an empty/partial body — surface
	// it rather than returning silently empty text.
	if resp.StopReason == anthropic.StopReasonRefusal {
		return "", fmt.Errorf("claude refused the request: %s", resp.StopDetails.Explanation)
	}

	var out strings.Builder
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			out.WriteString(t.Text)
		}
	}
	return out.String(), nil
}
