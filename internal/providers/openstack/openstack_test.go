package openstack

import (
 "context"
 "os"
 "path/filepath"
 "strings"
 "testing"

 "example.com/support-shell/internal/core"
 "example.com/support-shell/internal/provider"
)

func writeCloudConfig(t *testing.T, contents string) string {
 t.Helper()
 path:=filepath.Join(t.TempDir(),"clouds.yaml")
 if err:=os.WriteFile(path,[]byte(contents),0600);err!=nil{t.Fatal(err)}
 t.Setenv("OS_CLIENT_CONFIG_FILE",path)
 return path
}

func fakeOpenStack(t *testing.T, body string) (string,string) {
 t.Helper()
 dir:=t.TempDir()
 script:=filepath.Join(dir,"openstack")
 if err:=os.WriteFile(script,[]byte("#!/bin/sh\n"+body+"\n"),0700);err!=nil{t.Fatal(err)}
 t.Setenv("PATH",dir+string(os.PathListSeparator)+os.Getenv("PATH"))
 args:=filepath.Join(dir,"args")
 auth:=filepath.Join(dir,"auth")
 t.Setenv("SUPPORT_TEST_OPENSTACK_ARGS",args)
 t.Setenv("SUPPORT_TEST_OPENSTACK_AUTH",auth)
 return args,auth
}

func TestCloudSelectionAndSessionIsolation(t *testing.T) {
 writeCloudConfig(t,"clouds:\n  development:\n    auth:\n      auth_url: https://example.invalid/v3\n  production:\n    auth:\n      auth_url: https://example.invalid/v3\n")
 t.Setenv("OS_CLOUD","development")
 t.Setenv("OS_AUTH_URL","https://ambient-must-not-leak.invalid")
 args,auth:=fakeOpenStack(t,"printf '%s\\n' \"$*\" > \"$SUPPORT_TEST_OPENSTACK_ARGS\"\nprintf '%s\\n' \"$OS_AUTH_URL\" > \"$SUPPORT_TEST_OPENSTACK_AUTH\"\nprintf '[{\"ID\":\"vm-01\",\"Name\":\"node-a\"}]\\n'\n")
 providers:=provider.NewRegistry()
 if err:=providers.Register(New());err!=nil{t.Fatal(err)}
 one,two:=provider.NewSession(),provider.NewSession()
 for _,session:=range []*provider.Session{one,two}{
  if err:=session.SelectDefaults(context.Background(),providers);err!=nil{t.Fatal(err)}
 }
 if got:=one.Current("openstack");got!="development"{t.Fatalf("default=%q",got)}
 if err:=one.Select(context.Background(),providers,"openstack","production");err!=nil{t.Fatal(err)}
 if got:=two.Current("openstack");got!="development"{t.Fatalf("other session changed to %q",got)}
 commands:=core.NewRegistry()
 if err:=providers.BindTools(commands);err!=nil{t.Fatal(err)}
 value,err:=commands.Execute(one.Bind(context.Background()),"provider.openstack.servers",nil)
 if err!=nil{t.Fatal(err)}
 if !strings.Contains(core.JSON(value),"vm-01"){t.Fatalf("unexpected result: %v",value)}
 raw,err:=os.ReadFile(args);if err!=nil{t.Fatal(err)}
 if got:=strings.TrimSpace(string(raw));got!="--os-cloud production server list -f json"{t.Fatalf("argv=%q",got)}
 raw,err=os.ReadFile(auth);if err!=nil{t.Fatal(err)}
 if strings.TrimSpace(string(raw))!=""{t.Fatalf("ambient OS_AUTH_URL leaked into named cloud: %q",raw)}
 _,err=commands.Execute(two.Bind(context.Background()),"provider.openstack.networks",nil)
 if err!=nil{t.Fatal(err)}
 raw,err=os.ReadFile(args);if err!=nil{t.Fatal(err)}
 if got:=strings.TrimSpace(string(raw));got!="--os-cloud development network list -f json"{t.Fatalf("other argv=%q",got)}
}

func TestMissingAndStaleConnectionRejected(t *testing.T){
 path:=writeCloudConfig(t,"clouds:\n  production: {}\n")
 t.Setenv("OS_CLOUD","")
 t.Setenv("OS_AUTH_URL","")
 ctx:=context.Background()
 providers:=provider.NewRegistry()
 if err:=providers.Register(New());err!=nil{t.Fatal(err)}
 session:=provider.NewSession()
 if err:=session.SelectDefaults(ctx,providers);err!=nil{t.Fatal(err)}
 if selected:=session.Current("openstack");selected!=""{t.Fatalf("unexpected default %q",selected)}
 if _,err:=providers.ExecuteRead(session.Bind(ctx),"provider.openstack.servers",nil);err==nil||!strings.Contains(err.Error(),"no active OpenStack connection"){t.Fatalf("unbound error=%v",err)}
 if err:=session.Select(ctx,providers,"openstack","production");err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(path,[]byte("clouds:\n  development: {}\n"),0600);err!=nil{t.Fatal(err)}
 if _,err:=providers.ExecuteRead(session.Bind(ctx),"provider.openstack.servers",nil);err==nil||!strings.Contains(err.Error(),"no longer available"){t.Fatalf("missing selected cloud: %v",err)}
}

func TestEnvironmentConnectionAndRedactedErrors(t *testing.T){
 writeCloudConfig(t,"clouds: {}\n")
 t.Setenv("OS_CLOUD","")
 t.Setenv("OS_AUTH_URL","https://example.invalid/v3")
 args,_:=fakeOpenStack(t,"printf '%s\\n' \"$*\" > \"$SUPPORT_TEST_OPENSTACK_ARGS\"\necho 'password=super-secret-value' >&2\nexit 5\n")
 providers:=provider.NewRegistry()
 if err:=providers.Register(New());err!=nil{t.Fatal(err)}
 session:=provider.NewSession()
 if err:=session.SelectDefaults(context.Background(),providers);err!=nil{t.Fatal(err)}
 if got:=session.Current("openstack");got!="env"{t.Fatalf("environment profile not selected: %q",got)}
 _,err:=providers.ExecuteRead(session.Bind(context.Background()),"provider.openstack.ports",nil)
 if err==nil{t.Fatal("expected command error")}
 if strings.Contains(err.Error(),"super-secret-value"){t.Fatal("sensitive stderr leaked")}
 raw,err:=os.ReadFile(args);if err!=nil{t.Fatal(err)}
 if got:=strings.TrimSpace(string(raw));got!="port list -f json"{t.Fatalf("environment argv=%q",got)}
}

func TestMalformedCloudsConfigDoesNotExposeCredentials(t *testing.T){
 writeCloudConfig(t,"clouds:\n  prod: [secret-password:abc\n")
 _,err:=discoverConnections(context.Background())
 if err==nil{t.Fatal("expected parse error")}
 if strings.Contains(err.Error(),"secret-password"){t.Fatal("credentials appeared in YAML parse error")}
}
