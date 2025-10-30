package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"charm.land/fantasy"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/agent"
	"github.com/charmbracelet/crush/internal/agent/tools"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/csync"
	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/format"
	"github.com/charmbracelet/crush/internal/history"
	"github.com/charmbracelet/crush/internal/log"
	"github.com/charmbracelet/crush/internal/lsp"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/permission"
	"github.com/charmbracelet/crush/internal/plugin"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/charmbracelet/crush/internal/session"
	"github.com/charmbracelet/x/ansi"
)

type App struct {
	Sessions    session.Service
	Messages    message.Service
	History     history.Service
	Permissions permission.Service

	AgentCoordinator agent.Coordinator

	LSPClients     *csync.Map[string, *lsp.Client]
	PluginRegistry *plugin.Registry

	config *config.Config

	serviceEventsWG *sync.WaitGroup
	eventsCtx       context.Context
	events          chan tea.Msg
	tuiWG           *sync.WaitGroup

	// global context and cleanup functions
	globalCtx    context.Context
	cleanupFuncs []func() error
}

// New initializes a new applcation instance.
func New(ctx context.Context, conn *sql.DB, cfg *config.Config) (*App, error) {
	q := db.New(conn)
	sessions := session.NewService(q)
	messages := message.NewService(q)
	files := history.NewService(q, conn)
	skipPermissionsRequests := cfg.Permissions != nil && cfg.Permissions.SkipRequests
	allowedTools := []string{}
	if cfg.Permissions != nil && cfg.Permissions.AllowedTools != nil {
		allowedTools = cfg.Permissions.AllowedTools
	}

	app := &App{
		Sessions:       sessions,
		Messages:       messages,
		History:        files,
		Permissions:    permission.NewPermissionService(cfg.WorkingDir(), skipPermissionsRequests, allowedTools),
		LSPClients:     csync.NewMap[string, *lsp.Client](),
		PluginRegistry: plugin.NewRegistry(),

		globalCtx: ctx,

		config: cfg,

		events:          make(chan tea.Msg, 100),
		serviceEventsWG: &sync.WaitGroup{},
		tuiWG:           &sync.WaitGroup{},
	}

	app.setupEvents()

	// Initialize LSP clients in the background.
	app.initLSPClients(ctx)

	// Initialize plugins
	if err := app.initPlugins(ctx); err != nil {
		slog.Warn("Failed to initialize plugins", "error", err)
	}

	// cleanup database upon app shutdown
	app.cleanupFuncs = append(app.cleanupFuncs, conn.Close)

	// TODO: remove the concept of agent config, most likely.
	if !cfg.IsConfigured() {
		slog.Warn("No agent configuration found")
		return app, nil
	}
	if err := app.InitCoderAgent(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize coder agent: %w", err)
	}
	return app, nil
}

// Config returns the application configuration.
func (app *App) Config() *config.Config {
	return app.config
}

// RunNonInteractive handles the execution flow when a prompt is provided via
// CLI flag.
func (app *App) RunNonInteractive(ctx context.Context, prompt string, quiet bool) error {
	slog.Info("Running in non-interactive mode")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var spinner *format.Spinner
	if !quiet {
		spinner = format.NewSpinner(ctx, cancel, "Generating")
		spinner.Start()
	}

	// Helper function to stop spinner once.
	stopSpinner := func() {
		if !quiet && spinner != nil {
			spinner.Stop()
			spinner = nil
		}
	}
	defer stopSpinner()

	const maxPromptLengthForTitle = 100
	titlePrefix := "Non-interactive: "
	var titleSuffix string

	if len(prompt) > maxPromptLengthForTitle {
		titleSuffix = prompt[:maxPromptLengthForTitle] + "..."
	} else {
		titleSuffix = prompt
	}
	title := titlePrefix + titleSuffix

	sess, err := app.Sessions.Create(ctx, title)
	if err != nil {
		return fmt.Errorf("failed to create session for non-interactive mode: %w", err)
	}
	slog.Info("Created session for non-interactive run", "session_id", sess.ID)

	// Automatically approve all permission requests for this non-interactive session
	app.Permissions.AutoApproveSession(sess.ID)

	type response struct {
		result *fantasy.AgentResult
		err    error
	}
	done := make(chan response, 1)

	go func(ctx context.Context, sessionID, prompt string) {
		result, err := app.AgentCoordinator.Run(ctx, sess.ID, prompt)
		if err != nil {
			done <- response{
				err: fmt.Errorf("failed to start agent processing stream: %w", err),
			}
		}
		done <- response{
			result: result,
		}
	}(ctx, sess.ID, prompt)

	messageEvents := app.Messages.Subscribe(ctx)
	messageReadBytes := make(map[string]int)

	defer fmt.Printf(ansi.ResetProgressBar)
	for {
		// HACK: add it again on every iteration so it doesn't get hidden by
		// the terminal due to inactivity.
		fmt.Printf(ansi.SetIndeterminateProgressBar)
		select {
		case result := <-done:
			stopSpinner()
			if result.err != nil {
				if errors.Is(result.err, context.Canceled) || errors.Is(result.err, agent.ErrRequestCancelled) {
					slog.Info("Non-interactive: agent processing cancelled", "session_id", sess.ID)
					return nil
				}
				return fmt.Errorf("agent processing failed: %w", result.err)
			}
			return nil

		case event := <-messageEvents:
			msg := event.Payload
			if msg.SessionID == sess.ID && msg.Role == message.Assistant && len(msg.Parts) > 0 {
				stopSpinner()

				content := msg.Content().String()
				readBytes := messageReadBytes[msg.ID]

				if len(content) < readBytes {
					slog.Error("Non-interactive: message content is shorter than read bytes", "message_length", len(content), "read_bytes", readBytes)
					return fmt.Errorf("message content is shorter than read bytes: %d < %d", len(content), readBytes)
				}

				part := content[readBytes:]
				fmt.Print(part)
				messageReadBytes[msg.ID] = len(content)
			}

		case <-ctx.Done():
			stopSpinner()
			return ctx.Err()
		}
	}
}

func (app *App) UpdateAgentModel(ctx context.Context) error {
	return app.AgentCoordinator.UpdateModels(ctx)
}

func (app *App) setupEvents() {
	ctx, cancel := context.WithCancel(app.globalCtx)
	app.eventsCtx = ctx
	setupSubscriber(ctx, app.serviceEventsWG, "sessions", app.Sessions.Subscribe, app.events)
	setupSubscriber(ctx, app.serviceEventsWG, "messages", app.Messages.Subscribe, app.events)
	setupSubscriber(ctx, app.serviceEventsWG, "permissions", app.Permissions.Subscribe, app.events)
	setupSubscriber(ctx, app.serviceEventsWG, "permissions-notifications", app.Permissions.SubscribeNotifications, app.events)
	setupSubscriber(ctx, app.serviceEventsWG, "history", app.History.Subscribe, app.events)
	setupSubscriber(ctx, app.serviceEventsWG, "mcp", tools.SubscribeMCPEvents, app.events)
	setupSubscriber(ctx, app.serviceEventsWG, "lsp", SubscribeLSPEvents, app.events)

	// Setup plugin event forwarding
	app.setupPluginEventForwarding(ctx)

	cleanupFunc := func() error {
		cancel()
		app.serviceEventsWG.Wait()
		return nil
	}
	app.cleanupFuncs = append(app.cleanupFuncs, cleanupFunc)
}

// initPlugins initializes all plugins from configuration
func (app *App) initPlugins(ctx context.Context) error {
	pluginCtx := plugin.PluginContext{
		Config: app.config,
		Services: plugin.Services{
			Session:    app.Sessions,
			Message:    app.Messages,
			Permission: app.Permissions,
		},
		WorkingDir: app.config.WorkingDir(),
	}

	// Load plugins from config
	loader := plugin.NewLoader(app.PluginRegistry)
	if err := loader.LoadFromConfig(ctx, app.config, pluginCtx); err != nil {
		return fmt.Errorf("failed to load plugins from config: %w", err)
	}

	// Trigger config hooks after plugins are loaded
	if err := app.PluginRegistry.TriggerConfigHooks(ctx, app.config); err != nil {
		return fmt.Errorf("failed to trigger config hooks: %w", err)
	}

	// Add plugin shutdown to cleanup functions
	app.cleanupFuncs = append(app.cleanupFuncs, func() error {
		return app.PluginRegistry.Shutdown(ctx)
	})

	slog.Info("Plugins initialized", "count", len(app.PluginRegistry.ListPlugins()))
	return nil
}

// setupPluginEventForwarding forwards service events to plugin hooks
func (app *App) setupPluginEventForwarding(ctx context.Context) {
	// Forward session events to plugins
	app.serviceEventsWG.Go(func() {
		ch := app.Sessions.Subscribe(ctx)
		for {
			select {
			case event, ok := <-ch:
				if !ok {
					return
				}
				switch event.Type {
				case pubsub.CreatedEvent:
					if err := app.PluginRegistry.TriggerSessionCreated(ctx, event.Payload); err != nil {
						slog.Error("Plugin session created hook failed", "error", err)
					}
				case pubsub.UpdatedEvent:
					if err := app.PluginRegistry.TriggerSessionUpdated(ctx, event.Payload); err != nil {
						slog.Error("Plugin session updated hook failed", "error", err)
					}
				case pubsub.DeletedEvent:
					if err := app.PluginRegistry.TriggerSessionDeleted(ctx, event.Payload.ID); err != nil {
						slog.Error("Plugin session deleted hook failed", "error", err)
					}
				}
			case <-ctx.Done():
				return
			}
		}
	})

	// Forward message events to plugins
	app.serviceEventsWG.Go(func() {
		ch := app.Messages.Subscribe(ctx)
		for {
			select {
			case event, ok := <-ch:
				if !ok {
					return
				}
				switch event.Type {
				case pubsub.CreatedEvent:
					if err := app.PluginRegistry.TriggerMessageCreated(ctx, event.Payload); err != nil {
						slog.Error("Plugin message created hook failed", "error", err)
					}
				case pubsub.UpdatedEvent:
					if err := app.PluginRegistry.TriggerMessageUpdated(ctx, event.Payload); err != nil {
						slog.Error("Plugin message updated hook failed", "error", err)
					}
				}
			case <-ctx.Done():
				return
			}
		}
	})
}

func setupSubscriber[T any](
	ctx context.Context,
	wg *sync.WaitGroup,
	name string,
	subscriber func(context.Context) <-chan pubsub.Event[T],
	outputCh chan<- tea.Msg,
) {
	wg.Go(func() {
		subCh := subscriber(ctx)
		for {
			select {
			case event, ok := <-subCh:
				if !ok {
					slog.Debug("subscription channel closed", "name", name)
					return
				}
				var msg tea.Msg = event
				select {
				case outputCh <- msg:
				case <-time.After(2 * time.Second):
					slog.Warn("message dropped due to slow consumer", "name", name)
				case <-ctx.Done():
					slog.Debug("subscription cancelled", "name", name)
					return
				}
			case <-ctx.Done():
				slog.Debug("subscription cancelled", "name", name)
				return
			}
		}
	})
}

func (app *App) InitCoderAgent(ctx context.Context) error {
	coderAgentCfg := app.config.Agents[config.AgentCoder]
	if coderAgentCfg.ID == "" {
		return fmt.Errorf("coder agent configuration is missing")
	}
	var err error
	app.AgentCoordinator, err = agent.NewCoordinator(
		ctx,
		app.config,
		app.Sessions,
		app.Messages,
		app.Permissions,
		app.History,
		app.LSPClients,
		app.PluginRegistry,
	)
	if err != nil {
		slog.Error("Failed to create coder agent", "err", err)
		return err
	}

	// Add MCP client cleanup to shutdown process
	app.cleanupFuncs = append(app.cleanupFuncs, tools.CloseMCPClients)
	return nil
}

// Subscribe sends events to the TUI as tea.Msgs.
func (app *App) Subscribe(program *tea.Program) {
	defer log.RecoverPanic("app.Subscribe", func() {
		slog.Info("TUI subscription panic: attempting graceful shutdown")
		program.Quit()
	})

	app.tuiWG.Add(1)
	tuiCtx, tuiCancel := context.WithCancel(app.globalCtx)
	app.cleanupFuncs = append(app.cleanupFuncs, func() error {
		slog.Debug("Cancelling TUI message handler")
		tuiCancel()
		app.tuiWG.Wait()
		return nil
	})
	defer app.tuiWG.Done()

	for {
		select {
		case <-tuiCtx.Done():
			slog.Debug("TUI message handler shutting down")
			return
		case msg, ok := <-app.events:
			if !ok {
				slog.Debug("TUI message channel closed")
				return
			}
			program.Send(msg)
		}
	}
}

// Shutdown performs a graceful shutdown of the application.
func (app *App) Shutdown() {
	if app.AgentCoordinator != nil {
		app.AgentCoordinator.CancelAll()
	}

	// Shutdown all LSP clients.
	for name, client := range app.LSPClients.Seq2() {
		shutdownCtx, cancel := context.WithTimeout(app.globalCtx, 5*time.Second)
		if err := client.Close(shutdownCtx); err != nil {
			slog.Error("Failed to shutdown LSP client", "name", name, "error", err)
		}
		cancel()
	}

	// Call call cleanup functions.
	for _, cleanup := range app.cleanupFuncs {
		if cleanup != nil {
			if err := cleanup(); err != nil {
				slog.Error("Failed to cleanup app properly on shutdown", "error", err)
			}
		}
	}
}
