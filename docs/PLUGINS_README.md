# Crush Plugin System

Crush now features a powerful plugin system inspired by OpenCode's extensibility, allowing you to customize and extend Crush's functionality without modifying core code.

## What Can Plugins Do?

- ✅ **Add Custom Tools** - Create new tools the AI agent can use
- ✅ **Intercept Permissions** - Auto-approve/deny tool executions based on custom logic
- ✅ **Modify Configuration** - Inject settings and customize behavior
- ✅ **React to Events** - Listen to sessions, messages, and tool executions
- ✅ **Track Metrics** - Monitor usage, performance, and errors
- ✅ **Extend Agent Behavior** - Hook into agent lifecycle events

## Quick Example

Here's a complete plugin in ~30 lines:

```go
package main

import (
    "context"
    "fmt"
    "github.com/charmbracelet/crush/pkg/crushsdk"
    "github.com/charmbracelet/fantasy"
)

var Plugin crushsdk.Plugin = &HelloPlugin{}

type HelloPlugin struct {
    *crushsdk.SimplePlugin
}

func init() {
    base := crushsdk.NewSimplePlugin(crushsdk.PluginInfo{
        Name: "hello", Version: "1.0.0",
        Description: "Adds a hello tool",
        Author: "You",
    })

    helloTool := crushsdk.NewSimpleTool(
        "hello", "Greets someone",
        map[string]any{
            "name": map[string]any{"type": "string", "description": "Name to greet"},
        },
        []string{"name"},
        func(ctx context.Context, params fantasy.ToolCall) (fantasy.ToolResponse, error) {
            // Parse input and return response
            return fantasy.ToolResponse{Text: "Hello!"}, nil
        },
    )

    base.AddTool(helloTool)
    Plugin = &HelloPlugin{SimplePlugin: base}
}
```

Build and use:

```bash
go build -buildmode=plugin -o hello.so
```

```json
{
  "plugins": ["./hello.so"]
}
```

## Architecture

### Plugin System Components

```
┌─────────────────────────────────────────────────────┐
│                      Crush App                       │
│                                                      │
│  ┌────────────────────────────────────────────┐    │
│  │         Plugin Registry                    │    │
│  │  - Loads .so files                         │    │
│  │  - Manages hooks                           │    │
│  │  - Registers tools                         │    │
│  └────────────────────────────────────────────┘    │
│                       │                              │
│          ┌────────────┼────────────┐                │
│          │            │            │                 │
│    ┌─────▼───┐  ┌────▼────┐  ┌───▼────┐           │
│    │ Plugin  │  │ Plugin  │  │ Plugin │           │
│    │    1    │  │    2    │  │    3   │           │
│    └─────────┘  └─────────┘  └────────┘           │
└─────────────────────────────────────────────────────┘
```

### Hook Execution Flow

```
Agent Run Start
     │
     ▼
OnAgentStart Hook (Plugins)
     │
     ▼
For each step:
     │
     ▼
Tool Selection
     │
     ▼
OnPermissionRequest Hook (Plugins) ──> Auto Approve/Deny?
     │                                       │
     ▼                                       ▼
User Prompt (if not decided)         OnToolExecuteBefore Hook
     │                                       │
     ▼                                       ▼
Tool Execution                         Tool Execution
     │                                       │
     ▼                                       ▼
OnToolExecuteAfter Hook (Plugins) <────────┘
     │
     ▼
OnAgentStep Hook (Plugins)
     │
     ▼
OnAgentFinish Hook (Plugins)
```

## Available Hooks

| Hook Type | Methods | Purpose |
|-----------|---------|---------|
| **Config** | `OnConfigLoad` | Modify config after loading |
| **Session** | `OnSessionCreated`, `OnSessionUpdated`, `OnSessionDeleted` | Track sessions |
| **Message** | `OnMessageCreated`, `OnMessageUpdated` | Monitor messages |
| **Permission** | `OnPermissionRequest` | Auto-approve/deny tools |
| **Tool** | `OnToolExecuteBefore`, `OnToolExecuteAfter` | Intercept tool execution |
| **Agent** | `OnAgentStart`, `OnAgentStep`, `OnAgentFinish` | Track agent lifecycle |

## Comparison with OpenCode

Crush's plugin system was inspired by OpenCode but adapted for Go:

| Feature | OpenCode (TypeScript) | Crush (Go) | Status |
|---------|----------------------|------------|--------|
| Custom Tools | ✅ TypeScript SDK | ✅ Go SDK | ✅ Complete |
| Permission Hooks | ✅ `permission.ask` | ✅ `OnPermissionRequest` | ✅ Complete |
| Config Hooks | ✅ `config` | ✅ `OnConfigLoad` | ✅ Complete |
| Tool Hooks | ✅ `tool.execute.before/after` | ✅ `OnToolExecuteBefore/After` | ✅ Complete |
| Event Hooks | ✅ Event bus | ✅ Pub/sub events | ✅ Complete |
| Agent Hooks | ✅ `chat.*` | ✅ `OnAgentStart/Step/Finish` | ✅ Complete |
| Plugin Loading | ✅ npm packages | ✅ Go shared libs | ✅ Complete |
| Dynamic Tools | ✅ Runtime registration | ✅ Runtime registration | ✅ Complete |

### Key Differences

1. **Loading Mechanism**:
   - OpenCode: npm packages with dynamic `import()`
   - Crush: Go shared libraries (`.so` files)

2. **Type Safety**:
   - OpenCode: TypeScript with Zod schemas
   - Crush: Go with compile-time type checking

3. **Performance**:
   - OpenCode: Interpreted JavaScript
   - Crush: Compiled native code

4. **Dependency Management**:
   - OpenCode: npm/package.json
   - Crush: Go modules (must match Crush's Go version)

## Examples

### 1. Hello World Plugin

Adds a simple greeting tool.

**Location**: `examples/plugins/hello-world/`

**Features**:
- Custom tool registration
- Parameter parsing
- JSON schema definition

### 2. Auto-Approve Plugin

Automatically approves read-only tools.

**Location**: `examples/plugins/auto-approve/`

**Features**:
- Permission hooks
- Policy implementation
- Pattern matching

**Use case**: Skip permission prompts for safe tools like `view`, `grep`, `ls`

### 3. Metrics Plugin

Collects usage statistics and reports them periodically.

**Location**: `examples/plugins/metrics/`

**Features**:
- Multiple hook types
- Session tracking
- Message monitoring
- Tool execution stats
- Periodic reporting

**Use case**: Understand usage patterns, track most-used tools, monitor errors

## Building Plugins

### Requirements

- Go 1.25.0 (must match Crush's Go version)
- Same architecture as Crush (amd64/arm64)

### Build Command

```bash
go build -buildmode=plugin -o plugin-name.so main.go
```

### Testing

```bash
# Build Crush with plugin support
cd crush
go build

# Build your plugin
cd my-plugin
go build -buildmode=plugin -o my-plugin.so

# Configure Crush
cat > crush.json <<EOF
{
  "plugins": ["./my-plugin/my-plugin.so"]
}
EOF

# Run Crush with debug logging
CRUSH_LOG_LEVEL=debug ./crush
```

## Configuration

### Basic Configuration

```json
{
  "plugins": [
    "./plugins/my-plugin.so",
    "/absolute/path/to/plugin.so"
  ]
}
```

### Plugin Discovery

Crush searches for `.so` files in:
1. Exact file paths (if `.so` extension)
2. Directories (looks for first `.so` file found)

## SDK Reference

The `crushsdk` package provides:

### Core Types

```go
// Main plugin interface
type Plugin interface {
    Info() PluginInfo
    Init(ctx context.Context, pluginCtx PluginContext) error
    Hooks() Hooks
    Shutdown(ctx context.Context) error
}

// Simplified base implementation
type SimplePlugin struct { /* ... */ }

// Tool creation helper
func NewSimpleTool(name, desc string, params map[string]any, ...) *SimpleTool
```

### Hook Helpers

```go
// No-op implementations to embed
type NilConfigHook struct{}
type NilSessionHook struct{}
type NilMessageHook struct{}
type NilPermissionHook struct{}
type NilToolHook struct{}
type NilAgentHook struct{}

// Base hooks collection
func NewBaseHooks() *BaseHooks
```

### Permission Helpers

```go
crushsdk.Allow()      // Auto-approve
crushsdk.Deny()       // Auto-deny
crushsdk.NoDecision() // Let user/other plugins decide
```

## Best Practices

### ✅ Do

- Keep hooks fast (they run synchronously)
- Handle errors gracefully
- Use goroutines for slow operations
- Clean up resources in `Shutdown()`
- Use structured logging (`slog`)
- Implement only the hooks you need
- Test with same Go version as Crush

### ❌ Don't

- Block hook execution with slow operations
- Panic in hook implementations
- Modify shared state without synchronization
- Ignore context cancellation
- Use incompatible Go versions
- Depend on plugin load order

## Performance Considerations

Hooks are called **synchronously** in sequence:

```
Plugin 1 → Plugin 2 → Plugin 3 → ... → Plugin N
```

Each hook execution adds to total latency. For performance-critical paths:

1. **Keep hooks minimal**:
   ```go
   // Good: Fast operation
   func (h *Hook) OnToolExecuteBefore(ctx context.Context, input Input) (map[string]any, error) {
       h.counter++  // Fast
       return nil, nil
   }
   ```

2. **Offload slow work**:
   ```go
   // Good: Async processing
   func (h *Hook) OnMessageCreated(ctx context.Context, msg Message) error {
       go h.processAsync(msg)  // Don't block
       return nil
   }
   ```

3. **Use caching**:
   ```go
   func (h *Hook) OnPermissionRequest(ctx context.Context, req Request) (*bool, error) {
       if decision, ok := h.cache[req.ToolName]; ok {
           return decision, nil  // Fast cache hit
       }
       // Compute and cache...
   }
   ```

## Troubleshooting

### Plugin won't load

```
Error: plugin does not export 'Plugin' symbol
```

**Solution**: Ensure you have `var Plugin crushsdk.Plugin = ...` at package level

---

```
Error: plugin was built with a different version of package
```

**Solution**: Rebuild plugin with the same Go version as Crush

---

```
Error: cannot find plugin at path
```

**Solution**: Check the path in `plugins` config is correct and `.so` file exists

### Hook not executing

1. Check hook is registered via `SetHooks()`
2. Verify correct interface implementation
3. Enable debug logging: `CRUSH_LOG_LEVEL=debug`

### Tool not appearing

1. Implement `ToolProvider` interface
2. Add tools via `AddTool()`
3. Check tool JSON schema is valid

## Roadmap

Future enhancements:

- [ ] Plugin marketplace/registry
- [ ] WASM plugin support (for portability)
- [ ] gRPC plugins (for cross-language support)
- [ ] Plugin dependency management
- [ ] Hot reload support
- [ ] Plugin sandboxing
- [ ] Plugin version compatibility checking
- [ ] Built-in plugin testing framework

## Resources

- **[Plugin Development Guide](./PLUGIN_DEVELOPMENT.md)** - Complete guide
- **SDK Source**: `pkg/crushsdk/`
- **Internal Implementation**: `internal/plugin/`
- **Examples**: `examples/plugins/`

## Contributing

We welcome plugin contributions! To share your plugin:

1. Create a GitHub repository
2. Add README with build/usage instructions
3. Tag releases
4. Open a PR to add to community plugins list

## Community Plugins

(Coming soon - submit yours!)

- **Your Plugin Here** - Submit a PR!

## Credits

The Crush plugin system was inspired by:
- **OpenCode's plugin architecture** - Hook system and extensibility patterns
- **VSCode Extensions** - Tool and command registration
- **Terraform Providers** - Go plugin loading approach

## License

The plugin system is part of Crush and follows Crush's license (FSL-1.1-MIT).

Individual plugins can use any license chosen by their authors.
