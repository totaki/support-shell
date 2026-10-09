package shell

import "strings"

// indentOutput shifts a completed command result to the right while preserving
// relative alignment of tables, Markdown, and multi-line JSON.
func indentOutput(value string) string {
 if value == "" { return "" }
 return "  " + strings.ReplaceAll(value, "\n", "\n  ")
}
