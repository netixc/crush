# Crush Plugin Development Guide

This guide explains how to create plugins for Crush that extend its functionality through hooks and custom tools.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Plugin Architecture](#plugin-architecture)
- [Available Hooks](#available-hooks)
- [Creating Custom Tools](#creating-custom-tools)
- [Building and Installing Plugins](#building-and-installing-plugins)
- [Best Practices](#best-practices)
- [Examples](#examples)

## Overview

Crush plugins allow you to:

- **Add custom tools** that the AI agent can use
- **Intercept and modify** configuration, permissions, and tool execution
- **React to events** like session creation, message updates, and agent lifecycle
- **Extend functionality** without modifying Crush's core code

Plugins are implemented as Go shared libraries (`.so` files) that implement the `Plugin` interface.

## Quick Start

### 1. Create a new plugin

```go
package main

import (
	"context"
	"github.com/charmbracelet/crush/pkg/crushsdk"
)

// Plugin is the exported symbol that Crush will load
var Plugin crushsdk.Plugin = &MyPlugin{}

type MyPlugin struct {
	*crushsdk.SimplePlugin
}

func init() {
	base := crushsdk.NewSimplePlugin(crushsdk.PluginInfo{
		Name:        "my-plugin",
		Version:     "1.0.0",
		Description: "My awesome Crush plugin",
		Author:      "Your Name",
	})

	Plugin = &MyPlugin{SimplePlugin: base}
}
```

### 2. Build the plugin

```bash
go build -buildmode=plugin -o my-plugin.so main.go
```

### 3. Configure Crush to load your plugin

Add to your `crush.json`:

```json
{
  "plugins": [
    "./path/to/my-plugin.so"
  ]
}
```

## Plugin Architecture

### Plugin Interface

All plugins must implement the `Plugin` interface:

```go
type Plugin interface {
    // Info returns metadata about the plugin
    Info() PluginInfo

    // Init is called when the plugin is loaded
    Init(ctx context.Context, pluginCtx PluginContext) error

    // Hooks returns the hook implementations
    Hooks() Hooks

    // Shutdown is called when the application shuts down
    Shutdown(ctx context.Context) error
}
```

### Plugin Context

During initialization, plugins receive a `PluginContext` with access to:

```go
type PluginContext struct {
    // Config is the application configuration
    Config *config.Config

    // Services provides access to core services
    Services Services

    // WorkingDir is the current working directory
    WorkingDir string
}

type Services struct {
    Session    session.Service    // Manage sessions
    Message    message.Service    // Manage messages
    Permission permission.Service // Handle permissions
}
```

### Using SimplePlugin

The SDK provides `SimplePlugin` to handle boilerplate:

```go
func init() {
    plugin := &MyPlugin{
        SimplePlugin: crushsdk.NewSimplePlugin(crushsdk.PluginInfo{
            Name:        "my-plugin",
            Version:     "1.0.0",
            Description: "Description",
            Author:      "Author",
        }),
    }

    // Add hooks
    hooks := crushsdk.NewBaseHooks()
    hooks.SessionHook = &mySessionHook{}
    plugin.SetHooks(hooks)

    // Add tools
    plugin.AddTool(myTool)

    Plugin = plugin
}
```

## Available Hooks

### Config Hook

Modify configuration after it's loaded:

```go
type ConfigHook interface {
    OnConfigLoad(ctx context.Context, cfg *config.Config) error
}
```

**Use cases:**
- Add default settings
- Validate configuration
- Inject custom agent configurations

### Session Hooks

React to session lifecycle events:

```go
type SessionHook interface {
    OnSessionCreated(ctx context.Context, sess session.Session) error
    OnSessionUpdated(ctx context.Context, sess session.Session) error
    OnSessionDeleted(ctx context.Context, sessionID string) error
}
```

**Use cases:**
- Track active sessions
- Initialize session-specific state
- Clean up resources when sessions end

### Message Hooks

React to message events:

```go
type MessageHook interface {
    OnMessageCreated(ctx context.Context, msg message.Message) error
    OnMessageUpdated(ctx context.Context, msg message.Message) error
}
```

**Use cases:**
- Log conversation history
- Analyze message patterns
- Trigger actions based on message content

### Permission Hook

Intercept permission requests:

```go
type PermissionHook interface {
    OnPermissionRequest(ctx context.Context, req permission.CreatePermissionRequest) (*bool, error)
}
```

**Return values:**
- `crushsdk.Allow()` - Auto-approve the request
- `crushsdk.Deny()` - Auto-deny the request
- `crushsdk.NoDecision()` - Let another plugin or user decide

**Use cases:**
- Auto-approve read-only operations
- Enforce security policies
- Implement custom approval logic

**Example:**

```go
func (h *MyHook) OnPermissionRequest(ctx context.Context, req permission.CreatePermissionRequest) (*bool, error) {
    // Auto-approve read-only tools
    if req.ToolName == "view" || req.ToolName == "grep" {
        return crushsdk.Allow(), nil
    }
    return crushsdk.NoDecision(), nil
}
```

### Tool Hooks

Intercept tool execution:

```go
type ToolHook interface {
    // Called before tool execution - can modify arguments
    OnToolExecuteBefore(ctx context.Context, input ToolExecuteInput) (map[string]any, error)

    // Called after tool execution - can modify result
    OnToolExecuteAfter(ctx context.Context, input ToolExecuteInput, result ToolExecuteResult) (*ToolExecuteResult, error)
}
```

**Use cases:**
- Log tool usage
- Modify tool arguments or results
- Add timing metrics
- Implement caching

**Example:**

```go
func (h *MyHook) OnToolExecuteBefore(ctx context.Context, input ToolExecuteInput) (map[string]any, error) {
    log.Printf("Executing tool: %s", input.ToolName)
    // Return nil to not modify arguments
    return nil, nil
}

func (h *MyHook) OnToolExecuteAfter(ctx context.Context, input ToolExecuteInput, result ToolExecuteResult) (*ToolExecuteResult, error) {
    // Append metadata to result
    result.Output += "\n(Logged by my-plugin)"
    return &result, nil
}
```

### Agent Hooks

Track agent execution lifecycle:

```go
type AgentHook interface {
    OnAgentStart(ctx context.Context, input AgentStartInput) error
    OnAgentStep(ctx context.Context, input AgentStepInput) error
    OnAgentFinish(ctx context.Context, input AgentFinishInput) error
}
```

**Use cases:**
- Collect execution metrics
- Monitor agent performance
- Track token usage
- Implement custom logging

## Creating Custom Tools

Plugins can add custom tools that the AI agent can use.

### Simple Tool Example

```go
func init() {
    plugin := crushsdk.NewSimplePlugin(/* ... */)

    // Create a simple tool
    helloTool := crushsdk.NewSimpleTool(
        "hello",                          // Tool name
        "Says hello to a person",         // Description
        map[string]any{                   // Parameters (JSON Schema)
            "name": map[string]any{
                "type":        "string",
                "description": "Person's name",
            },
        },
        []string{"name"},                 // Required parameters
        func(ctx context.Context, params fantasy.ToolCall) (fantasy.ToolResponse, error) {
            var input struct {
                Name string `json:"name"`
            }
            json.Unmarshal(params.Input, &input)

            return fantasy.ToolResponse{
                Text: fmt.Sprintf("Hello, %s!", input.Name),
            }, nil
        },
    )

    plugin.AddTool(helloTool)
}
```

### Advanced Tool Implementation

For more control, implement the `PluginTool` interface:

```go
type MyTool struct {
    // Your tool state
}

func (t *MyTool) Info() fantasy.ToolInfo {
    return fantasy.ToolInfo{
        Name:        "my-tool",
        Description: "Does something useful",
        Parameters: map[string]any{
            "input": map[string]any{
                "type":        "string",
                "description": "Input data",
            },
        },
        Required: []string{"input"},
    }
}

func (t *MyTool) Run(ctx context.Context, params fantasy.ToolCall) (fantasy.ToolResponse, error) {
    // Parse input
    var input struct {
        Input string `json:"input"`
    }
    if err := json.Unmarshal(params.Input, &input); err != nil {
        return fantasy.ToolResponse{
            Error: err,
        }, nil
    }

    // Do work
    result := doSomething(input.Input)

    return fantasy.ToolResponse{
        Text: result,
    }, nil
}
```

### Tool Parameters Schema

Tool parameters use JSON Schema format:

```go
Parameters: map[string]any{
    "query": map[string]any{
        "type":        "string",
        "description": "Search query",
    },
    "limit": map[string]any{
        "type":        "integer",
        "description": "Maximum results",
        "default":     10,
        "minimum":     1,
        "maximum":     100,
    },
    "filters": map[string]any{
        "type": "array",
        "items": map[string]any{
            "type": "string",
        },
        "description": "Filter criteria",
    },
}
```

## Building and Installing Plugins

### Building

Plugins must be built with `-buildmode=plugin`:

```bash
go build -buildmode=plugin -o plugin-name.so main.go
```

**Important Notes:**

1. **Go Version Compatibility**: The plugin must be built with the **same Go version** as Crush
2. **Architecture**: Must match the architecture Crush is running on (amd64, arm64, etc.)
3. **Dependencies**: Plugin dependencies should be compatible with Crush's dependencies

### Installing

#### Option 1: Absolute Path

```json
{
  "plugins": [
    "/Users/me/plugins/my-plugin.so"
  ]
}
```

#### Option 2: Relative Path

Relative to your project directory:

```json
{
  "plugins": [
    "./plugins/my-plugin.so"
  ]
}
```

#### Option 3: Plugin Directory

Place `.so` file in a directory:

```json
{
  "plugins": [
    "./plugins/my-plugin/"
  ]
}
```

Crush will find the `.so` file in the directory.

### Debugging

Enable debug logging to see plugin loading:

```bash
CRUSH_LOG_LEVEL=debug crush
```

Check logs for:
- Plugin loading messages
- Hook execution errors
- Tool registration

## Best Practices

### 1. Error Handling

Always handle errors gracefully:

```go
func (h *MyHook) OnSessionCreated(ctx context.Context, sess session.Session) error {
    if err := h.doSomething(); err != nil {
        // Log but don't fail - hooks should be resilient
        slog.Error("Hook failed", "error", err)
        return nil
    }
    return nil
}
```

### 2. Performance

Hooks are called synchronously - keep them fast:

```go
// Good: Fast operation
func (h *MyHook) OnMessageCreated(ctx context.Context, msg message.Message) error {
    h.counter++
    return nil
}

// Bad: Slow operation
func (h *MyHook) OnMessageCreated(ctx context.Context, msg message.Message) error {
    // This blocks message processing!
    time.Sleep(5 * time.Second)
    return nil
}

// Good: Use goroutines for slow work
func (h *MyHook) OnMessageCreated(ctx context.Context, msg message.Message) error {
    go h.doSlowWork(msg) // Don't block
    return nil
}
```

### 3. Context Awareness

Respect context cancellation:

```go
func (t *MyTool) Run(ctx context.Context, params fantasy.ToolCall) (fantasy.ToolResponse, error) {
    select {
    case <-ctx.Done():
        return fantasy.ToolResponse{
            Error: ctx.Err(),
        }, nil
    default:
        // Do work
    }
}
```

### 4. Thread Safety

Use mutexes for shared state:

```go
type MyPlugin struct {
    *crushsdk.SimplePlugin
    mu      sync.RWMutex
    counter int
}

func (p *MyPlugin) incrementCounter() {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.counter++
}
```

### 5. Logging

Use structured logging:

```go
import "log/slog"

slog.Info("Plugin action",
    "plugin", "my-plugin",
    "session", sessionID,
    "count", count)

slog.Error("Operation failed",
    "error", err,
    "context", "additional info")
```

### 6. Resource Cleanup

Clean up in Shutdown:

```go
func (p *MyPlugin) Shutdown(ctx context.Context) error {
    // Close connections
    if p.conn != nil {
        p.conn.Close()
    }

    // Stop goroutines
    close(p.stopCh)
    p.wg.Wait()

    return nil
}
```

## Examples

### Complete Examples

See the `examples/plugins/` directory for full examples:

1. **hello-world** - Basic plugin with custom tool
   - File: `examples/plugins/hello-world/main.go`
   - Demonstrates: Tool registration, parameter handling

2. **auto-approve** - Permission automation
   - File: `examples/plugins/auto-approve/main.go`
   - Demonstrates: Permission hooks, policy implementation

3. **metrics** - Usage tracking
   - File: `examples/plugins/metrics/main.go`
   - Demonstrates: Multiple hooks, metrics collection

### Example Use Cases

#### Logging Plugin

```go
type LoggingPlugin struct {
    *crushsdk.SimplePlugin
    logFile *os.File
}

func init() {
    plugin := &LoggingPlugin{
        SimplePlugin: crushsdk.NewSimplePlugin(/* ... */),
    }

    hooks := crushsdk.NewBaseHooks()
    hooks.ToolHook = &loggingToolHook{plugin: plugin}
    plugin.SetHooks(hooks)

    Plugin = plugin
}

func (p *LoggingPlugin) Init(ctx context.Context, pluginCtx crushsdk.PluginContext) error {
    var err error
    p.logFile, err = os.OpenFile("crush-tools.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    return err
}

type loggingToolHook struct {
    plugin *LoggingPlugin
    crushsdk.NilToolHook
}

func (h *loggingToolHook) OnToolExecuteBefore(ctx context.Context, input crushsdk.ToolExecuteInput) (map[string]any, error) {
    fmt.Fprintf(h.plugin.logFile, "%s: Executing %s\n", time.Now(), input.ToolName)
    return nil, nil
}
```

#### Custom Search Tool

```go
searchTool := crushsdk.NewSimpleTool(
    "search-docs",
    "Searches internal documentation",
    map[string]any{
        "query": map[string]any{
            "type":        "string",
            "description": "Search query",
        },
    },
    []string{"query"},
    func(ctx context.Context, params fantasy.ToolCall) (fantasy.ToolResponse, error) {
        var input struct {
            Query string `json:"query"`
        }
        json.Unmarshal(params.Input, &input)

        // Search your internal docs
        results := searchInternalDocs(input.Query)

        return fantasy.ToolResponse{
            Text: formatResults(results),
        }, nil
    },
)
```

## Troubleshooting

### Plugin won't load

1. Check Go version compatibility
2. Verify the `.so` file exists at the specified path
3. Ensure the `Plugin` variable is exported
4. Check build flags: must use `-buildmode=plugin`

### Hook not being called

1. Verify you're implementing the correct interface
2. Check that hooks are registered via `SetHooks()`
3. Look for errors in logs

### Tool not appearing

1. Verify the plugin implements `ToolProvider`
2. Check that tools are added via `AddTool()`
3. Ensure tool has valid JSON schema

## Advanced Topics

### Plugin State Management

Store plugin state in memory or externally:

```go
type MyPlugin struct {
    *crushsdk.SimplePlugin
    db *sql.DB  // External database
}
```

### Inter-Plugin Communication

Plugins run in the same process and can communicate through shared state (use carefully):

```go
var SharedRegistry = make(map[string]interface{})

func (p *MyPlugin) Init(ctx context.Context, pluginCtx crushsdk.PluginContext) error {
    SharedRegistry["my-plugin"] = p.getAPI()
    return nil
}
```

### Dynamic Tool Generation

Generate tools programmatically:

```go
func (p *MyPlugin) Init(ctx context.Context, pluginCtx crushsdk.PluginContext) error {
    // Read API definitions
    apis := loadAPIs()

    // Generate tool for each API
    for _, api := range apis {
        tool := generateToolFromAPI(api)
        p.AddTool(tool)
    }

    return nil
}
```

## Resources

- **Crush SDK**: `pkg/crushsdk/`
- **Internal Plugin Package**: `internal/plugin/`
- **Example Plugins**: `examples/plugins/`
- **Fantasy SDK Docs**: https://github.com/charmbracelet/fantasy

## Contributing

We welcome plugin contributions! To share your plugin:

1. Create a repository for your plugin
2. Include build instructions
3. Document configuration and usage
4. Submit a PR to add it to the plugin registry (coming soon)

## License

Plugins are separate from Crush and can use any license you choose.
