// Package openstack implements read-only infrastructure operations using
// python-openstackclient. Credentials remain in clouds.yaml or OS_* environment.
package openstack

import (
    "bytes"
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "sort"
    "strings"
    "time"

    "example.com/support-shell/internal/provider"
    "gopkg.in/yaml.v3"
)

const providerID = "openstack"

func New() *provider.Catalog {
    catalog := provider.NewCatalog(providerID)
    catalog.SetConnectionDiscovery(discoverConnections)

    for _,spec := range []struct {
        id, path, description string
        args []string
    }{
        {"servers", "openstack servers", "List compute servers", []string{"server", "list"}},
        {"networks", "openstack networks", "List Neutron networks", []string{"network", "list"}},
        {"subnets", "openstack subnets", "List Neutron subnets", []string{"subnet", "list"}},
        {"routers", "openstack routers", "List Neutron routers", []string{"router", "list"}},
        {"ports", "openstack ports", "List Neutron ports", []string{"port", "list"}},
        {"volumes", "openstack volumes", "List Cinder volumes", []string{"volume", "list"}},
    } {
        spec:=spec
        if err:=catalog.Add(provider.Operation{
            ID:"provider.openstack."+spec.id,
            Path:spec.path,
            Description:spec.description+" in the selected OpenStack cloud",
            Risk:"read",
            Handler:func(ctx context.Context,_ map[string]any)(any,error){
                return run(ctx,spec.args...)
            },
        });err!=nil{panic(err)}
    }
    return catalog
}

type cloudsFile struct {
    Clouds map[string]yaml.Node `yaml:"clouds"`
}

// cloudFilePath follows the standard search order used by openstacksdk.
// An explicit OS_CLIENT_CONFIG_FILE must not silently fall back to another file.
func cloudFilePath() (string,error) {
    if chosen:=strings.TrimSpace(os.Getenv("OS_CLIENT_CONFIG_FILE"));chosen!="" {
        info,err:=os.Stat(chosen)
        if err!=nil{return "",fmt.Errorf("OS_CLIENT_CONFIG_FILE: %w",err)}
        if info.IsDir(){return "",errors.New("OS_CLIENT_CONFIG_FILE is a directory")}
        return chosen,nil
    }
    candidates:=[]string{"clouds.yaml"}
    if config:=os.Getenv("XDG_CONFIG_HOME");config!="" {
        candidates=append(candidates,filepath.Join(config,"openstack","clouds.yaml"))
    } else if home,err:=os.UserHomeDir();err==nil {
        candidates=append(candidates,filepath.Join(home,".config","openstack","clouds.yaml"))
    }
    candidates=append(candidates,"/etc/openstack/clouds.yaml")
    for _,path:=range candidates {
        info,err:=os.Stat(path)
        if err==nil && !info.IsDir(){return path,nil}
        if err!=nil && !os.IsNotExist(err){return "",fmt.Errorf("find clouds.yaml: %w",err)}
    }
    return "",nil
}

func discoverConnections(ctx context.Context)([]provider.Connection,error){
    if err:=ctx.Err();err!=nil{return nil,err}
    path,err:=cloudFilePath()
    if err!=nil{return nil,err}
    names:=[]string{}
    if path!="" {
        raw,err:=os.ReadFile(path)
        if err!=nil{return nil,fmt.Errorf("read clouds.yaml: %w",err)}
        var cfg cloudsFile
        if err:=yaml.Unmarshal(raw,&cfg);err!=nil{
            // Never print file content or credentials in diagnostics.
            return nil,fmt.Errorf("invalid clouds.yaml at %s",path)
        }
        for name:=range cfg.Clouds{
            if strings.TrimSpace(name)==""{return nil,errors.New("clouds.yaml contains an empty cloud name")}
            names=append(names,name)
        }
    }
    sort.Strings(names)
    preferred:=strings.TrimSpace(os.Getenv("OS_CLOUD"))
    result:=make([]provider.Connection,0,len(names)+1)
    for _,name:=range names{
        result=append(result,provider.Connection{Provider:providerID,ID:name,Target:name,Default:name==preferred})
    }
    // Native OpenStack environment authentication is explicitly represented
    // as a separate connection and never replaces a selected named cloud.
    if strings.TrimSpace(os.Getenv("OS_AUTH_URL"))!=""{
        result=append(result,provider.Connection{Provider:providerID,ID:"env",Target:"env",Default:preferred==""})
    }
    return result,nil
}

func run(ctx context.Context,args ...string)(any,error){
    cloud,ok:=provider.ConnectionFromContext(ctx,providerID)
    if !ok{return nil,errors.New("no active OpenStack connection; use connections use openstack/<cloud>")}
    available,err:=discoverConnections(ctx)
    if err!=nil{return nil,err}
    found:=false
    for _,candidate:=range available{
        if candidate.ID==cloud{found=true;break}
    }
    if !found{return nil,fmt.Errorf("OpenStack connection %q is no longer available",cloud)}
    timeoutCtx,cancel:=context.WithTimeout(ctx,20*time.Second)
    defer cancel()
    argv:=[]string{}
    if cloud!="env" { argv=append(argv,"--os-cloud",cloud) }
    argv=append(argv,args...)
    argv=append(argv,"-f","json")
    cmd:=exec.CommandContext(timeoutCtx,"openstack",argv...)
    if cloud!="env" {
        // Prevent ambient OS_* auth variables from overriding another cloud's
        // credentials, project or region. Leave the config-file locator intact.
        env:=make([]string,0,len(os.Environ()))
        for _,entry:=range os.Environ(){
            key,_,_:=strings.Cut(entry,"=")
            if strings.HasPrefix(key,"OS_") && key!="OS_CLIENT_CONFIG_FILE" {continue}
            env=append(env,entry)
        }
        cmd.Env=env
    }
    var stdout,stderr bytes.Buffer
    cmd.Stdout=&stdout
    cmd.Stderr=&stderr
    err=cmd.Run()
    if err!=nil{
        if timeoutCtx.Err()!=nil{return nil,timeoutCtx.Err()}
        // Raw stderr can contain authentication tokens and endpoints.
        // Report only the failure class, not credentials or output.
        return nil,fmt.Errorf("openstack %s failed: %w",strings.Join(args," "),err)
    }
    if stdout.Len()>4<<20{return nil,errors.New("OpenStack response exceeds 4 MiB")}
    var value any
    if err:=json.Unmarshal(stdout.Bytes(),&value);err!=nil{
        return nil,fmt.Errorf("invalid OpenStack JSON for %s: %w",strings.Join(args," "),err)
    }
    return value,nil
}
