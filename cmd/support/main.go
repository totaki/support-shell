package main

import (
	"context"
	"example.com/support-shell/internal/agent"
	"example.com/support-shell/internal/core"
	"example.com/support-shell/internal/provider"
	"example.com/support-shell/internal/providers/kubernetes"
	"example.com/support-shell/internal/providers/openstack"
	"example.com/support-shell/internal/mcp"
	"example.com/support-shell/internal/pluginmanager"
	"example.com/support-shell/internal/shell"
	"flag"
	"strings"
	"fmt"
	"os"
)

func main() {
	if err := loadDotEnv(".env"); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
	command := flag.String("command", "", "execute one command and exit")
	flag.Parse()
	r := core.NewRegistry()
	if err := configureToolPolicy(r); err != nil { fmt.Fprintln(os.Stderr, "tool policy:", err); os.Exit(1) }
	providers := provider.NewRegistry()
	if err := providers.Register(kubernetes.New()); err != nil { fmt.Fprintln(os.Stderr,"provider:",err);os.Exit(1) }
	if err := providers.Register(openstack.New()); err != nil { fmt.Fprintln(os.Stderr,"provider:",err);os.Exit(1) }
	if err := providers.BindTools(r); err != nil { fmt.Fprintln(os.Stderr,"provider tools:",err);os.Exit(1) }
	manager := pluginmanager.New()
	if err := manager.Register(r); err != nil { fmt.Fprintln(os.Stderr, "plugin manager:", err); os.Exit(1) }
	if err := loadPlugins(r, providers, manager); err != nil {
		fmt.Fprintln(os.Stderr, "plugin:", err)
		os.Exit(1)
	}
	if len(flag.Args()) == 2 && flag.Arg(0) == "mcp" && flag.Arg(1) == "serve" {
		ctx:=context.Background()
  session:=provider.NewSession()
  if err:=session.SelectDefaults(ctx,providers);err!=nil{
   fmt.Fprintln(os.Stderr,"provider connection discovery:",err)
  }
  if selected:=strings.TrimSpace(os.Getenv("SUPPORT_KUBERNETES_CONNECTION"));selected!=""{
   if err:=session.Select(ctx,providers,"kubernetes",selected);err!=nil{fmt.Fprintln(os.Stderr,"provider connection:",err);os.Exit(1)}
  }
  if selected:=strings.TrimSpace(os.Getenv("SUPPORT_OPENSTACK_CONNECTION"));selected!=""{
   if err:=session.Select(ctx,providers,"openstack",selected);err!=nil{fmt.Fprintln(os.Stderr,"provider connection:",err);os.Exit(1)}
  }
  if err := (&mcp.Server{Registry:r}).Serve(session.Bind(ctx),os.Stdin,os.Stdout);err!=nil {
			fmt.Fprintln(os.Stderr,"mcp:",err)
			os.Exit(1)
		}
		return
	}
	if len(flag.Args()) > 0 {
		fmt.Fprintln(os.Stderr,"usage: support [--command text] | support mcp serve")
		os.Exit(2)
	}
	s := &shell.Shell{Registry: r, Providers: providers, Connections:provider.NewSession(), Agent: agent.New(r)}
	s.LoadHistory()
	if *command != "" {
		if !s.Handle(context.Background(), *command) {
			os.Exit(0)
		}
		return
	}
	fmt.Println("Support Shell: commands + AI fallback (configure SUPPORT_API_KEY, SUPPORT_API_BASE, SUPPORT_MODEL)")
	s.Run(context.Background())
}
