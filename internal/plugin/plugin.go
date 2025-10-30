package plugin

import (
	"context"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/permission"
	"github.com/charmbracelet/crush/internal/session"
)

// Plugin is the main interface that all plugins must implement.
// Plugins are loaded during application initialization and can register
// hooks to customize behavior across the application lifecycle.
type Plugin interface {
	// Info returns metadata about the plugin
	Info() PluginInfo

	// Init is called when the plugin is loaded, before any hooks are registered.
	// The plugin should use this to perform any necessary initialization.
	Init(ctx context.Context, pluginCtx PluginContext) error

	// Hooks returns the hook implementations provided by this plugin.
	// Returning nil for any hook means the plugin doesn't implement it.
	Hooks() Hooks

	// Shutdown is called when the application is shutting down.
	// The plugin should clean up any resources it has allocated.
	Shutdown(ctx context.Context) error
}

// PluginInfo contains metadata about a plugin
type PluginInfo struct {
	// Name is the unique identifier for the plugin
	Name string

	// Version is the semantic version of the plugin
	Version string

	// Description is a human-readable description of what the plugin does
	Description string

	// Author is the plugin author or organization
	Author string
}

// PluginContext provides plugins with access to application services and state.
// This context is passed to plugins during initialization and allows them to
// interact with the core application.
type PluginContext struct {
	// Config is the application configuration
	Config *config.Config

	// Services provides access to core application services
	Services Services

	// WorkingDir is the current working directory
	WorkingDir string
}

// Services provides access to core application services that plugins can use
type Services struct {
	// Session service for managing conversation sessions
	Session session.Service

	// Message service for managing messages within sessions
	Message message.Service

	// Permission service for permission requests
	Permission permission.Service
}

// Hooks defines all available hook points that plugins can implement.
// Each hook is optional - plugins only need to implement the hooks they need.
type Hooks interface {
	// Config hooks are called during configuration loading and allow
	// plugins to modify the configuration before it's used.
	Config() ConfigHook

	// Session hooks are called during session lifecycle events
	Session() SessionHook

	// Message hooks are called during message lifecycle events
	Message() MessageHook

	// Permission hooks are called during permission request handling
	Permission() PermissionHook

	// Tool hooks are called before and after tool execution
	Tool() ToolHook

	// Agent hooks are called during agent execution lifecycle
	Agent() AgentHook
}

// ConfigHook allows plugins to modify configuration during loading
type ConfigHook interface {
	// OnConfigLoad is called after the config is loaded from files but
	// before it's used. Plugins can modify the config in place.
	OnConfigLoad(ctx context.Context, cfg *config.Config) error
}

// SessionHook provides hooks for session lifecycle events
type SessionHook interface {
	// OnSessionCreated is called after a new session is created
	OnSessionCreated(ctx context.Context, sess session.Session) error

	// OnSessionUpdated is called after a session is updated
	OnSessionUpdated(ctx context.Context, sess session.Session) error

	// OnSessionDeleted is called after a session is deleted
	OnSessionDeleted(ctx context.Context, sessionID string) error
}

// MessageHook provides hooks for message lifecycle events
type MessageHook interface {
	// OnMessageCreated is called after a new message is created
	OnMessageCreated(ctx context.Context, msg message.Message) error

	// OnMessageUpdated is called after a message is updated
	OnMessageUpdated(ctx context.Context, msg message.Message) error
}

// PermissionHook provides hooks for permission request handling
type PermissionHook interface {
	// OnPermissionRequest is called when a permission request is made,
	// before prompting the user. The plugin can modify the decision to
	// auto-approve or auto-deny the request.
	//
	// Return values:
	//   - decision: pointer to bool (nil = no decision, true = allow, false = deny)
	//   - error: if non-nil, the permission request fails
	OnPermissionRequest(ctx context.Context, req permission.CreatePermissionRequest) (*bool, error)
}

// ToolHook provides hooks for tool execution
type ToolHook interface {
	// OnToolExecuteBefore is called before a tool is executed.
	// The plugin can modify the input arguments by returning a modified map.
	// Returning nil means no modifications.
	OnToolExecuteBefore(ctx context.Context, input ToolExecuteInput) (map[string]any, error)

	// OnToolExecuteAfter is called after a tool has executed.
	// The plugin can modify the tool result by returning a modified result.
	// Returning nil means no modifications.
	OnToolExecuteAfter(ctx context.Context, input ToolExecuteInput, result ToolExecuteResult) (*ToolExecuteResult, error)
}

// ToolExecuteInput contains information about a tool execution
type ToolExecuteInput struct {
	// ToolName is the name of the tool being executed
	ToolName string

	// SessionID is the ID of the session in which the tool is being executed
	SessionID string

	// MessageID is the ID of the message associated with the tool call
	MessageID string

	// ToolCallID is the unique ID of this specific tool call
	ToolCallID string

	// Arguments are the input arguments to the tool (as JSON-serializable map)
	Arguments map[string]any
}

// ToolExecuteResult contains the result of a tool execution
type ToolExecuteResult struct {
	// Output is the text output from the tool
	Output string

	// Error is any error that occurred during tool execution
	Error error

	// Metadata contains additional metadata about the execution
	Metadata map[string]any
}

// AgentHook provides hooks for agent execution lifecycle
type AgentHook interface {
	// OnAgentStart is called when an agent starts processing a prompt
	OnAgentStart(ctx context.Context, input AgentStartInput) error

	// OnAgentStep is called after each step of agent execution
	OnAgentStep(ctx context.Context, input AgentStepInput) error

	// OnAgentFinish is called when an agent completes execution
	OnAgentFinish(ctx context.Context, input AgentFinishInput) error
}

// AgentStartInput contains information about an agent starting execution
type AgentStartInput struct {
	// SessionID is the ID of the session
	SessionID string

	// Prompt is the user's prompt
	Prompt string

	// Model is the model being used
	Model string

	// Provider is the provider being used
	Provider string
}

// AgentStepInput contains information about an agent step
type AgentStepInput struct {
	// SessionID is the ID of the session
	SessionID string

	// StepNumber is the current step number
	StepNumber int

	// ToolCalls are the tool calls made in this step
	ToolCalls []fantasy.ToolCallContent

	// Response is the agent's text response in this step
	Response string
}

// AgentFinishInput contains information about an agent completing execution
type AgentFinishInput struct {
	// SessionID is the ID of the session
	SessionID string

	// TotalSteps is the total number of steps executed
	TotalSteps int

	// Result is the final agent result
	Result *fantasy.AgentResult

	// Error is any error that occurred during execution
	Error error
}

// NilConfigHook implements ConfigHook with no-op methods
type NilConfigHook struct{}

func (n NilConfigHook) OnConfigLoad(ctx context.Context, cfg *config.Config) error { return nil }

// NilSessionHook implements SessionHook with no-op methods
type NilSessionHook struct{}

func (n NilSessionHook) OnSessionCreated(ctx context.Context, sess session.Session) error {
	return nil
}
func (n NilSessionHook) OnSessionUpdated(ctx context.Context, sess session.Session) error {
	return nil
}
func (n NilSessionHook) OnSessionDeleted(ctx context.Context, sessionID string) error { return nil }

// NilMessageHook implements MessageHook with no-op methods
type NilMessageHook struct{}

func (n NilMessageHook) OnMessageCreated(ctx context.Context, msg message.Message) error { return nil }
func (n NilMessageHook) OnMessageUpdated(ctx context.Context, msg message.Message) error { return nil }

// NilPermissionHook implements PermissionHook with no-op methods
type NilPermissionHook struct{}

func (n NilPermissionHook) OnPermissionRequest(ctx context.Context, req permission.CreatePermissionRequest) (*bool, error) {
	return nil, nil
}

// NilToolHook implements ToolHook with no-op methods
type NilToolHook struct{}

func (n NilToolHook) OnToolExecuteBefore(ctx context.Context, input ToolExecuteInput) (map[string]any, error) {
	return nil, nil
}
func (n NilToolHook) OnToolExecuteAfter(ctx context.Context, input ToolExecuteInput, result ToolExecuteResult) (*ToolExecuteResult, error) {
	return nil, nil
}

// NilAgentHook implements AgentHook with no-op methods
type NilAgentHook struct{}

func (n NilAgentHook) OnAgentStart(ctx context.Context, input AgentStartInput) error    { return nil }
func (n NilAgentHook) OnAgentStep(ctx context.Context, input AgentStepInput) error      { return nil }
func (n NilAgentHook) OnAgentFinish(ctx context.Context, input AgentFinishInput) error  { return nil }

// BaseHooks provides default no-op implementations for all hooks.
// Plugins can embed this to only implement the hooks they need.
type BaseHooks struct {
	ConfigHook     ConfigHook
	SessionHook    SessionHook
	MessageHook    MessageHook
	PermissionHook PermissionHook
	ToolHook       ToolHook
	AgentHook      AgentHook
}

func (b *BaseHooks) Config() ConfigHook         { return b.ConfigHook }
func (b *BaseHooks) Session() SessionHook       { return b.SessionHook }
func (b *BaseHooks) Message() MessageHook       { return b.MessageHook }
func (b *BaseHooks) Permission() PermissionHook { return b.PermissionHook }
func (b *BaseHooks) Tool() ToolHook             { return b.ToolHook }
func (b *BaseHooks) Agent() AgentHook           { return b.AgentHook }

// NewBaseHooks creates a new BaseHooks with all nil implementations
func NewBaseHooks() *BaseHooks {
	return &BaseHooks{
		ConfigHook:     NilConfigHook{},
		SessionHook:    NilSessionHook{},
		MessageHook:    NilMessageHook{},
		PermissionHook: NilPermissionHook{},
		ToolHook:       NilToolHook{},
		AgentHook:      NilAgentHook{},
	}
}
