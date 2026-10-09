package shell

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// selectCompletion draws a temporary in-terminal menu. Tab/down cycles, up
// moves backwards, Enter accepts and Escape cancels. It never executes commands.
func selectCompletion(reader *bufio.Reader, out io.Writer, choices []string, colored bool) (string, bool) {
	if len(choices) == 0 {
		return "", false
	}
	selected := 0
	const maxVisible = 10
	start := 0
	menuHeight := len(choices)
	if menuHeight > maxVisible {
		menuHeight = maxVisible
	}
	draw := func() {
		visible := menuHeight
		fmt.Fprint(out, "\r\n")
		for i := 0; i < visible; i++ {
			choice := choices[start+i]
			if start+i == selected {
				if colored {
					fmt.Fprint(out, "\x1b[38;5;218m")
				}
				fmt.Fprintf(out, "  ❯ %s", choice)
				if colored {
					fmt.Fprint(out, "\x1b[0m")
				}
			} else {
				fmt.Fprintf(out, "    %s", choice)
			}
			fmt.Fprint(out, "\x1b[K\r\n")
		}
		fmt.Fprint(out, "  ↑↓ select · Tab next · Enter accept · Esc cancel\x1b[K")
	}
	// Repaint only menu rows, leaving the current input intact.
	repaint := func() {
		visible := menuHeight
		fmt.Fprintf(out, "\r\x1b[%dA", visible+1)
		for i := 0; i < visible; i++ {
			fmt.Fprint(out, "\r\x1b[2K")
			idx := start + i
			if idx == selected {
				if colored {
					fmt.Fprint(out, "\x1b[38;5;218m")
				}
				fmt.Fprint(out, "  ❯ ", choices[idx])
				if colored {
					fmt.Fprint(out, "\x1b[0m")
				}
			} else {
				fmt.Fprint(out, "    ", choices[idx])
			}
			fmt.Fprint(out, "\r\n")
		}
		fmt.Fprint(out, "\r\x1b[2K  ↑↓ select · Tab next · Enter accept · Esc cancel")
	}
	cleanup := func() {
		visible := menuHeight
		fmt.Fprintf(out, "\r\x1b[%dA", visible+1)
		for i := 0; i < visible+1; i++ {
			fmt.Fprint(out, "\r\x1b[2K")
			if i < visible {
				fmt.Fprint(out, "\x1b[1B")
			}
		}
		fmt.Fprintf(out, "\r\x1b[%dA", visible)
	}
	draw()
	for {
		ch, e := reader.ReadByte()
		if e != nil {
			cleanup()
			return "", false
		}
		switch ch {
		case 13, 10:
			cleanup()
			return choices[selected], true
		case 27:
			// Escape alone cancels, CSI arrows navigate.
			if reader.Buffered() >= 2 {
				b, _ := reader.Peek(2)
				if b[0] == '[' || b[0] == 'O' {
					_, _ = reader.ReadByte()
					key, _ := reader.ReadByte()
					if key == 'A' {
						selected = (selected + len(choices) - 1) % len(choices)
					} else if key == 'B' {
						selected = (selected + 1) % len(choices)
					} else {
						cleanup()
						return "", false
					}
				} else {
					cleanup()
					return "", false
				}
			} else {
				cleanup()
				return "", false
			}
		case 9:
			selected = (selected + 1) % len(choices)
		case 3, 7:
			cleanup()
			return "", false
		default:
			cleanup()
			return "", false
		}
		if selected < start {
			start = selected
		}
		if selected >= start+maxVisible {
			start = selected - maxVisible + 1
		}
		// Repaint the currently allocated area; preserve its height on scrolling.
		repaint()
	}
}

func completionMatches(input string, matches []string) []string {
	out := make([]string, 0, len(matches))
	for _, s := range matches {
		if strings.HasPrefix(s, input) {
			out = append(out, s)
		}
	}
	return out
}
