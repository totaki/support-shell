package main

import (
 "bufio"
 "fmt"
 "os"
 "strconv"
 "strings"
)

// loadDotEnv imports local development settings without overwriting real environment variables.
// It is intentionally limited to KEY=value, optional export, and quoted values.
func loadDotEnv(path string) error {
 f,err:=os.Open(path)
 if os.IsNotExist(err){return nil}
 if err!=nil{return fmt.Errorf("open .env: %w",err)}
 defer f.Close()
 scan:=bufio.NewScanner(f)
 line:=0
 for scan.Scan(){
  line++
  s:=strings.TrimSpace(scan.Text())
  if s==""||strings.HasPrefix(s,"#"){continue}
  s=strings.TrimPrefix(s,"export ")
  key,value,ok:=strings.Cut(s,"=")
  key=strings.TrimSpace(key)
  if !ok||!validEnvName(key){return fmt.Errorf(".env line %d: invalid variable declaration",line)}
  value=strings.TrimSpace(value)
  if strings.HasPrefix(value,"\"")||strings.HasPrefix(value,"'"){
   quote:=value[0]
   if len(value)<2||value[len(value)-1]!=quote {return fmt.Errorf(".env line %d: unclosed quotation",line)}
   if quote=='"'{
    parsed,e:=strconv.Unquote(value)
    if e!=nil{return fmt.Errorf(".env line %d: %w",line,e)}
    value=parsed
   }else{value=value[1:len(value)-1]}
  }else{
   if cut,_,found:=strings.Cut(value," #");found{value=strings.TrimSpace(cut)}
  }
  if _,exists:=os.LookupEnv(key);!exists{
   if err:=os.Setenv(key,value);err!=nil{return fmt.Errorf(".env line %d: %w",line,err)}
  }
 }
 return scan.Err()
}
func validEnvName(s string)bool{
 if s==""{return false}
 for i,r:=range s {
  if r=='_'||(r>='A'&&r<='Z')||(r>='a'&&r<='z'){continue}
  if i>0&&r>='0'&&r<='9'{continue}
  return false
 }
 return true
}
