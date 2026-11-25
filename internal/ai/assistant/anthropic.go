package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"
)

// Claude talks to the Claude API through the official Go SDK. Refusals are
// re-served by the server-side fallback ("default" routing).
type Claude struct {
	client anthropic.Client
	model  string
}

func NewClaude(apiKey, model string) *Claude {
	if model == "" {
		model = "claude-opus-5-5"
	}
	return &Claude{client: anthropic.NewClient(option.WithAPIKey(apiKey)), model: model}
}

func (c *Claude) Name() string  { return "anthropic" }
func (c *Claude) Model() string { return c.model }

func (c *Claude) Converse(ctx context.Context, system string, turns []Turn, tools []Tool, exec Exec) (string, error) {
	var msgs []anthropic.BetaMessageParam
	for _, t := range turns {
		if t.Role == "assistant" {
			msgs = append(msgs, anthropic.BetaMessageParam{OfAssistant: &anthropic.BetaAssistantMessageParam{
				Content: []anthropic.BetaContentBlockParamUnion{anthropic.NewBetaTextBlock(t.Text)},
			}})
		} else {
			msgs = append(msgs, anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(t.Text)))
		}
	}
	var toolParams []anthropic.BetaToolUnionParam
	for _, t := range tools {
		tp := anthropic.BetaToolParam{
			Name:        t.Name,
			Description: anthropic.String(t.Description),
			InputSchema: anthropic.BetaToolInputSchemaParam{Properties: t.Schema, Required: t.Required},
		}
		toolParams = append(toolParams, anthropic.BetaToolUnionParam{OfTool: &tp})
	}

	var final strings.Builder
	for step := 0; step < 8; step++ {
		resp, err := c.client.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
			Model:        c.model,
			MaxTokens:    8000,
			System:       []anthropic.BetaTextBlockParam{{Text: system}},
			Messages:     msgs,
			Tools:        toolParams,
			OutputConfig: anthropic.BetaOutputConfigParam{Effort: anthropic.BetaOutputConfigEffortMedium},
			Fallbacks:    anthropic.BetaFallbacksParamOfDefault(),
			Betas:        []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
		})
		if err != nil {
			var apiErr *anthropic.Error
			if errors.As(err, &apiErr) {
				switch apiErr.StatusCode {
				case 401, 403:
					return "", fmt.Errorf("Claude API rejected the key (HTTP %d)", apiErr.StatusCode)
				case 429:
					return "", fmt.Errorf("Claude API rate limit reached, try again shortly")
				}
				return "", fmt.Errorf("Claude API error (HTTP %d)", apiErr.StatusCode)
			}
			return "", fmt.Errorf("cannot reach the Claude API: %w", err)
		}
		if resp.StopReason == anthropic.BetaStopReasonRefusal {
			return "", fmt.Errorf("the model declined this request")
		}
		msgs = append(msgs, resp.ToParam())

		var results []anthropic.BetaContentBlockParamUnion
		final.Reset()
		for _, block := range resp.Content {
			switch b := block.AsAny().(type) {
			case anthropic.BetaTextBlock:
				final.WriteString(b.Text)
			case anthropic.BetaToolUseBlock:
				out, isErr := exec(ctx, b.Name, json.RawMessage(b.JSON.Input.Raw()))
				results = append(results, anthropic.NewBetaToolResultBlock(b.ID, truncate(out, 12000), isErr))
			}
		}
		if resp.StopReason != anthropic.BetaStopReasonToolUse {
			return strings.TrimSpace(final.String()), nil
		}
		msgs = append(msgs, anthropic.NewBetaUserMessage(results...))
	}
	return strings.TrimSpace(final.String()), nil
}
