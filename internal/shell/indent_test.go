package shell

import "testing"

func TestIndentOutput(t *testing.T) {
 cases:=[]struct{input,want string}{
  {"",""},
  {"hello","  hello"},
  {"NAME  STATUS\nnode1 Ready","  NAME  STATUS\n  node1 Ready"},
  {"{\n  \"ok\": true\n}","  {\n    \"ok\": true\n  }"},
 }
 for _,tc:=range cases { if got:=indentOutput(tc.input);got!=tc.want {t.Errorf("indentOutput(%q) = %q, want %q",tc.input,got,tc.want)} }
}
