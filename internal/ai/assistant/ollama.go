package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Ollama talks to an Ollama server on the user's LAN (native /api/chat
// with tool calling), so nothing leaves the home network.
type Ollama struct {
	base  string
	model string
	http  *http.Client
}

func NewOllama(endpoint, model string) (*Ollama, error) {
	u, err := url.Parse(strings.TrimRight(endpoint, "/"))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("invalid Ollama endpoint %q", endpoint)
	}
	if model == "" {
		model = "qwen3:8b"
	}
	return &Ollama{base: u.String(), model: model, http: &http.Client{Timeout: 3 * time.Minute}}, nil
}

func (o *Ollama) Name() string  { return "ollama" }
func (o *Ollama) Model() string { return o.model }

type ollamaMsg struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
	ToolName  string           `json:"tool_name,omitempty"`
}

type ollamaToolCall struct {
	Function struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"function"`
}

func (o *Ollama) Converse(ctx context.Context, system string, turns []Turn, tools []Tool, exec Exec) (string, error) {
	msgs := []ollamaMsg{{Role: "system", Content: system}}
	for _, t := range turns {
		msgs = append(msgs, ollamaMsg{Role: t.Role, Content: t.Text})
	}
	var toolDefs []map[string]any
	for _, t := range tools {
		toolDefs = append(toolDefs, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name": t.Name, "description": t.Description,
				"parameters": map[string]any{"type": "object", "properties": t.Schema, "required": t.Required},
			},
		})
	}
	for step := 0; step < 8; step++ {
		body, _ := json.Marshal(map[string]any{"model": o.model, "messages": msgs, "tools": toolDefs, "stream": false})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.base+"/api/chat", bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := o.http.Do(req)
		if err != nil {
			return "", fmt.Errorf("cannot reach Ollama at %s: %w", o.base, err)
		}
		var out struct {
			Message ollamaMsg `json:"message"`
			Error   string    `json:"error"`
		}
		err = json.NewDecoder(resp.Body).Decode(&out)
		resp.Body.Close()
		if err != nil {
			return "", fmt.Errorf("bad response from Ollama: %w", err)
		}
		if out.Error != "" || resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("Ollama error: %s (HTTP %d)", out.Error, resp.StatusCode)
		}
		msgs = append(msgs, out.Message)
		if len(out.Message.ToolCalls) == 0 {
			return strings.TrimSpace(out.Message.Content), nil
		}
		for _, tc := range out.Message.ToolCalls {
			res, _ := exec(ctx, tc.Function.Name, tc.Function.Arguments)
			msgs = append(msgs, ollamaMsg{Role: "tool", Content: truncate(res, 12000), ToolName: tc.Function.Name})
		}
	}
	return "", fmt.Errorf("assistant stopped after too many tool calls")
}
