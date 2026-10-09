package shell

import "testing"

func TestHistoryAndDraft(t *testing.T) {
	e := newEditor([]string{"one", "two"})
	e.set("dra")
	e.previous()
	if string(e.line) != "two" {
		t.Fatal(string(e.line))
	}
	e.previous()
	e.next()
	e.next()
	if string(e.line) != "dra" {
		t.Fatal(string(e.line))
	}
}
func TestCursorEditing(t *testing.T) {
	e := newEditor(nil)
	e.set("ac")
	e.cursor = 1
	e.insert('b')
	if string(e.line) != "abc" {
		t.Fatal(string(e.line))
	}
	e.backspace()
	if string(e.line) != "ac" {
		t.Fatal(string(e.line))
	}
	e.delete()
	if string(e.line) != "a" {
		t.Fatal(string(e.line))
	}
}
func TestReverseSearch(t *testing.T) {
	e := newEditor([]string{"clusters", "cluster health prod", "pods"})
	v, i, ok := e.searchBack("cluster", 2)
	if !ok || v != "cluster health prod" || i != 1 {
		t.Fatal(v, i, ok)
	}
}

func TestLongestCommonPrefix(t *testing.T) {
	tests := []struct {
		values []string
		want   string
	}{
		{[]string{"cluster", "clusters"}, "cluster"},
		{[]string{"k8s pod logs", "k8s pod describe"}, "k8s pod "},
		{[]string{"clusters"}, "clusters"},
		{nil, ""},
		{[]string{"ёжик", "ёж"}, "ёж"},
	}
	for _, tc := range tests {
		if got := longestCommonPrefix(tc.values); got != tc.want {
			t.Errorf("longestCommonPrefix(%q) = %q, want %q", tc.values, got, tc.want)
		}
	}
}
