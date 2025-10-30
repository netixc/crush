package plugin

import (
	"context"
	"fmt"
	"sync"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/csync"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/permission"
	"github.com/charmbracelet/crush/internal/session"
)

// Registry manages all loaded plugins and their hooks.
// It provides methods to load plugins, register hooks, and trigger hook execution.
type Registry struct {
	plugins      *csync.Map[string, Plugin]
	configHooks  []ConfigHook
	sessionHooks []SessionHook
	messageHooks []MessageHook
	permHooks    []PermissionHook
	toolHooks    []ToolHook
	agentHooks   []AgentHook
	mu           sync.RWMutex
}

// NewRegistry creates a new plugin registry
func NewRegistry() *Registry {
	return &Registry{
		plugins:      csync.NewMap[string, Plugin](),
		configHooks:  make([]ConfigHook, 0),
		sessionHooks: make([]SessionHook, 0),
		messageHooks: make([]MessageHook, 0),
		permHooks:    make([]PermissionHook, 0),
		toolHooks:    make([]ToolHook, 0),
		agentHooks:   make([]AgentHook, 0),
	}
}

// LoadPlugin loads a plugin and registers its hooks.
// The plugin is initialized with the provided context.
func (r *Registry) LoadPlugin(ctx context.Context, plugin Plugin, pluginCtx PluginContext) error {
	info := plugin.Info()

	// Check if plugin is already loaded
	if _, exists := r.plugins.Load(info.Name); exists {
		return fmt.Errorf("plugin %s is already loaded", info.Name)
	}

	// Initialize the plugin
	if err := plugin.Init(ctx, pluginCtx); err != nil {
		return fmt.Errorf("failed to initialize plugin %s: %w", info.Name, err)
	}

	// Register the plugin
	r.plugins.Store(info.Name, plugin)

	// Register all hooks
	hooks := plugin.Hooks()
	r.registerHooks(hooks)

	return nil
}

// registerHooks registers all hooks from a plugin
func (r *Registry) registerHooks(hooks Hooks) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if configHook := hooks.Config(); configHook != nil {
		r.configHooks = append(r.configHooks, configHook)
	}

	if sessionHook := hooks.Session(); sessionHook != nil {
		r.sessionHooks = append(r.sessionHooks, sessionHook)
	}

	if messageHook := hooks.Message(); messageHook != nil {
		r.messageHooks = append(r.messageHooks, messageHook)
	}

	if permHook := hooks.Permission(); permHook != nil {
		r.permHooks = append(r.permHooks, permHook)
	}

	if toolHook := hooks.Tool(); toolHook != nil {
		r.toolHooks = append(r.toolHooks, toolHook)
	}

	if agentHook := hooks.Agent(); agentHook != nil {
		r.agentHooks = append(r.agentHooks, agentHook)
	}
}

// UnloadPlugin unloads a plugin by name
func (r *Registry) UnloadPlugin(ctx context.Context, name string) error {
	plugin, exists := r.plugins.Load(name)
	if !exists {
		return fmt.Errorf("plugin %s is not loaded", name)
	}

	// Shutdown the plugin
	if err := plugin.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown plugin %s: %w", name, err)
	}

	// Remove from registry
	r.plugins.Delete(name)

	// Note: We don't remove hooks here because it would require rebuilding
	// the hook arrays. In practice, plugins are loaded once at startup.
	// For dynamic plugin loading/unloading, we'd need a more sophisticated approach.

	return nil
}

// GetPlugin retrieves a loaded plugin by name
func (r *Registry) GetPlugin(name string) (Plugin, bool) {
	return r.plugins.Load(name)
}

// ListPlugins returns a list of all loaded plugins
func (r *Registry) ListPlugins() []PluginInfo {
	var infos []PluginInfo
	r.plugins.Range(func(_ string, plugin Plugin) bool {
		infos = append(infos, plugin.Info())
		return true
	})
	return infos
}

// Shutdown shuts down all loaded plugins
func (r *Registry) Shutdown(ctx context.Context) error {
	var errors []error
	r.plugins.Range(func(name string, plugin Plugin) bool {
		if err := plugin.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("plugin %s: %w", name, err))
		}
		return true
	})

	if len(errors) > 0 {
		return fmt.Errorf("failed to shutdown %d plugin(s): %v", len(errors), errors)
	}

	return nil
}

// Hook Trigger Methods
// These methods trigger all registered hooks of a specific type in sequence.

// TriggerConfigHooks triggers all config hooks
func (r *Registry) TriggerConfigHooks(ctx context.Context, cfg *config.Config) error {
	r.mu.RLock()
	hooks := make([]ConfigHook, len(r.configHooks))
	copy(hooks, r.configHooks)
	r.mu.RUnlock()

	for _, hook := range hooks {
		if err := hook.OnConfigLoad(ctx, cfg); err != nil {
			return fmt.Errorf("config hook failed: %w", err)
		}
	}
	return nil
}

// TriggerSessionCreated triggers all session created hooks
func (r *Registry) TriggerSessionCreated(ctx context.Context, sess session.Session) error {
	r.mu.RLock()
	hooks := make([]SessionHook, len(r.sessionHooks))
	copy(hooks, r.sessionHooks)
	r.mu.RUnlock()

	for _, hook := range hooks {
		if err := hook.OnSessionCreated(ctx, sess); err != nil {
			return fmt.Errorf("session created hook failed: %w", err)
		}
	}
	return nil
}

// TriggerSessionUpdated triggers all session updated hooks
func (r *Registry) TriggerSessionUpdated(ctx context.Context, sess session.Session) error {
	r.mu.RLock()
	hooks := make([]SessionHook, len(r.sessionHooks))
	copy(hooks, r.sessionHooks)
	r.mu.RUnlock()

	for _, hook := range hooks {
		if err := hook.OnSessionUpdated(ctx, sess); err != nil {
			return fmt.Errorf("session updated hook failed: %w", err)
		}
	}
	return nil
}

// TriggerSessionDeleted triggers all session deleted hooks
func (r *Registry) TriggerSessionDeleted(ctx context.Context, sessionID string) error {
	r.mu.RLock()
	hooks := make([]SessionHook, len(r.sessionHooks))
	copy(hooks, r.sessionHooks)
	r.mu.RUnlock()

	for _, hook := range hooks {
		if err := hook.OnSessionDeleted(ctx, sessionID); err != nil {
			return fmt.Errorf("session deleted hook failed: %w", err)
		}
	}
	return nil
}

// TriggerMessageCreated triggers all message created hooks
func (r *Registry) TriggerMessageCreated(ctx context.Context, msg message.Message) error {
	r.mu.RLock()
	hooks := make([]MessageHook, len(r.messageHooks))
	copy(hooks, r.messageHooks)
	r.mu.RUnlock()

	for _, hook := range hooks {
		if err := hook.OnMessageCreated(ctx, msg); err != nil {
			return fmt.Errorf("message created hook failed: %w", err)
		}
	}
	return nil
}

// TriggerMessageUpdated triggers all message updated hooks
func (r *Registry) TriggerMessageUpdated(ctx context.Context, msg message.Message) error {
	r.mu.RLock()
	hooks := make([]MessageHook, len(r.messageHooks))
	copy(hooks, r.messageHooks)
	r.mu.RUnlock()

	for _, hook := range hooks {
		if err := hook.OnMessageUpdated(ctx, msg); err != nil {
			return fmt.Errorf("message updated hook failed: %w", err)
		}
	}
	return nil
}

// TriggerPermissionRequest triggers all permission request hooks.
// Returns the first non-nil decision, or nil if all hooks return nil.
func (r *Registry) TriggerPermissionRequest(ctx context.Context, req permission.CreatePermissionRequest) (*bool, error) {
	r.mu.RLock()
	hooks := make([]PermissionHook, len(r.permHooks))
	copy(hooks, r.permHooks)
	r.mu.RUnlock()

	for _, hook := range hooks {
		decision, err := hook.OnPermissionRequest(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("permission hook failed: %w", err)
		}
		// Return the first non-nil decision
		if decision != nil {
			return decision, nil
		}
	}
	return nil, nil
}

// TriggerToolExecuteBefore triggers all tool execute before hooks.
// Each hook can modify the arguments, and the modifications are passed to the next hook.
func (r *Registry) TriggerToolExecuteBefore(ctx context.Context, input ToolExecuteInput) (map[string]any, error) {
	r.mu.RLock()
	hooks := make([]ToolHook, len(r.toolHooks))
	copy(hooks, r.toolHooks)
	r.mu.RUnlock()

	args := input.Arguments
	for _, hook := range hooks {
		modifiedArgs, err := hook.OnToolExecuteBefore(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("tool execute before hook failed: %w", err)
		}
		// Apply modifications if returned
		if modifiedArgs != nil {
			args = modifiedArgs
			// Update input for next hook
			input.Arguments = args
		}
	}
	return args, nil
}

// TriggerToolExecuteAfter triggers all tool execute after hooks.
// Each hook can modify the result, and the modifications are passed to the next hook.
func (r *Registry) TriggerToolExecuteAfter(ctx context.Context, input ToolExecuteInput, result ToolExecuteResult) (ToolExecuteResult, error) {
	r.mu.RLock()
	hooks := make([]ToolHook, len(r.toolHooks))
	copy(hooks, r.toolHooks)
	r.mu.RUnlock()

	for _, hook := range hooks {
		modifiedResult, err := hook.OnToolExecuteAfter(ctx, input, result)
		if err != nil {
			return result, fmt.Errorf("tool execute after hook failed: %w", err)
		}
		// Apply modifications if returned
		if modifiedResult != nil {
			result = *modifiedResult
		}
	}
	return result, nil
}

// TriggerAgentStart triggers all agent start hooks
func (r *Registry) TriggerAgentStart(ctx context.Context, input AgentStartInput) error {
	r.mu.RLock()
	hooks := make([]AgentHook, len(r.agentHooks))
	copy(hooks, r.agentHooks)
	r.mu.RUnlock()

	for _, hook := range hooks {
		if err := hook.OnAgentStart(ctx, input); err != nil {
			return fmt.Errorf("agent start hook failed: %w", err)
		}
	}
	return nil
}

// TriggerAgentStep triggers all agent step hooks
func (r *Registry) TriggerAgentStep(ctx context.Context, input AgentStepInput) error {
	r.mu.RLock()
	hooks := make([]AgentHook, len(r.agentHooks))
	copy(hooks, r.agentHooks)
	r.mu.RUnlock()

	for _, hook := range hooks {
		if err := hook.OnAgentStep(ctx, input); err != nil {
			return fmt.Errorf("agent step hook failed: %w", err)
		}
	}
	return nil
}

// TriggerAgentFinish triggers all agent finish hooks
func (r *Registry) TriggerAgentFinish(ctx context.Context, input AgentFinishInput) error {
	r.mu.RLock()
	hooks := make([]AgentHook, len(r.agentHooks))
	copy(hooks, r.agentHooks)
	r.mu.RUnlock()

	for _, hook := range hooks {
		if err := hook.OnAgentFinish(ctx, input); err != nil {
			return fmt.Errorf("agent finish hook failed: %w", err)
		}
	}
	return nil
}
