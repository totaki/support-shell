//go:build !extism

package main

import (
 "example.com/support-shell/internal/core"
 "example.com/support-shell/internal/pluginmanager"
)

func loadPlugins(r *core.Registry, _ *pluginmanager.Manager) error { return nil }
