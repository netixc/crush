package plugin

import (
	"context"

	"charm.land/fantasy"
)

// ToolProvider is an interface that plugins can implement to provide custom tools
type ToolProvider interface {
	// GetTools returns the list of custom tools provided by this plugin
	GetTools() []PluginTool
}

// PluginTool defines the interface for a custom tool provided by a plugin.
// It mirrors the fantasy.AgentTool interface but with plugin-specific metadata.
type PluginTool interface {
	// Info returns metadata about the tool
	Info() fantasy.ToolInfo

	// Run executes the tool with the given parameters
	Run(ctx context.Context, params fantasy.ToolCall) (fantasy.ToolResponse, error)
}

// pluginToolAdapter adapts a PluginTool to the fantasy.AgentTool interface
type pluginToolAdapter struct {
	tool            PluginTool
	providerOptions fantasy.ProviderOptions
}

// NewAgentTool wraps a PluginTool to make it compatible with fantasy.AgentTool
func NewAgentTool(tool PluginTool) fantasy.AgentTool {
	return &pluginToolAdapter{
		tool:            tool,
		providerOptions: make(fantasy.ProviderOptions),
	}
}

func (a *pluginToolAdapter) Info() fantasy.ToolInfo {
	return a.tool.Info()
}

func (a *pluginToolAdapter) Run(ctx context.Context, params fantasy.ToolCall) (fantasy.ToolResponse, error) {
	return a.tool.Run(ctx, params)
}

func (a *pluginToolAdapter) ProviderOptions() fantasy.ProviderOptions {
	return a.providerOptions
}

func (a *pluginToolAdapter) SetProviderOptions(opts fantasy.ProviderOptions) {
	a.providerOptions = opts
}

// GetPluginTools extracts all custom tools from loaded plugins
func (r *Registry) GetPluginTools() []fantasy.AgentTool {
	var tools []fantasy.AgentTool

	for _, plugin := range r.plugins.Seq2() {
		// Check if plugin implements ToolProvider
		if toolProvider, ok := plugin.(ToolProvider); ok {
			for _, pluginTool := range toolProvider.GetTools() {
				tools = append(tools, NewAgentTool(pluginTool))
			}
		}
	}

	return tools
}
