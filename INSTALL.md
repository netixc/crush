# Crush Installation & Development Guide

Complete guide for installing, building, and running Crush from source in development mode.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Development Mode](#development-mode)
- [Configuration](#configuration)
- [Database](#database)
- [Development Workflow](#development-workflow)
- [Testing](#testing)
- [Plugin System](#plugin-system)
- [Troubleshooting](#troubleshooting)
- [Environment Variables](#environment-variables)
- [Project Structure](#project-structure)

---

## Prerequisites

### Required

- **Go 1.25.0 or later** (specified in `go.mod`)
  ```bash
  go version  # Should show go1.25.0 or higher
  ```

- **Git** (for cloning the repository)

### Optional Development Tools

- **Task** - Task runner (recommended but not required)
  ```bash
  go install github.com/go-task/task/v3/cmd/task@latest
  ```

- **golangci-lint** - Code linter
  ```bash
  # Install via Task
  task lint:install
  # Or manually from https://golangci-lint.run/usage/install/
  ```

- **gofumpt** - Stricter formatter than gofmt
  ```bash
  go install mvdan.cc/gofumpt@latest
  ```

---

## Quick Start

### 1. Clone or Navigate to Repository

```bash
cd /path/to/crush
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Build Crush

```bash
# Option A: Using Go
go build .

# Option B: Using Task (if installed)
task build
```

This creates a `./crush` executable in the current directory.

### 4. Run Crush

```bash
# Interactive mode (TUI)
./crush

# Or run directly without building
go run .
```

### 5. Configure (Optional but Recommended)

Create a config file with your AI provider:

```bash
cat > .crush.json <<EOF
{
  "providers": {
    "openai": {
      "type": "openai",
      "api_key": "$OPENAI_API_KEY"
    }
  }
}
EOF
```

**Important**: Set your API key:
```bash
export OPENAI_API_KEY="your-api-key-here"
```

### 6. Test It Works

```bash
# Run a simple command
./crush run "hello, who are you?"

# Or enter interactive mode
./crush
```

---

## Development Mode

### Running Without Building

```bash
# Run directly with Go
go run .

# Using Task
task run
```

### Debug Mode

```bash
# Enable debug logging
go run . --debug

# Shorthand
go run . -d
```

Debug logs show:
- Plugin loading
- Configuration parsing
- Tool execution
- LSP/MCP interactions

### Profiling Mode

```bash
# Using Task (recommended)
task dev

# Or manually
CRUSH_PROFILE=true go run .
```

When profiling is enabled, pprof endpoints are available at `http://localhost:6060`:

- **CPU Profile**: http://localhost:6060/debug/pprof/profile?seconds=10
- **Heap Profile**: http://localhost:6060/debug/pprof/heap
- **Goroutines**: http://localhost:6060/debug/pprof/goroutine
- **All Profiles**: http://localhost:6060/debug/pprof/

**Analyze a profile:**
```bash
# Capture CPU profile
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof

# View with pprof
go tool pprof cpu.prof
```

### Non-Interactive Mode

```bash
# Run a single command
go run . run "explain the main.go file"

# With custom working directory
go run . -c /path/to/project run "analyze this code"
```

### Custom Directories

```bash
# Use custom working directory
go run . -c /path/to/project

# Use custom data directory (for .crush folder)
go run . -D /path/to/custom/.crush

# Both together
go run . -c /path/to/project -D /tmp/.crush
```

### Other Useful Flags

```bash
# Yolo mode - auto-accept all permissions (DANGEROUS!)
go run . -y

# View logs
go run . logs

# Update provider database
go run . update-providers
```

---

## Configuration

### Config File Locations

Crush searches for configuration files in this order (first found wins):

1. `./.crush.json` - Project-specific (highest priority)
2. `./crush.json` - Project-specific
3. `~/.config/crush/crush.json` - User global (Unix/macOS)
4. `%LOCALAPPDATA%\crush\crush.json` - User global (Windows)

### Minimal Configuration

```json
{
  "providers": {
    "openai": {
      "type": "openai",
      "api_key": "$OPENAI_API_KEY"
    }
  }
}
```

### Full Configuration Example

```json
{
  "$schema": "https://charm.land/crush.json",
  "models": {
    "large": {
      "model": "gpt-4o",
      "provider": "openai"
    },
    "small": {
      "model": "gpt-4o-mini",
      "provider": "openai"
    }
  },
  "providers": {
    "openai": {
      "type": "openai",
      "base_url": "https://api.openai.com/v1",
      "api_key": "$OPENAI_API_KEY"
    },
    "anthropic": {
      "type": "anthropic",
      "api_key": "$ANTHROPIC_API_KEY"
    }
  },
  "lsp": {
    "gopls": {
      "command": "gopls",
      "filetypes": ["go"],
      "enabled": true
    },
    "typescript-language-server": {
      "command": "typescript-language-server",
      "args": ["--stdio"],
      "filetypes": ["typescript", "javascript"]
    }
  },
  "mcp": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/allowed/directory"],
      "type": "stdio"
    }
  },
  "options": {
    "debug": false,
    "context_paths": [".cursorrules", "CLAUDE.md", "CRUSH.md"],
    "disabled_tools": []
  },
  "plugins": [
    "./path/to/plugin.so"
  ]
}
```

### Auto-Loaded Context Files

Crush automatically loads context from these files (if they exist):

```
.github/copilot-instructions.md
.cursorrules
.cursor/rules/
CLAUDE.md, CLAUDE.local.md
GEMINI.md, gemini.md
crush.md, crush.local.md
Crush.md, Crush.local.md
CRUSH.md, CRUSH.local.md
AGENTS.md, agents.md, Agents.md
```

### Generating JSON Schema

```bash
# Generate schema.json file
task schema

# Or use it in your editor
# In VS Code: add "$schema": "https://charm.land/crush.json"
```

---

## Database

### Location

- **Default**: `./.crush/crush.db` (in your current working directory)
- **Custom**: Use `--data-dir` flag to specify a different location

### Schema

The database is **automatically created** and **migrations run automatically** on first launch.

**Tables:**
- `sessions` - Chat sessions with metadata (title, tokens, cost, timestamps)
- `messages` - Individual messages in conversations
- `files` - File snapshots associated with sessions

### Database Configuration

SQLite is configured with optimal settings automatically:

```sql
PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;        -- Write-Ahead Logging
PRAGMA page_size = 4096;
PRAGMA cache_size = -8000;        -- 8MB cache
PRAGMA synchronous = NORMAL;
PRAGMA secure_delete = ON;
```

### Viewing the Database

```bash
# Using sqlite3 CLI
sqlite3 .crush/crush.db

# View sessions
sqlite> SELECT id, title, created_at FROM sessions;

# View messages
sqlite> SELECT id, session_id, role FROM messages LIMIT 5;
```

### Reset Database

```bash
# Delete the database (it will be recreated on next run)
rm -rf .crush/crush.db

# Or delete entire data directory
rm -rf .crush/
```

---

## Development Workflow

### Using Task (Recommended)

```bash
# Build
task build

# Run
task run

# Run with profiling
task dev

# Run tests
task test

# Run tests with custom flags
task test -- -v -race

# Lint code
task lint

# Lint and auto-fix
task lint:fix

# Format code
task fmt

# Install Crush to $GOPATH/bin
task install
```

### Without Task (Direct Go Commands)

```bash
# Build
go build .

# Run
go run .

# Test
go test ./...

# Test specific package
go test ./internal/agent

# Test with coverage
go test -cover ./...

# Format
gofumpt -w .

# Lint
golangci-lint run
```

### Code Style

Crush follows these conventions:

- **Formatting**: Use `gofumpt` (stricter than `gofmt`)
- **Imports**: Grouped by stdlib, external, internal (via `goimports`)
- **JSON fields**: Use `snake_case` for JSON field names
- **File permissions**: Octal notation (`0o755`, `0o644`)
- **Commit messages**: Semantic commits (`feat:`, `fix:`, `chore:`, `docs:`, etc.)

---

## Testing

### Run All Tests

```bash
# Using Task
task test

# Direct
go test ./...

# With verbose output
go test -v ./...

# With race detection
go test -race ./...

# With coverage
go test -cover ./...
```

### Run Specific Tests

```bash
# Test single package
go test ./internal/config

# Test with pattern
go test ./... -run TestConfigLoad

# Update golden files (snapshot tests)
go test ./internal/tui/components/core -update
```

### Test Configuration

Tests use:
- **Testify** for assertions (`github.com/stretchr/testify`)
- **Table-driven tests** for multiple scenarios
- **Golden files** for snapshot testing (TUI components)
- **Parallel execution** where appropriate (`t.Parallel()`)

---

## Plugin System

Crush now has a powerful plugin system (newly added)! Plugins allow you to extend Crush with custom tools, hooks, and behavior.

### Building Example Plugins

```bash
# Navigate to plugin directory
cd examples/plugins/hello-world

# Build the plugin
go build -buildmode=plugin -o hello-world.so main.go

# Return to Crush root
cd ../../..
```

### Available Example Plugins

1. **hello-world** - Adds a custom "hello" tool
   ```bash
   cd examples/plugins/hello-world
   go build -buildmode=plugin -o hello-world.so main.go
   ```

2. **auto-approve** - Auto-approves read-only tool permissions
   ```bash
   cd examples/plugins/auto-approve
   go build -buildmode=plugin -o auto-approve.so main.go
   ```

3. **metrics** - Collects and reports usage metrics
   ```bash
   cd examples/plugins/metrics
   go build -buildmode=plugin -o metrics.so main.go
   ```

### Using Plugins

**1. Build the plugin:**
```bash
cd examples/plugins/hello-world
go build -buildmode=plugin -o hello-world.so main.go
cd ../../..
```

**2. Configure Crush to load it:**
```json
{
  "plugins": [
    "./examples/plugins/hello-world/hello-world.so"
  ],
  "providers": {
    "openai": {
      "type": "openai",
      "api_key": "$OPENAI_API_KEY"
    }
  }
}
```

**3. Run Crush with debug to see plugin loading:**
```bash
go run . --debug
```

You should see:
```
INFO Plugins initialized count=1
```

**4. Test the plugin:**

In the TUI, you can now use the custom tool. For example, with hello-world plugin:
```
> use the hello tool to greet "Alice"
```

### Creating Your Own Plugin

See the comprehensive plugin development guide:
- **[Plugin Development Guide](docs/PLUGIN_DEVELOPMENT.md)** - Complete tutorial
- **[Plugin System Overview](docs/PLUGINS_README.md)** - Architecture and examples

**Quick start:**

```go
package main

import (
    "context"
    "github.com/charmbracelet/crush/pkg/crushsdk"
    "github.com/charmbracelet/fantasy"
)

var Plugin crushsdk.Plugin = &MyPlugin{}

type MyPlugin struct {
    *crushsdk.SimplePlugin
}

func init() {
    base := crushsdk.NewSimplePlugin(crushsdk.PluginInfo{
        Name:        "my-plugin",
        Version:     "1.0.0",
        Description: "My awesome plugin",
        Author:      "Your Name",
    })

    // Add custom tool
    tool := crushsdk.NewSimpleTool(
        "my-tool",
        "Does something useful",
        map[string]any{
            "input": map[string]any{
                "type": "string",
                "description": "Input data",
            },
        },
        []string{"input"},
        func(ctx context.Context, params fantasy.ToolCall) (fantasy.ToolResponse, error) {
            return fantasy.ToolResponse{Text: "Hello!"}, nil
        },
    )

    base.AddTool(tool)
    Plugin = &MyPlugin{SimplePlugin: base}
}
```

**Build and use:**
```bash
go build -buildmode=plugin -o my-plugin.so main.go
```

---

## Troubleshooting

### Plugin Won't Load

**Error: `plugin does not export 'Plugin' symbol`**

**Solution**: Ensure you have this at package level:
```go
var Plugin crushsdk.Plugin = &YourPlugin{}
```

---

**Error: `plugin was built with a different version of package`**

**Solution**: Rebuild plugin with the **exact same Go version** as Crush:
```bash
go version  # Check Crush's Go version
cd your-plugin
go build -buildmode=plugin -o plugin.so main.go
```

---

**Error: `cannot find plugin at path`**

**Solution**:
- Check the path in `plugins` config is correct
- Use absolute path or relative to working directory
- Ensure `.so` file exists: `ls -la path/to/plugin.so`

### Build Errors

**Error: `go: cannot find main module`**

**Solution**: Ensure you're in the Crush root directory:
```bash
cd /path/to/crush
ls go.mod  # Should exist
```

---

**Error: `package X is not in std`**

**Solution**: Update dependencies:
```bash
go mod download
go mod tidy
```

### Database Issues

**Error: `database is locked`**

**Solution**: Another Crush process is running. Close it or use a different data directory:
```bash
go run . -D /tmp/.crush-test
```

---

**Database is corrupted**

**Solution**: Delete and let it recreate:
```bash
rm -rf .crush/crush.db*
go run .
```

### Configuration Issues

**Config not being loaded**

**Solution**: Check config file location and syntax:
```bash
# Validate JSON
cat .crush.json | jq .

# Run with debug to see config loading
go run . --debug
```

---

**Environment variables not expanding**

**Solution**: Ensure they're set and use `$VAR` syntax:
```bash
export OPENAI_API_KEY="sk-..."
echo $OPENAI_API_KEY  # Should print the key
```

### Runtime Issues

**Crash on startup**

**Solution**: Run with debug logging to see the error:
```bash
go run . --debug
```

---

**TUI rendering issues**

**Solution**: Check terminal compatibility:
```bash
# Set TERM if needed
export TERM=xterm-256color

# Try different terminal
# Some terminals work better than others
```

---

**LSP not working**

**Solution**: Ensure LSP server is installed:
```bash
# For Go
go install golang.org/x/tools/gopls@latest

# For TypeScript
npm install -g typescript-language-server

# Check it's in PATH
which gopls
```

---

## Environment Variables

### API Keys

```bash
# OpenAI
export OPENAI_API_KEY="sk-..."

# Anthropic
export ANTHROPIC_API_KEY="sk-ant-..."

# Google Gemini
export GEMINI_API_KEY="..."

# Groq
export GROQ_API_KEY="gsk_..."

# OpenRouter
export OPENROUTER_API_KEY="sk-or-..."

# AWS Bedrock
export AWS_ACCESS_KEY_ID="..."
export AWS_SECRET_ACCESS_KEY="..."
export AWS_REGION="us-east-1"

# Azure OpenAI
export AZURE_OPENAI_API_KEY="..."
export AZURE_OPENAI_ENDPOINT="https://..."
```

### Crush Configuration

```bash
# Enable profiling
export CRUSH_PROFILE=true

# Disable telemetry
export CRUSH_DISABLE_METRICS=1
export DO_NOT_TRACK=1  # Standard env var

# Disable provider auto-updates
export CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1

# Custom log level
export CRUSH_LOG_LEVEL=debug  # debug, info, warn, error
```

### Development

```bash
# Use custom Go toolchain
export GOTOOLCHAIN=go1.25.0

# Enable Go experiments
export GOEXPERIMENT=greenteagc

# Disable CGO (for static builds)
export CGO_ENABLED=0
```

---

## Project Structure

```
crush/
├── main.go                          # Entry point
├── go.mod, go.sum                  # Dependencies
├── Taskfile.yaml                   # Task runner config
├── .golangci.yml                   # Linter config
├── INSTALL.md                      # This file
├── README.md                       # Project README
├── CRUSH.md                        # Development docs
├── crush.json                      # Example config
│
├── internal/                       # Private packages
│   ├── cmd/                        # CLI commands
│   │   ├── root.go                # Root command
│   │   ├── run.go                 # Run command
│   │   └── logs.go                # Logs command
│   ├── config/                     # Configuration
│   │   ├── config.go              # Config types and loading
│   │   └── load.go                # Config file discovery
│   ├── db/                         # Database layer
│   │   ├── connect.go             # SQLite connection
│   │   ├── migrations/            # SQL migrations
│   │   └── sql/                   # SQL queries (sqlc)
│   ├── app/                        # Application logic
│   │   └── app.go                 # Main app struct
│   ├── agent/                      # AI agent
│   │   ├── agent.go               # Agent implementation
│   │   ├── coordinator.go         # Multi-agent coordinator
│   │   ├── prompt/                # System prompts
│   │   └── tools/                 # Built-in tools
│   ├── tui/                        # Terminal UI
│   │   ├── tui.go                 # Main TUI
│   │   └── components/            # UI components
│   ├── plugin/                     # Plugin system (NEW!)
│   │   ├── plugin.go              # Plugin interfaces
│   │   ├── registry.go            # Plugin registry
│   │   ├── tool.go                # Plugin tools
│   │   └── loader.go              # Plugin loader
│   ├── session/                    # Session management
│   ├── message/                    # Message management
│   ├── permission/                 # Permission system
│   └── lsp/                        # LSP integration
│
├── pkg/                            # Public packages
│   └── crushsdk/                   # Plugin SDK (NEW!)
│       └── sdk.go                  # Public SDK
│
├── examples/                       # Examples
│   └── plugins/                    # Example plugins (NEW!)
│       ├── hello-world/            # Basic plugin
│       ├── auto-approve/           # Permission plugin
│       └── metrics/                # Metrics plugin
│
├── docs/                           # Documentation
│   ├── PLUGIN_DEVELOPMENT.md       # Plugin dev guide
│   ├── PLUGINS_README.md           # Plugin system overview
│   └── PLUGIN_SYSTEM_IMPLEMENTATION.md
│
└── .crush/                         # Runtime data (created on first run)
    ├── crush.db                    # SQLite database
    ├── crush.db-shm               # Shared memory
    ├── crush.db-wal               # Write-ahead log
    └── logs/                       # Log files
        └── crush.log               # Application logs
```

---

## Additional Resources

- **Contributing Guide**: See [CRUSH.md](CRUSH.md)
- **Plugin Development**: See [docs/PLUGIN_DEVELOPMENT.md](docs/PLUGIN_DEVELOPMENT.md)
- **Plugin System Overview**: See [docs/PLUGINS_README.md](docs/PLUGINS_README.md)
- **Configuration Schema**: Generate with `task schema`
- **Charm Docs**: https://github.com/charmbracelet/crush

---

## Quick Reference

```bash
# Build and run
go build . && ./crush

# Dev mode with debug
go run . --debug

# Run tests
go test ./...

# Build plugin
cd examples/plugins/hello-world
go build -buildmode=plugin -o hello-world.so main.go

# Format and lint
task fmt && task lint:fix

# View logs
./crush logs

# Non-interactive
./crush run "your prompt here"
```

---

## Getting Help

- **GitHub Issues**: https://github.com/charmbracelet/crush/issues
- **Discussions**: https://github.com/charmbracelet/crush/discussions
- **Discord**: https://charm.sh/discord

---

**Happy coding! 🚀**
