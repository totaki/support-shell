//go:build !extism

package main

import "example.com/support-shell/internal/core"

func loadPlugins(r *core.Registry) error { return nil }
