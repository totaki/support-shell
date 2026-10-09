package shell

import (
	"strings"
)

// editor tracks the current line independently of terminal rendering.
type editor struct {
	line    []rune
	cursor  int
	history []string
	index   int
	draft   string
}

func newEditor(history []string) *editor { return &editor{history: history, index: len(history)} }
func (e *editor) set(s string)           { e.line = []rune(s); e.cursor = len(e.line) }
func (e *editor) insert(r rune) {
	e.line = append(e.line, 0)
	copy(e.line[e.cursor+1:], e.line[e.cursor:])
	e.line[e.cursor] = r
	e.cursor++
}
func (e *editor) backspace() {
	if e.cursor > 0 {
		e.line = append(e.line[:e.cursor-1], e.line[e.cursor:]...)
		e.cursor--
	}
}
func (e *editor) delete() {
	if e.cursor < len(e.line) {
		e.line = append(e.line[:e.cursor], e.line[e.cursor+1:]...)
	}
}
func (e *editor) previous() {
	if e.index == len(e.history) {
		e.draft = string(e.line)
	}
	if e.index > 0 {
		e.index--
		e.set(e.history[e.index])
	}
}
func (e *editor) next() {
	if e.index < len(e.history)-1 {
		e.index++
		e.set(e.history[e.index])
	} else if e.index < len(e.history) {
		e.index = len(e.history)
		e.set(e.draft)
	}
}
func (e *editor) searchBack(query string, from int) (string, int, bool) {
	for i := from; i >= 0; i-- {
		if strings.Contains(e.history[i], query) {
			return e.history[i], i, true
		}
	}
	return "", 0, false
}
