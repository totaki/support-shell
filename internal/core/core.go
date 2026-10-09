package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

type Command struct {
	ID          string                                             `json:"id"`
	Path        string                                             `json:"path"`
	Description string                                             `json:"description"`
	Risk        string                                             `json:"risk"`
	Args        []string                                           `json:"args,omitempty"`
	Handler     func(context.Context, map[string]any) (any, error) `json:"-"`
	Completer   func(context.Context, string) []string             `json:"-"`
}
type Registry struct{ commands map[string]Command }

func NewRegistry() *Registry { return &Registry{commands: map[string]Command{}} }
func (r *Registry) Add(c Command) error {
	if c.ID == "" || c.Path == "" || c.Handler == nil {
		return errors.New("invalid command")
	}
	if _, ok := r.commands[c.Path]; ok {
		return fmt.Errorf("duplicate command %s", c.Path)
	}
	r.commands[c.Path] = c
	return nil
}
func (r *Registry) List() []Command {
	var v []Command
	for _, c := range r.commands {
		v = append(v, c)
	}
	sort.Slice(v, func(i, j int) bool { return v[i].Path < v[j].Path })
	return v
}
func (r *Registry) Resolve(input string) (Command, []string, bool) {
	parts := strings.Fields(input)
	for n := len(parts); n > 0; n-- {
		if c, ok := r.commands[strings.Join(parts[:n], " ")]; ok {
			return c, parts[n:], true
		}
	}
	return Command{}, nil, false
}
func (r *Registry) Complete(prefix string) []string {
	seen := map[string]bool{}
	var out []string
	for _, c := range r.List() {
		if strings.HasPrefix(c.Path, prefix) {
			suffix := strings.TrimPrefix(c.Path, prefix)
			word := strings.SplitN(suffix, " ", 2)[0]
			s := prefix + word
			if !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
	}
	return out
}
func (r *Registry) CompleteWithContext(ctx context.Context, prefix string) []string {
	for _, c := range r.commands {
		if c.Completer != nil && strings.HasPrefix(prefix, c.Path+" ") {
			return c.Completer(ctx, strings.TrimPrefix(prefix, c.Path+" "))
		}
	}
	return r.Complete(prefix)
}

func (r *Registry) Execute(ctx context.Context, id string, in map[string]any) (any, error) {
	return r.execute(ctx, id, in, false)
}

// ExecuteApproved must only be called after an explicit interactive user confirmation.
func (r *Registry) ExecuteApproved(ctx context.Context, id string, in map[string]any) (any, error) {
	return r.execute(ctx, id, in, true)
}

func (r *Registry) execute(ctx context.Context, id string, in map[string]any, approved bool) (any, error) {
	for _, c := range r.commands {
		if c.ID == id {
			if c.Risk != "read" && !approved {
				return nil, fmt.Errorf("mutating command %s requires explicit approval (not implemented)", id)
			}
			return c.Handler(ctx, in)
		}
	}
	return nil, fmt.Errorf("unknown command ID %s", id)
}
func ParseInput(c Command, args []string) (map[string]any, error) {
	out := map[string]any{}
	for i := 0; i < len(args); i++ {
		s := args[i]
		if strings.HasPrefix(s, "--") {
			key, val, ok := strings.Cut(strings.TrimPrefix(s, "--"), "=")
			if !ok {
				if i+1 >= len(args) {
					return nil, fmt.Errorf("flag --%s requires value", key)
				}
				i++
				val = args[i]
			}
			out[key] = val
		} else {
			idx := 0
			for {
				if idx >= len(c.Args) {
					return nil, fmt.Errorf("unexpected argument: %s", s)
				}
				if _, exists := out[c.Args[idx]]; !exists {
					out[c.Args[idx]] = s
					break
				}
				idx++
			}
		}
	}
	return out, nil
}
func JSON(v any) string {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}
