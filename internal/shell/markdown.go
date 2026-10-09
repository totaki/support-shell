package shell

import (
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
)

// renderAgentMarkdown renders common Markdown constructs as readable terminal text.
// The original Markdown stays unchanged in `report save`.
// NO_COLOR and non-TTY output disable ANSI decoration.
func renderAgentMarkdown(source string) string {
	color := os.Getenv("NO_COLOR") == "" && stdoutTTY()
	return renderMarkdown(source, color)
}
func stdoutTTY() bool { s, e := os.Stdout.Stat(); return e == nil && s.Mode()&os.ModeCharDevice != 0 }

const (
	mdReset  = "\x1b[0m"
	mdBold   = "\x1b[1m"
	mdDim    = "\x1b[38;5;245m"
	mdTitle  = "\x1b[1;38;5;117m"
	mdSecond = "\x1b[1;38;5;181m"
	mdCode   = "\x1b[38;5;151m"
	mdRule   = "\x1b[38;5;240m"
)

var mdInline = regexp.MustCompile("\\[([^\\]]+)\\]\\((https?://[^ )]+)\\)|`([^`]+)`|\\*\\*([^*]+)\\*\\*|\\*([^*\\n]+)\\*")
var mdList = regexp.MustCompile(`^(\s*)([-*+] |[0-9]+\. )(.*)$`)

func paint(s, style string, on bool) string {
	if !on {
		return s
	}
	return style + s + mdReset
}
func inlineRender(s string, color bool) string {
	return mdInline.ReplaceAllStringFunc(s, func(m string) string {
		v := mdInline.FindStringSubmatch(m)
		switch {
		case v[1] != "":
			return v[1] + " (" + v[2] + ")"
		case v[3] != "":
			return paint(v[3], mdCode, color)
		case v[4] != "":
			return paint(v[4], mdBold, color)
		default:
			return v[5]
		}
	})
}
func tableCells(s string) []string {
	s = strings.TrimSpace(strings.Trim(s, "|"))
	items := strings.Split(s, "|")
	for i := range items {
		items[i] = strings.TrimSpace(items[i])
	}
	return items
}
func isSeparator(s string) bool {
	parts := tableCells(s)
	if len(parts) == 0 {
		return false
	}
	for _, p := range parts {
		p = strings.TrimSpace(strings.ReplaceAll(p, ":", ""))
		if len(p) < 3 || strings.Trim(p, "-") != "" {
			return false
		}
	}
	return true
}
func runeWidth(s string) int { return utf8.RuneCountInString(s) }
func renderMarkdown(raw string, color bool) string {
	raw = strings.ReplaceAll(strings.ReplaceAll(raw, "\r\n", "\n"), "\r", "")
	// Prevent remote model output from injecting terminal control sequences.
	raw = strings.Map(func(r rune) rune {
		if r == 0x1b || r == 0x7f || (r < 32 && r != '\n' && r != '\t') {
			return -1
		}
		return r
	}, raw)
	lines := strings.Split(strings.TrimSuffix(raw, "\n"), "\n")
	var out []string
	code := false
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "```") {
			code = !code
			if code {
				lang := strings.TrimSpace(strings.TrimPrefix(trim, "```"))
				if lang != "" {
					out = append(out, paint("  "+lang, mdDim, color))
				}
			}
			continue
		}
		if code {
			out = append(out, paint("  "+line, mdCode, color))
			continue
		}
		if strings.HasPrefix(trim, "|") && i+1 < len(lines) && isSeparator(lines[i+1]) {
			header := tableCells(line)
			i += 2
			rows := [][]string{header}
			for ; i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "|"); i++ {
				rows = append(rows, tableCells(lines[i]))
			}
			i--
			widths := make([]int, len(header))
			for _, row := range rows {
				for j := 0; j < len(widths) && j < len(row); j++ {
					if n := runeWidth(row[j]); n > widths[j] {
						widths[j] = n
					}
				}
			}
			for rid, row := range rows {
				parts := make([]string, len(widths))
				for j := range widths {
					v := ""
					if j < len(row) {
						v = row[j]
					}
					parts[j] = v + strings.Repeat(" ", widths[j]-runeWidth(v))
				}
				v := "  " + strings.Join(parts, "  ")
				if rid == 0 {
					v = paint(v, mdBold, color)
				}
				out = append(out, v)
				if rid == 0 {
					out = append(out, paint("  "+strings.Repeat("─", 2+sumWidths(widths)), mdRule, color))
				}
			}
			continue
		}
		switch {
		case strings.HasPrefix(trim, "### "):
			out = append(out, paint("  "+inlineRender(strings.TrimPrefix(trim, "### "), false), mdSecond, color))
		case strings.HasPrefix(trim, "## "):
			out = append(out, "", paint(inlineRender(strings.TrimPrefix(trim, "## "), false), mdTitle, color))
		case strings.HasPrefix(trim, "# "):
			out = append(out, "", paint(inlineRender(strings.TrimPrefix(trim, "# "), false), mdTitle, color))
		case trim == "---" || trim == "***":
			out = append(out, paint(strings.Repeat("─", 50), mdRule, color))
		case strings.HasPrefix(trim, "> "):
			out = append(out, paint("│ ", mdDim, color)+inlineRender(strings.TrimPrefix(trim, "> "), color))
		case mdList.MatchString(line):
			m := mdList.FindStringSubmatch(line)
			bullet := "• "
			if strings.HasSuffix(m[2], ". ") {
				bullet = m[2]
			}
			out = append(out, m[1]+paint(bullet, mdSecond, color)+inlineRender(m[3], color))
		default:
			out = append(out, inlineRender(line, color))
		}
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n")
}
func sumWidths(v []int) int {
	n := 0
	for _, x := range v {
		n += x + 2
	}
	return n
}
