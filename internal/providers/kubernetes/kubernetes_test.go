package kubernetes

import (
	"context"
	"example.com/support-shell/internal/core"
	"example.com/support-shell/internal/provider"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKubectlTextAndErrors(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "kubectl")
	script := `#!/bin/sh
case "$*" in
 "config view -o=json") echo '{"current-context":"dev","contexts":[{"name":"dev","context":{"cluster":"dev-cluster"}}]}' ;;
 *) printf '{"items":[]}\\n'; printf 'Error from server: nodes forbidden\\n' >&2; exit 1;;
esac
`
	if err := os.WriteFile(fake, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	r := core.NewRegistry()
	registerForTest(t, r)
	v, err := r.Execute(provider.WithConnection(context.Background(),"kubernetes","dev"), "provider.kubernetes.contexts", nil)
	if err != nil || !strings.Contains(v.(string), "dev-cluster") {
		t.Fatalf("contexts: %v %v", v, err)
	}
	_, err = r.Execute(provider.WithConnection(context.Background(),"kubernetes","dev"), "provider.kubernetes.nodes", nil)
	if err == nil || !strings.Contains(err.Error(), "nodes forbidden") || strings.Contains(err.Error(), `"items"`) {
		t.Fatalf("error details not isolated: %v", err)
	}
}

func TestSessionContextSelectionWithoutKubeconfigMutation(t *testing.T) {
 dir:=t.TempDir()
 script:=`#!/bin/sh
case "$*" in
 "config view -o=json") echo '{"current-context":"dev","contexts":[{"name":"dev","context":{"cluster":"dev"}},{"name":"prod","context":{"cluster":"prod"}}]}' ;;
 "--context prod get nodes -o json") echo '{"items":[{"metadata":{"name":"prod-node"}}]}' ;;
 "--context dev get nodes -o json") echo '{"items":[{"metadata":{"name":"dev-node"}}]}' ;;
 *) echo "unexpected $*" >&2; exit 12 ;;
esac
`
 if err:=os.WriteFile(filepath.Join(dir,"kubectl"),[]byte(script),0700);err!=nil{t.Fatal(err)}
 t.Setenv("PATH",dir+string(os.PathListSeparator)+os.Getenv("PATH"))
 r:=core.NewRegistry()
 registerForTest(t,r)
 providers:=provider.NewRegistry()
 if err:=providers.Register(New());err!=nil{t.Fatal(err)}
 session:=provider.NewSession()
 if err:=session.SelectDefaults(context.Background(),providers);err!=nil{t.Fatal(err)}
 if got:=session.Current("kubernetes");got!="dev"{t.Fatalf("initial context=%q",got)}
 if err:=session.Select(context.Background(),providers,"kubernetes","prod");err!=nil{t.Fatal(err)}
 if got:=session.Current("kubernetes");got!="prod"{t.Fatalf("selected context=%q",got)}
 result,err:=r.Execute(session.Bind(context.Background()),"provider.kubernetes.nodes",nil)
 if err!=nil||!strings.Contains(core.JSON(result),"prod-node"){t.Fatalf("result=%v err=%v",result,err)}
 other:=provider.NewSession()
 if err:=other.SelectDefaults(context.Background(),providers);err!=nil{t.Fatal(err)}
 result,err=r.Execute(other.Bind(context.Background()),"provider.kubernetes.nodes",nil)
 if err!=nil||!strings.Contains(core.JSON(result),"dev-node"){t.Fatalf("other session=%v err=%v",result,err)}
 raw,err:=os.ReadFile(filepath.Join(dir,"kubectl"))
 if err!=nil{t.Fatal(err)}
 if strings.Contains(string(raw),"use-context"){t.Fatal("fake expects no kubeconfig mutation")}
}

func TestKubernetesOperationsRequireSelectedConnection(t *testing.T){
 r:=core.NewRegistry()
 registerForTest(t,r)
 if _,err:=r.Execute(context.Background(),"provider.kubernetes.nodes",nil);err==nil || !strings.Contains(err.Error(),"no active Kubernetes connection"){
  t.Fatalf("operation was not rejected without connection: %v",err)
 }
}
