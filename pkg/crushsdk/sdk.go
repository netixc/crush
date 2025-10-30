// Package crushsdk provides a public SDK for developing Crush plugins.
//
// This package offers helper functions and types that make it easier to create
// plugins that extend Crush's functionality through hooks and custom tools.
package crushsdk

import (
	"context"

	"github.com/charmbracelet/crush/internal/plugin"
	"github.com/charmbracelet/fantasy"
)

// Re-export key types for plugin developers
type (
	// Plugin is the main interface that all plugins must implement
	Plugin = plugin.Plugin

	// PluginInfo contains metadata about a plugin
	PluginInfo = plugin.PluginInfo

	// PluginContext provides plugins with access to application services
	PluginContext = plugin.PluginContext

	// Hooks defines all available hook points
	Hooks = plugin.Hooks

	// ConfigHook allows plugins to modify configuration
	ConfigHook = plugin.ConfigHook

	// SessionHook provides hooks for session lifecycle events
	SessionHook = plugin.SessionHook

	// MessageHook provides hooks for message lifecycle events
	MessageHook = plugin.MessageHook

	// PermissionHook provides hooks for permission requests
	PermissionHook = plugin.PermissionHook

	// ToolHook provides hooks for tool execution
	ToolHook = plugin.ToolHook

	// AgentHook provides hooks for agent lifecycle
	AgentHook = plugin.AgentHook

	// ToolExecuteInput contains information about a tool execution
	ToolExecuteInput = plugin.ToolExecuteInput

	// ToolExecuteResult contains the result of a tool execution
	ToolExecuteResult = plugin.ToolExecuteResult

	// AgentStartInput contains information about an agent starting
	AgentStartInput = plugin.AgentStartInput

	// AgentStepInput contains information about an agent step
	AgentStepInput = plugin.AgentStepInput

	// AgentFinishInput contains information about an agent finishing
	AgentFinishInput = plugin.AgentFinishInput

	// PluginTool defines the interface for custom tools
	PluginTool = plugin.PluginTool

	// ToolProvider is implemented by plugins that provide custom tools
	ToolProvider = plugin.ToolProvider
)

// Helper functions

// NewBaseHooks creates a BaseHooks struct with all nil implementations.
// Plugins can embed this and override only the hooks they need.
func NewBaseHooks() *plugin.BaseHooks {
	return plugin.NewBaseHooks()
}

// SimplePlugin provides a base implementation that plugins can embed.
// It handles the basic plugin lifecycle and allows plugins to focus on
// implementing their specific hooks and tools.
type SimplePlugin struct {
	info        PluginInfo
	hooks       Hooks
	tools       []PluginTool
	initialized bool
}

// NewSimplePlugin creates a new SimplePlugin with the given metadata
func NewSimplePlugin(info PluginInfo) *SimplePlugin {
	return &SimplePlugin{
		info:  info,
		hooks: plugin.NewBaseHooks(),
		tools: []PluginTool{},
	}
}

func (p *SimplePlugin) Info() PluginInfo {
	return p.info
}

func (p *SimplePlugin) Init(ctx context.Context, pluginCtx PluginContext) error {
	p.initialized = true
	return nil
}

func (p *SimplePlugin) Hooks() Hooks {
	return p.hooks
}

func (p *SimplePlugin) Shutdown(ctx context.Context) error {
	return nil
}

// SetHooks allows setting custom hooks
func (p *SimplePlugin) SetHooks(hooks Hooks) {
	p.hooks = hooks
}

// AddTool adds a custom tool to the plugin
func (p *SimplePlugin) AddTool(tool PluginTool) {
	p.tools = append(p.tools, tool)
}

// GetTools implements ToolProvider
func (p *SimplePlugin) GetTools() []PluginTool {
	return p.tools
}

// SimpleTool provides a helper for creating simple tools
type SimpleTool struct {
	info    fantasy.ToolInfo
	handler func(ctx context.Context, params fantasy.ToolCall) (fantasy.ToolResponse, error)
}

// NewSimpleTool creates a new SimpleTool
func NewSimpleTool(
	name string,
	description string,
	parameters map[string]any,
	required []string,
	handler func(ctx context.Context, params fantasy.ToolCall) (fantasy.ToolResponse, error),
) *SimpleTool {
	return &SimpleTool{
		info: fantasy.ToolInfo{
			Name:        name,
			Description: description,
			Parameters:  parameters,
			Required:    required,
		},
		handler: handler,
	}
}

func (t *SimpleTool) Info() fantasy.ToolInfo {
	return t.info
}

func (t *SimpleTool) Run(ctx context.Context, params fantasy.ToolCall) (fantasy.ToolResponse, error) {
	return t.handler(ctx, params)
}

// Permission helpers

// Allow returns a pointer to true for permission hooks
func Allow() *bool {
	t := true
	return &t
}

// Deny returns a pointer to false for permission hooks
func Deny() *bool {
	f := false
	return &f
}

// NoDecision returns nil for permission hooks (let another hook or user decide)
func NoDecision() *bool {
	return nil
}
