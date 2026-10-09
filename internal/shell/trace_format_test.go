package shell

import (
 "strings"
 "testing"
)
func TestTraceBoundaries(t *testing.T) {
 for _,color:=range []bool{false,true}{
  if !strings.Contains(traceHeader(color),"Tools"){t.Fatal("missing tools header")}
  footer:=traceFooter(color)
  if !strings.Contains(footer,"Answer")||!strings.Contains(footer,"└"){t.Fatal("missing boundary")}
 }
}
