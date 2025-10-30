// Package main provides an auto-approve plugin example for Crush.
//
// This plugin demonstrates:
// - Implementing permission hooks
// - Auto-approving specific tools or patterns
// - Reading configuration from plugin context
//
// To build this plugin:
//   go build -buildmode=plugin -o auto-approve.so main.go
//
// To use this plugin, add to your crush config:
//   {
//     "plugins": ["./examples/plugins/auto-approve/auto-approve.so"]
//   }
package main

import (
	"context"
	"log/slog"
	"strings"

	"github.com/charmbracelet/crush/internal/permission"
	"github.com/charmbracelet/crush/pkg/crushsdk"
)

// Plugin is the exported symbol that Crush will load
var Plugin crushsdk.Plugin = &AutoApprovePlugin{}

// AutoApprovePlugin automatically approves read-only tools
type AutoApprovePlugin struct {
	*crushsdk.SimplePlugin
	readOnlyTools map[string]bool
}

func init() {
	plugin := &AutoApprovePlugin{
		SimplePlugin: crushsdk.NewSimplePlugin(crushsdk.PluginInfo{
			Name:        "auto-approve",
			Version:     "1.0.0",
			Description: "Automatically approves permission requests for read-only tools",
			Author:      "Crush Examples",
		}),
		readOnlyTools: map[string]bool{
			"view":   true,
			"glob":   true,
			"grep":   true,
			"ls":     true,
			"fetch":  true,
		},
	}

	// Set up custom hooks
	hooks := crushsdk.NewBaseHooks()
	hooks.PermissionHook = &autoApprovePermissionHook{plugin: plugin}
	plugin.SetHooks(hooks)

	Plugin = plugin
}

func (p *AutoApprovePlugin) Init(ctx context.Context, pluginCtx crushsdk.PluginContext) error {
	slog.Info("Auto-approve plugin initialized",
		"read_only_tools", len(p.readOnlyTools))
	return p.SimplePlugin.Init(ctx, pluginCtx)
}

// autoApprovePermissionHook implements PermissionHook
type autoApprovePermissionHook struct {
	plugin *AutoApprovePlugin
	crushsdk.NilPermissionHook // Embed to get default implementations
}

func (h *autoApprovePermissionHook) OnPermissionRequest(
	ctx context.Context,
	req permission.CreatePermissionRequest,
) (*bool, error) {
	// Auto-approve read-only tools
	if h.plugin.readOnlyTools[req.ToolName] {
		slog.Debug("Auto-approving read-only tool",
			"tool", req.ToolName,
			"session", req.SessionID)
		return crushsdk.Allow(), nil
	}

	// Auto-approve tools with "read" in their action
	if strings.Contains(strings.ToLower(req.Action), "read") {
		slog.Debug("Auto-approving read action",
			"tool", req.ToolName,
			"action", req.Action)
		return crushsdk.Allow(), nil
	}

	// Let other plugins or the user decide
	return crushsdk.NoDecision(), nil
}
