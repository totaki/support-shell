package shell

import (
	"bufio"
	"context"
	"encoding/json"
	"example.com/support-shell/internal/core"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"errors"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

type Agent interface {
	Ask(context.Context, string) (string, error)
}
type Shell struct {
	Registry       *core.Registry
	Agent          Agent
	History        []string
	HistoryPath    string
	PendingContext string
	CurrentContext string
	OutputFormat string // "table" (default) or "json"
}

func (s *Shell) Handle(ctx context.Context, line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return true
	}
	s.History = append(s.History, line)
	s.SaveHistory()
	if strings.HasPrefix(line, "set format ") {
		mode := strings.TrimSpace(strings.TrimPrefix(line, "set format "))
		if mode != "table" && mode != "json" { fmt.Println("Usage: set format table|json"); return true }
		s.OutputFormat = mode
		fmt.Println("Output format:", mode)
		return true
	}
	if strings.HasPrefix(line, "help ") {
        name := strings.TrimSpace(strings.TrimPrefix(line, "help "))
        for _, c := range s.Registry.ListFor(core.SurfaceCLI) {
            if c.Path == name || c.ID == name {
                fmt.Println(indentOutput(commandHelp(c)))
                return true
            }
        }
        fmt.Println("Unknown command:", name)
        return true
    }
	switch line {
	case "quit", "exit":
		return false
	case "help":
		fmt.Println("Built-in: help, commands, tools permissions, history, set format table|json, help <command>, report, report save, agent reset, exit")
		for _, c := range s.Registry.ListFor(core.SurfaceCLI) {
			fmt.Printf("  %-24s %s\n", c.Path, c.Description)
		}
		return true
	case "tools permissions":
        fmt.Println("TOOL ID                       CLI          AGENT        MCP")
        for _,c:=range s.Registry.List(){
            state:=func(surface core.Surface)string{ reason:=s.Registry.AccessReason(surface,c); if reason=="" {return "allowed"}; return reason }
            fmt.Printf("%-29s %-12s %-12s %s\n",c.ID,state(core.SurfaceCLI),state(core.SurfaceAgent),state(core.SurfaceMCP))
        }
        return true
	case "commands":
		for _, c := range s.Registry.ListFor(core.SurfaceCLI) {
			fmt.Println(c.Path)
		}
		return true
	case "agent reset":
		if resetter, ok := s.Agent.(interface{ Reset() }); ok {
			resetter.Reset()
		}
		fmt.Println("Agent context cleared")
		return true
	case "report":
		if rep, ok := s.Agent.(interface{ Report() string }); ok && rep.Report() != "" {
			fmt.Println(indentOutput(s.renderAgentAnswer(rep.Report())))
		} else {
			fmt.Println("No agent report yet")
		}
		return true
	case "report save":
		if rep, ok := s.Agent.(interface{ Report() string }); ok && rep.Report() != "" {
			path, err := saveReport(rep.Report(), s.CurrentContext)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
			} else {
				fmt.Println("Report saved:", path)
			}
		} else {
			fmt.Println("No agent report yet")
		}
		return true
	case "history":
		for i, h := range s.History {
			fmt.Printf("%d  %s\n", i+1, h)
		}
		return true
	}
	if c, args, ok := s.Registry.Resolve(line); ok {
		if !s.Registry.Allowed(core.SurfaceCLI, c) { fmt.Fprintln(os.Stderr, "error: tool not permitted in CLI"); return true }
		if c.ID == "k8s.context.use" {
			input, err := core.ParseInput(c, args)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				return true
			}
			target, _ := input["name"].(string)
			if target == "" {
				fmt.Fprintln(os.Stderr, "error: context name required")
				return true
			}
			current, err := s.Registry.Execute(ctx, "k8s.contexts", nil)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				return true
			}
			// Validation of existence happens only on confirmed execution.
			fmt.Println(indentOutput(s.renderResult("k8s.contexts", current)))
			fmt.Printf("\nSwitch Kubernetes context:\n  Current: %s\n  Target:  %s\n", currentContextName(current), target)
			s.PendingContext = target
			return true
		}

		input, err := core.ParseInput(c, args)
		if err == nil {
			var result any
			result, err = s.Registry.ExecuteFor(ctx, core.SurfaceCLI, c.ID, input)
			if err == nil {
				fmt.Println(indentOutput(s.renderResult(c.ID, result)))
				return true
			}
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		return true
	}

	if s.Agent != nil {
		if contextAware, ok := s.Agent.(interface{ SetContext(string) }); ok {
			contextAware.SetContext(s.CurrentContext)
		}
		toolCount := 0
		if traced, ok := s.Agent.(interface {
			SetTrace(func(string, map[string]any, time.Duration, error))
		}); ok {
			traced.SetTrace(func(id string, input map[string]any, d time.Duration, err error) {
				if toolCount == 0 { fmt.Println(traceHeader(colorsEnabled())) }
				toolCount++
				status := "✓"
				if err != nil {
					status = "✗"
				}
				fmt.Printf("    %s %s %s (%d ms)\n", status, id, compactInput(input), d.Milliseconds())
				if err != nil {
					fmt.Printf("      error: %v\n", err)
				}
			})
		}
		ans, err := s.Agent.Ask(ctx, line)
		if toolCount > 0 {
            if ctx.Err() != nil { fmt.Println("  └──────────────") } else { fmt.Println(traceFooter(colorsEnabled())) }
        }
		if err != nil {
            if errors.Is(err, context.Canceled) || ctx.Err() != nil {
                fmt.Println("  Cancelled (Ctrl+C)")
            } else {
                fmt.Fprintln(os.Stderr, "agent:", err)
            }
		} else {
			fmt.Println(indentOutput(s.renderAgentAnswer(ans)))
		}
	} else {
		fmt.Println("Unknown command (no agent configured). Type help.")
	}
	return true
}

 
// handleInterruptible enables terminal-generated SIGINT only while a command
// is executing. During line editing we deliberately keep ISIG disabled so
// Ctrl+C remains an editor key rather than killing the process.
func (s *Shell) handleInterruptible(parent context.Context, line string, raw bool) bool {
    ctx, cancel := context.WithCancel(parent)
    defer cancel()
    signals := make(chan os.Signal, 1)
    signal.Notify(signals, os.Interrupt)
    defer signal.Stop(signals)
    done := make(chan struct{})
    defer close(done)
    go func() {
        select {
        case <-signals:
            cancel()
        case <-done:
        }
    }()
    if raw {
        cmd := exec.Command("stty", "isig")
        cmd.Stdin = os.Stdin
        if err := cmd.Run(); err == nil {
            defer func() {
                restore := exec.Command("stty", "-isig")
                restore.Stdin = os.Stdin
                _ = restore.Run()
            }()
        }
    }
    return s.Handle(ctx, line)
}

const (
	reset  = "\x1b[0m"
	blue   = "\x1b[38;5;75m"
	green  = "\x1b[38;5;114m"
	muted  = "\x1b[38;5;245m"
	yellow = "\x1b[38;5;221m"
	pink   = "\x1b[38;5;218m"
)

func confirmationPrompt(color bool) string {
	if color {
		return pink + "Apply?" + reset + " " + muted + "[y/N]" + reset + " "
	}
	return "Apply? [y/N] "
}

func colorsEnabled() bool {
	return os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb" && os.Getenv("CLICOLOR") != "0"
}
func (s *Shell) prompt(color bool) string {
	name := "support"
	if s.CurrentContext != "" && s.CurrentContext != "(unknown)" {
		name += ":" + s.CurrentContext
	}
	if color {
		return blue + name + reset + " " + green + "❯" + reset + " "
	}
	return name + " ❯ "
}

func (s *Shell) refreshContext(ctx context.Context) {
	v, err := s.Registry.Execute(ctx, "k8s.contexts", nil)
	if err != nil {
		s.CurrentContext = ""
		return
	}
	s.CurrentContext = currentContextName(v)
}
func (s *Shell) Run(ctx context.Context) {
	fmt.Println("Support Shell — Tab completion · ↑↓ history · Ctrl+R search · help")
	s.refreshContext(ctx)
	if s.raw(ctx) {
		return
	}
	sc := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(s.prompt(false))
		if !sc.Scan() {
			break
		}
		if !s.handleInterruptible(ctx, sc.Text(), false) {
			break
		}
		if s.PendingContext != "" {
			fmt.Print(confirmationPrompt(colorsEnabled()))
			if !sc.Scan() {
				s.PendingContext = ""
				break
			}
			s.applyContextDecision(ctx, sc.Text())
		}
	}
}
func (s *Shell) raw(ctx context.Context) bool {
	fi, err := os.Stdin.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	stateCmd := exec.Command("stty", "-g")
	stateCmd.Stdin = os.Stdin
	state, err := stateCmd.Output()
	if err != nil {
		return false
	}
	rawCmd := exec.Command("stty", "-icanon", "-echo", "-isig", "-ixon", "min", "1", "time", "0", "opost", "onlcr")
	rawCmd.Stdin = os.Stdin
	if err = rawCmd.Run(); err != nil {
		return false
	}
	defer func() {
		restore := exec.Command("stty", strings.TrimSpace(string(state)))
		restore.Stdin = os.Stdin
		_ = restore.Run()
	}()
	reader := bufio.NewReader(os.Stdin)
	ed := newEditor(s.History)
	colored := colorsEnabled()
	// Repaint the line and position the cursor, counting Unicode runes, not ANSI bytes.
	redraw := func() {
		fmt.Print("\r\x1b[2K", s.prompt(colored))
		if colored {
			fmt.Print("\x1b[38;5;252m")
		}
		fmt.Print(string(ed.line))
		if colored {
			fmt.Print(reset)
		}
		if back := len(ed.line) - ed.cursor; back > 0 {
			fmt.Printf("\x1b[%dD", back)
		}
	}
	// Search mode uses the normal editor buffer as the draft.
	searching := false
	query := ""
	searchIndex := 0
	searchMatch := ""
	beforeSearch := ""
	searchRedraw := func() {
		fmt.Print("\r\x1b[2K")
		if colored {
			fmt.Print(yellow)
		}
		fmt.Print("(reverse-i-search) ")
		if colored {
			fmt.Print(reset)
		}
		fmt.Printf("%s: %s", query, searchMatch)
	}
	redraw()
	for {
		ch, err := reader.ReadByte()
		if err != nil {
			if err != io.EOF {
				fmt.Fprintln(os.Stderr, err)
			}
			fmt.Print("\r\n")
			return true
		}
		if searching {
			switch ch {
			case 3, 7, 27:
				searching = false
				ed.set(beforeSearch)
				redraw()
				continue
			case 13, 10:
				searching = false
				ed.set(searchMatch)
				redraw()
				continue
			case 18:
				if found, idx, ok := ed.searchBack(query, searchIndex-1); ok {
					searchMatch = found
					searchIndex = idx
				}
				searchRedraw()
				continue
			case 127, 8:
				if len(query) > 0 {
					query = query[:len(query)-1]
				}
			default:
				if ch >= 32 && ch < 127 {
					query += string(ch)
				} else {
					continue
				}
			}
			if found, idx, ok := ed.searchBack(query, len(ed.history)-1); ok {
				searchMatch = found
				searchIndex = idx
			} else {
				searchMatch = ""
				searchIndex = len(ed.history)
			}
			searchRedraw()
			continue
		}
		switch ch {
		case 3:
            // Ctrl+C while editing discards the draft, not the shell session.
            ed = newEditor(s.History)
            fmt.Print("\r\n")
            redraw()
        case 4:
            fmt.Print("\r\n")
            return true
		case 13, 10:
			fmt.Print("\r\n")
			if !s.handleInterruptible(ctx, string(ed.line), true) {
				return true
			}
			if s.PendingContext != "" {
				fmt.Print(confirmationPrompt(colorsEnabled()))
				decision, err := readConfirmation(reader)
				if err != nil {
					s.PendingContext = ""
					fmt.Print("\r\n")
					return true
				}
				fmt.Print("\r\n")
				s.applyContextDecision(ctx, decision)
			}
			ed = newEditor(s.History)
			redraw()
		case 127, 8:
			ed.backspace()
			redraw()
		case 1:
			ed.cursor = 0
			redraw() // Ctrl+A
		case 5:
			ed.cursor = len(ed.line)
			redraw() // Ctrl+E
		case 18:
			searching = true
			query = ""
			beforeSearch = string(ed.line)
			searchIndex = len(ed.history)
			searchMatch = ""
			searchRedraw()
		case 9:
			if ed.cursor != len(ed.line) {
				break
			}
			matches := completionMatches(string(ed.line), s.completions(ctx, string(ed.line)))
			if len(matches) == 1 {
				ed.set(matches[0])
				if _, _, ok := s.Registry.Resolve(matches[0]); ok {
					ed.insert(' ')
				}
			} else if len(matches) > 1 {
				common := longestCommonPrefix(matches)
				if len(common) > len(string(ed.line)) {
					ed.set(common)
					redraw()
					break
				}
				if selection, ok := selectCompletion(reader, os.Stdout, matches, colored); ok {
					ed.set(selection)
					if _, _, found := s.Registry.Resolve(selection); found {
						ed.insert(' ')
					}
				}

			}
			redraw()
		case 27:
			prefix, e := reader.ReadByte()
			if e != nil {
				continue
			}
			if prefix != '[' && prefix != 'O' {
				continue
			}
			key, e := reader.ReadByte()
			if e != nil {
				continue
			}
			switch key {
			case 'A':
				ed.previous()
			case 'B':
				ed.next()
			case 'C':
				if ed.cursor < len(ed.line) {
					ed.cursor++
				}
			case 'D':
				if ed.cursor > 0 {
					ed.cursor--
				}
			case 'H':
				ed.cursor = 0
			case 'F':
				ed.cursor = len(ed.line)
			case '3':
				next, e := reader.ReadByte()
				if e == nil && next == '~' {
					ed.delete()
				}
			}
			redraw()
		default:
			if ch >= 32 && ch < 127 {
				ed.insert(rune(ch))
				redraw()
			} else if ch >= 128 {
				seq := []byte{ch}
				n := 0
				switch {
				case ch&0xe0 == 0xc0:
					n = 1
				case ch&0xf0 == 0xe0:
					n = 2
				case ch&0xf8 == 0xf0:
					n = 3
				}
				for i := 0; i < n; i++ {
					b, e := reader.ReadByte()
					if e != nil {
						break
					}
					seq = append(seq, b)
				}
				if utf8.Valid(seq) {
					for _, r := range string(seq) {
						ed.insert(r)
					}
				}
				redraw()
			}
		}
	}
}

func (s *Shell) historyFile() string {
	if s.HistoryPath != "" {
		return s.HistoryPath
	}
	dir, e := os.UserConfigDir()
	if e != nil {
		return ""
	}
	return filepath.Join(dir, "support-shell", "history")
}
func (s *Shell) LoadHistory() {
	path := s.historyFile()
	if path == "" {
		return
	}
	data, e := os.ReadFile(path)
	if e != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		if line != "" {
			s.History = append(s.History, line)
		}
	}
	if len(s.History) > 500 {
		s.History = s.History[len(s.History)-500:]
	}
}
func (s *Shell) SaveHistory() {
	path := s.historyFile()
	if path == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0700)
	lines := s.History
	if len(lines) > 500 {
		lines = lines[len(lines)-500:]
	}
	_ = os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0600)
}

// formatResult prints plain-text command output without JSON string escaping.
// Structured results remain pretty-printed JSON for inspection and agents.
func formatResult(result any) string {
	switch v := result.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return core.JSON(result)
	}
}

// longestCommonPrefix finds the maximal shared UTF-8 prefix of completion candidates.
func longestCommonPrefix(values []string) string {
	if len(values) == 0 {
		return ""
	}
	prefix := []rune(values[0])
	for _, value := range values[1:] {
		other := []rune(value)
		n := len(prefix)
		if len(other) < n {
			n = len(other)
		}
		i := 0
		for i < n && prefix[i] == other[i] {
			i++
		}
		prefix = prefix[:i]
		if len(prefix) == 0 {
			break
		}
	}
	return string(prefix)
}

// readConfirmation reads just one answer and never records it in shell history.
// Only a single y/Y is accepted; Enter, n and all other responses deny.
func readConfirmation(r *bufio.Reader) (string, error) {
	var answer []byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		switch b {
		case 3, 4:
			return "", nil
		case 13, 10:
			return string(answer), nil
		case 127, 8:
			if len(answer) > 0 {
				answer = answer[:len(answer)-1]
				fmt.Print("\b \b")
			}
		default:
			if b >= 32 && b < 127 && len(answer) < 16 {
				answer = append(answer, b)
				fmt.Printf("%c", b)
			}
		}
	}
}

func (s *Shell) applyContextDecision(ctx context.Context, decision string) {
	target := s.PendingContext
	s.PendingContext = ""
	if target == "" {
		return
	}
	if strings.TrimSpace(decision) != "y" && strings.TrimSpace(decision) != "Y" {
		fmt.Println("Context switch cancelled")
		return
	}
	result, err := s.Registry.ExecuteApproved(ctx, "k8s.context.use", map[string]any{"name": target})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return
	}
	fmt.Println(indentOutput(s.renderResult("k8s.context.use", result)))
	s.refreshContext(ctx)
}

func currentContextName(value any) string {
	// context list is formatted for humans; the active row is marked with *.
	text, ok := value.(string)
	if !ok {
		return "(unknown)"
	}
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 1 && fields[0] == "*" {
			return fields[1]
		}
	}
	return "(unknown)"
}

// compactInput avoids multi-line JSON in live tool tracing.
func compactInput(in map[string]any) string {
	if len(in) == 0 {
		return ""
	}
	parts := make([]string, 0, len(in))
	for _, k := range []string{"namespace", "pod", "container", "tail", "since", "name"} {
		if v, ok := in[k]; ok {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
	}
	if len(parts) > 0 {
		return "[" + strings.Join(parts, " ") + "]"
	}
	b, _ := json.Marshal(in)
	return string(b)
}

func saveReport(report, contextName string) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "support-shell", "reports")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	label := regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(contextName, "_")
	if label == "" {
		label = "unknown"
	}
	path := filepath.Join(dir, fmt.Sprintf("%s-%s.md", time.Now().Format("20060102-150405.000000000"), label))
	contents := fmt.Sprintf("# Support Shell diagnostic report\n\nContext: `%s`\nGenerated: %s\n\n%s\n", contextName, time.Now().Format(time.RFC3339), report)
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		return "", err
	}
	return path, nil
}

// renderAgentAnswer retains the model response as text in table mode and
// wraps it as a JSON string field when structured output is requested.
func (s *Shell) renderAgentAnswer(answer string) string {
 if s.OutputFormat == "json" { return core.JSON(map[string]any{"answer":answer}) }
 return renderAgentMarkdown(answer)
}
