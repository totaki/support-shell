package main

import (
 "os"
 "path/filepath"
 "strings"
 "testing"
)
func TestLoadDotEnv(t *testing.T){
 dir:=t.TempDir()
 path:=filepath.Join(dir,".env")
 data:="\n# ignored\nexport SUPPORT_TEST_A=loaded\nSUPPORT_TEST_B='hello world'\nSUPPORT_TEST_C=plain # comment\nSUPPORT_TEST_D=\"double quoted\"\nSUPPORT_TEST_EXISTING=from_file\n"
 if err:=os.WriteFile(path,[]byte(data),0600);err!=nil{t.Fatal(err)}
 for _,key:=range []string{"SUPPORT_TEST_A","SUPPORT_TEST_B","SUPPORT_TEST_C","SUPPORT_TEST_D"}{t.Setenv(key,"");os.Unsetenv(key)}
 t.Setenv("SUPPORT_TEST_EXISTING","from_shell")
 if err:=loadDotEnv(path);err!=nil{t.Fatal(err)}
 for key,want:=range map[string]string{"SUPPORT_TEST_A":"loaded","SUPPORT_TEST_B":"hello world","SUPPORT_TEST_C":"plain","SUPPORT_TEST_D":"double quoted","SUPPORT_TEST_EXISTING":"from_shell"}{
  if got:=os.Getenv(key);got!=want{t.Errorf("%s=%q want %q",key,got,want)}
 }
 for _,key:=range []string{"SUPPORT_TEST_A","SUPPORT_TEST_B","SUPPORT_TEST_C","SUPPORT_TEST_D"}{os.Unsetenv(key)}
}
func TestLoadDotEnvMissingAndInvalid(t *testing.T){
 dir:=t.TempDir()
 if err:=loadDotEnv(filepath.Join(dir,"missing"));err!=nil{t.Fatal(err)}
 path:=filepath.Join(dir,".env")
 for _,value:=range []string{"INVALID-NAME=value","NOT_AN_ASSIGNMENT","KEY=\"unterminated"}{
  if err:=os.WriteFile(path,[]byte(value+"\n"),0600);err!=nil{t.Fatal(err)}
  if err:=loadDotEnv(path);err==nil||!strings.Contains(err.Error(),".env line 1"){t.Fatalf("input %q err=%v",value,err)}
 }
}
