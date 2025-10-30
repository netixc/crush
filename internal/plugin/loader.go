package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"strings"

	"github.com/charmbracelet/crush/internal/config"
)

// Loader handles loading plugins from various sources
type Loader struct {
	registry *Registry
}

// NewLoader creates a new plugin loader
func NewLoader(registry *Registry) *Loader {
	return &Loader{
		registry: registry,
	}
}

// LoadFromPath loads a plugin from a file path.
// Supports:
//   - .so files (Go plugins compiled with -buildmode=plugin)
//   - Directories containing a .so file
func (l *Loader) LoadFromPath(ctx context.Context, path string, pluginCtx PluginContext) error {
	// Resolve the path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve plugin path: %w", err)
	}

	// Check if path exists
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("plugin path does not exist: %w", err)
	}

	var pluginPath string
	if info.IsDir() {
		// Look for .so file in directory
		pluginPath, err = l.findPluginInDir(absPath)
		if err != nil {
			return err
		}
	} else {
		pluginPath = absPath
	}

	// Validate it's a .so file
	if !strings.HasSuffix(pluginPath, ".so") {
		return fmt.Errorf("plugin must be a .so file, got: %s", pluginPath)
	}

	// Load the plugin
	return l.loadGoPlugin(ctx, pluginPath, pluginCtx)
}

// findPluginInDir finds the first .so file in a directory
func (l *Loader) findPluginInDir(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("failed to read plugin directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".so") {
			return filepath.Join(dir, entry.Name()), nil
		}
	}

	return "", fmt.Errorf("no .so file found in directory: %s", dir)
}

// loadGoPlugin loads a Go plugin (.so file)
func (l *Loader) loadGoPlugin(ctx context.Context, path string, pluginCtx PluginContext) error {
	// Open the plugin
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open plugin: %w", err)
	}

	// Look for the exported "Plugin" symbol
	symbol, err := p.Lookup("Plugin")
	if err != nil {
		return fmt.Errorf("plugin does not export 'Plugin' symbol: %w", err)
	}

	// Assert that it implements the Plugin interface
	pluginImpl, ok := symbol.(Plugin)
	if !ok {
		return fmt.Errorf("Plugin symbol does not implement plugin.Plugin interface")
	}

	// Load the plugin into the registry
	if err := l.registry.LoadPlugin(ctx, pluginImpl, pluginCtx); err != nil {
		return fmt.Errorf("failed to load plugin: %w", err)
	}

	return nil
}

// LoadFromConfig loads all plugins specified in the configuration
func (l *Loader) LoadFromConfig(ctx context.Context, cfg *config.Config, pluginCtx PluginContext) error {
	// Get plugin paths from config
	pluginPaths := cfg.GetPluginPaths()

	for _, path := range pluginPaths {
		if err := l.LoadFromPath(ctx, path, pluginCtx); err != nil {
			// Log error but continue loading other plugins
			fmt.Fprintf(os.Stderr, "Warning: failed to load plugin from %s: %v\n", path, err)
			continue
		}
	}

	return nil
}
