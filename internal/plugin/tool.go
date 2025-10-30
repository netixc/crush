package plugin

import (
	"context"

	"github.com/charmbracelet/fantasy"
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
	tool PluginTool
}

// NewAgentTool wraps a PluginTool to make it compatible with fantasy.AgentTool
func NewAgentTool(tool PluginTool) fantasy.AgentTool {
	return &pluginToolAdapter{tool: tool}
}

func (a *pluginToolAdapter) Info() fantasy.ToolInfo {
	return a.tool.Info()
}

func (a *pluginToolAdapter) Run(ctx context.Context, params fantasy.ToolCall) (fantasy.ToolResponse, error) {
	return a.tool.Run(ctx, params)
}

// GetPluginTools extracts all custom tools from loaded plugins
func (r *Registry) GetPluginTools() []fantasy.AgentTool {
	var tools []fantasy.AgentTool

	r.plugins.Range(func(_ string, plugin Plugin) bool {
		// Check if plugin implements ToolProvider
		if toolProvider, ok := plugin.(ToolProvider); ok {
			for _, pluginTool := range toolProvider.GetTools() {
				tools = append(tools, NewAgentTool(pluginTool))
			}
		}
		return true
	})

	return tools
}
