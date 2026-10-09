package agent

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"example.com/support-shell/internal/core"
)

type Model struct {
	Registry    *core.Registry
	URL         string
	ModelName   string
	Key         string
	Client      *http.Client
	Messages    []message
	SessionID   string
	OnTool      func(ToolEvent)
	LastReport  string
	LastContext string
	MaxCalls    int
	MaxRounds   int
	mu          sync.Mutex
}
type ToolEvent struct {
	Command  string
	Input    map[string]any
	Duration time.Duration
	Err      error
	Source   string
}

type message struct {
	Role       string     `json:"role"`
	Content    any        `json:"content,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}
type toolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}
type response struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func New(r *core.Registry) *Model {
	url := strings.TrimRight(os.Getenv("SUPPORT_API_BASE"), "/")
	if url == "" {
		url = "https://api.openai.com/v1"
	}
	model := os.Getenv("SUPPORT_MODEL")
	if model == "" {
		model = "gpt-4.1-mini"
	}
	return &Model{Registry: r, URL: url, ModelName: model, Key: os.Getenv("SUPPORT_API_KEY"), Client: &http.Client{Timeout: modelTimeout()}, SessionID: newSessionID(), MaxCalls: envBounded("SUPPORT_AGENT_MAX_CALLS", 16, 1, 100), MaxRounds: envBounded("SUPPORT_AGENT_MAX_ROUNDS", 8, 1, 20)}
}
func envBounded(name string, fallback, min, max int) int {
	n, err := strconv.Atoi(os.Getenv(name))
	if err != nil || n < min || n > max {
		return fallback
	}
	return n
}

func (a *Model) Report() string       { a.mu.Lock(); defer a.mu.Unlock(); return a.LastReport }
func (a *Model) ContextLabel() string { a.mu.Lock(); defer a.mu.Unlock(); return a.LastContext }
func (a *Model) SetContext(v string)  { a.mu.Lock(); defer a.mu.Unlock(); a.LastContext = v }

func modelTimeout() time.Duration {
	n, err := strconv.Atoi(os.Getenv("SUPPORT_AGENT_TIMEOUT_SECONDS"))
	if err == nil && n >= 10 && n <= 600 {
		return time.Duration(n) * time.Second
	}
	return 180 * time.Second
}

// Keep tool output small enough for the next LLM request. The full result is
// still printed by a direct CLI command; this only limits LLM context.
func compactToolResult(v any) string {
	const maxRunes = 6000
	s := core.JSON(v)
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return core.JSON(map[string]any{
		"truncated":           true,
		"original_characters": len(r),
		"note":                "Tool output truncated for agent context. Request a narrower scope for more detail.",
		"preview":             string(r[:maxRunes]),
	})
}

func isTimeout(err error) bool {
	var ne net.Error
	return errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout())
}

func newSessionID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return hex.EncodeToString(b[:4]) + "-" + hex.EncodeToString(b[4:6]) + "-" + hex.EncodeToString(b[6:8]) + "-" + hex.EncodeToString(b[8:10]) + "-" + hex.EncodeToString(b[10:])
}

var invalidToolName = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

func toolAlias(id string) string {
	name := invalidToolName.ReplaceAllString(id, "_")
	if name == "" {
		name = "tool"
	}
	return name
}

func (a *Model) Reset() {
	a.Messages = nil
	a.SessionID = newSessionID()
	a.mu.Lock()
	a.LastReport = ""
	a.mu.Unlock()
}
func (a *Model) Ask(ctx context.Context, prompt string) (string, error) {
	// An interrupted turn must not leave incomplete assistant/tool messages in history.
	previousMessages := append([]message(nil), a.Messages...)
	defer func() {
		if ctx.Err() != nil {
			a.Messages = previousMessages
		}
	}()
	if err := ctx.Err(); err != nil { return "", err }
	if a.Key == "" && a.URL == "https://api.openai.com/v1" {
		return "", errors.New("set SUPPORT_API_KEY (and optionally SUPPORT_API_BASE / SUPPORT_MODEL)")
	}
	if len(a.Messages) == 0 {
        a.Messages = append(a.Messages, message{Role: "system", Content: systemPrompt})
	}
	a.mu.Lock()
	contextLabel := a.LastContext
	a.mu.Unlock()
	if contextLabel != "" {
		prompt = "Selected Kubernetes context: " + contextLabel + ". Confirm actual scope in results.\n" + prompt
	}
	a.Messages = append(a.Messages, message{Role: "user", Content: prompt})
	if len(a.Messages) > 36 {
		a.Messages = append(a.Messages[:1], a.Messages[len(a.Messages)-30:]...)
	}
	tools := []any{}
	toolIDs := map[string]string{}
	for _, c := range a.Registry.List() {
		if c.Risk != "read" || strings.HasPrefix(c.ID, "demo.cluster.") {
			continue
		}
		properties := map[string]any{}
		required := []string{}
		for _, arg := range c.Args {
			properties[arg] = map[string]any{"type": "string"}
			required = append(required, arg)
		}
		if c.ID == "k8s.pod.logs" {
			for _, flag := range []string{"previous", "timestamps", "tail", "since", "container"} {
				properties[flag] = map[string]any{"type": "string", "description": "Optional kubectl logs flag; previous/timestamps accept true or false"}
			}
		}
		name := toolAlias(c.ID)
		if original, exists := toolIDs[name]; exists && original != c.ID {
			for n := 2; ; n++ {
				candidate := fmt.Sprintf("%s_%d", name, n)
				if _, used := toolIDs[candidate]; !used {
					name = candidate
					break
				}
			}
		}
		toolIDs[name] = c.ID
		tools = append(tools, map[string]any{"type": "function", "function": map[string]any{"name": name, "description": c.Description, "parameters": map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}}})
	}
	callsUsed := 0
	rounds := a.MaxRounds
	if rounds == 0 {
		rounds = 8
	}
	callLimit := a.MaxCalls
	if callLimit == 0 {
		callLimit = 16
	}
	for step := 0; step < rounds+1; step++ {
		if err := ctx.Err(); err != nil { return "", err }
		if step == rounds {
			return "", fmt.Errorf("agent reached %d model rounds; narrow your request", rounds)
		}
		if callsUsed >= callLimit {
			tools = nil
		}
		payload := map[string]any{"model": a.ModelName, "messages": a.Messages}
		if len(tools) > 0 {
			payload["tools"] = tools
			payload["tool_choice"] = "auto"
		}
		body, _ := json.Marshal(payload)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.URL+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "support-shell/0.5")
		if a.SessionID == "" {
			a.SessionID = newSessionID()
		}
		req.Header.Set("x-opencode-session", a.SessionID)
		if a.Key != "" {
			req.Header.Set("Authorization", "Bearer "+a.Key)
		}
		resp, err := a.Client.Do(req)
		// A model request after tool calls may take longer than one without
		// tools. Retry only the HTTP model request, never tool execution.
		if err != nil && isTimeout(err) && ctx.Err() == nil {
			req2, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, a.URL+"/chat/completions", bytes.NewReader(body))
			if reqErr != nil {
				return "", reqErr
			}
			req2.Header = req.Header.Clone()
			resp, err = a.Client.Do(req2)
		}
		if err != nil {
			return "", fmt.Errorf("model request failed (timeout %s; configure SUPPORT_AGENT_TIMEOUT_SECONDS): %w", a.Client.Timeout, err)
		}
		raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		resp.Body.Close()
		if err != nil {
			return "", err
		}
		if resp.StatusCode/100 != 2 {
			return "", fmt.Errorf("model HTTP %d: %s", resp.StatusCode, string(raw))
		}
		var out response
		if err = json.Unmarshal(raw, &out); err != nil {
			return "", err
		}
		if len(out.Choices) == 0 {
			return "", errors.New("empty model response")
		}
		msg := out.Choices[0].Message
		a.Messages = append(a.Messages, msg)
		if len(msg.ToolCalls) == 0 {
			if s, ok := msg.Content.(string); ok {
				a.mu.Lock()
				a.LastReport = s
				a.mu.Unlock()
				return s, nil
			}
			answer := core.JSON(msg.Content)
			a.mu.Lock()
			a.LastReport = answer
			a.mu.Unlock()
			return answer, nil
		}
		for _, call := range msg.ToolCalls {
			if err := ctx.Err(); err != nil { return "", err }
			if callsUsed >= callLimit {
				a.Messages = append(a.Messages, message{Role: "tool", ToolCallID: call.ID, Content: `{"error":"tool budget exceeded; provide final diagnosis from observations already collected"}`})
				continue
			}
			callsUsed++
			var input map[string]any
			if err = json.Unmarshal([]byte(call.Function.Arguments), &input); err != nil {
				input = map[string]any{}
			}
			commandID, known := toolIDs[call.Function.Name]
			if !known {
				a.Messages = append(a.Messages, message{Role: "tool", ToolCallID: call.ID, Content: core.JSON(map[string]string{"error": "unknown agent tool"})})
				continue
			}
			started := time.Now()
			result, e := a.Registry.Execute(ctx, commandID, input)
			elapsed := time.Since(started)
			content := compactToolResult(result)
			if e != nil {
				content = core.JSON(map[string]string{"error": e.Error()})
			}
			if a.OnTool != nil {
				a.OnTool(ToolEvent{Command: commandID, Input: input, Duration: elapsed, Err: e, Source: commandID})
			}
			a.Messages = append(a.Messages, message{Role: "tool", ToolCallID: call.ID, Content: content})
		}
	}
	return "", errors.New("agent exhausted tool rounds")
}

func (a *Model) SetTrace(fn func(string, map[string]any, time.Duration, error)) {
	if fn == nil {
		a.OnTool = nil
		return
	}
	a.OnTool = func(e ToolEvent) { fn(e.Command, e.Input, e.Duration, e.Err) }
}
