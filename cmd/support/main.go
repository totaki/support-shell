package main

import (
	"context"
	"example.com/support-shell/internal/agent"
	"example.com/support-shell/internal/core"
	"example.com/support-shell/internal/modules"
	"example.com/support-shell/internal/mcp"
	"example.com/support-shell/internal/pluginmanager"
	"example.com/support-shell/internal/shell"
	"flag"
	"fmt"
	"os"
)

func main() {
	command := flag.String("command", "", "execute one command and exit")
	flag.Parse()
	r := core.NewRegistry()
	modules.RegisterDemo(r)
	modules.RegisterKubernetes(r)
	manager := pluginmanager.New()
	if err := manager.Register(r); err != nil { fmt.Fprintln(os.Stderr, "plugin manager:", err); os.Exit(1) }
	if err := loadPlugins(r, manager); err != nil {
		fmt.Fprintln(os.Stderr, "plugin:", err)
		os.Exit(1)
	}
	if len(flag.Args()) == 2 && flag.Arg(0) == "mcp" && flag.Arg(1) == "serve" {
		if err := (&mcp.Server{Registry:r}).Serve(context.Background(),os.Stdin,os.Stdout);err!=nil {
			fmt.Fprintln(os.Stderr,"mcp:",err)
			os.Exit(1)
		}
		return
	}
	if len(flag.Args()) > 0 {
		fmt.Fprintln(os.Stderr,"usage: support [--command text] | support mcp serve")
		os.Exit(2)
	}
	s := &shell.Shell{Registry: r, Agent: agent.New(r)}
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
