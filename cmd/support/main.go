package main

import (
	"context"
	"example.com/support-shell/internal/agent"
	"example.com/support-shell/internal/core"
	"example.com/support-shell/internal/modules"
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
	if err := loadPlugins(r); err != nil {
		fmt.Fprintln(os.Stderr, "plugin:", err)
		os.Exit(1)
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
