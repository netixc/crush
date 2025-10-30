// Package main provides a metrics collection plugin example for Crush.
//
// This plugin demonstrates:
// - Subscribing to multiple hook types
// - Collecting metrics across sessions, messages, and tool executions
// - Implementing agent lifecycle hooks
//
// To build this plugin:
//   go build -buildmode=plugin -o metrics.so main.go
//
// To use this plugin, add to your crush config:
//   {
//     "plugins": ["./examples/plugins/metrics/metrics.so"]
//   }
package main

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/session"
	"github.com/charmbracelet/crush/pkg/crushsdk"
)

// Plugin is the exported symbol that Crush will load
var Plugin crushsdk.Plugin = &MetricsPlugin{}

// MetricsPlugin collects and logs metrics about Crush usage
type MetricsPlugin struct {
	*crushsdk.SimplePlugin
	metrics *Metrics
}

// Metrics stores various usage statistics
type Metrics struct {
	mu sync.RWMutex

	// Session metrics
	SessionsCreated int
	SessionsActive  map[string]bool

	// Message metrics
	MessagesCreated int
	MessagesByRole  map[string]int

	// Tool metrics
	ToolExecutions  int
	ToolsByName     map[string]int
	ToolErrors      int

	// Agent metrics
	AgentRuns       int
	TotalSteps      int
	AgentErrors     int

	// Timing
	StartTime       time.Time
	LastActivity    time.Time
}

func init() {
	plugin := &MetricsPlugin{
		SimplePlugin: crushsdk.NewSimplePlugin(crushsdk.PluginInfo{
			Name:        "metrics",
			Version:     "1.0.0",
			Description: "Collects and logs metrics about Crush usage patterns",
			Author:      "Crush Examples",
		}),
		metrics: &Metrics{
			SessionsActive: make(map[string]bool),
			MessagesByRole: make(map[string]int),
			ToolsByName:    make(map[string]int),
			StartTime:      time.Now(),
			LastActivity:   time.Now(),
		},
	}

	// Set up custom hooks
	hooks := crushsdk.NewBaseHooks()
	hooks.SessionHook = &metricsSessionHook{plugin: plugin}
	hooks.MessageHook = &metricsMessageHook{plugin: plugin}
	hooks.ToolHook = &metricsToolHook{plugin: plugin}
	hooks.AgentHook = &metricsAgentHook{plugin: plugin}
	plugin.SetHooks(hooks)

	Plugin = plugin
}

func (p *MetricsPlugin) Init(ctx context.Context, pluginCtx crushsdk.PluginContext) error {
	slog.Info("Metrics plugin initialized")

	// Start periodic metrics reporting
	go p.reportMetricsPeriodically(ctx)

	return p.SimplePlugin.Init(ctx, pluginCtx)
}

func (p *MetricsPlugin) Shutdown(ctx context.Context) error {
	p.logMetrics()
	return nil
}

func (p *MetricsPlugin) reportMetricsPeriodically(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.logMetrics()
		case <-ctx.Done():
			return
		}
	}
}

func (p *MetricsPlugin) logMetrics() {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	uptime := time.Since(p.metrics.StartTime)
	idleTime := time.Since(p.metrics.LastActivity)

	slog.Info("Crush Metrics Report",
		"uptime", uptime.Round(time.Second),
		"idle_time", idleTime.Round(time.Second),
		"sessions_created", p.metrics.SessionsCreated,
		"active_sessions", len(p.metrics.SessionsActive),
		"messages_created", p.metrics.MessagesCreated,
		"tool_executions", p.metrics.ToolExecutions,
		"tool_errors", p.metrics.ToolErrors,
		"agent_runs", p.metrics.AgentRuns,
		"total_agent_steps", p.metrics.TotalSteps,
		"agent_errors", p.metrics.AgentErrors,
	)

	if len(p.metrics.ToolsByName) > 0 {
		slog.Info("Top Tools", "tools", p.metrics.ToolsByName)
	}
}

// Session Hook Implementation

type metricsSessionHook struct {
	plugin *MetricsPlugin
	crushsdk.NilSessionHook
}

func (h *metricsSessionHook) OnSessionCreated(ctx context.Context, sess session.Session) error {
	h.plugin.metrics.mu.Lock()
	defer h.plugin.metrics.mu.Unlock()

	h.plugin.metrics.SessionsCreated++
	h.plugin.metrics.SessionsActive[sess.ID] = true
	h.plugin.metrics.LastActivity = time.Now()

	return nil
}

func (h *metricsSessionHook) OnSessionDeleted(ctx context.Context, sessionID string) error {
	h.plugin.metrics.mu.Lock()
	defer h.plugin.metrics.mu.Unlock()

	delete(h.plugin.metrics.SessionsActive, sessionID)
	h.plugin.metrics.LastActivity = time.Now()

	return nil
}

// Message Hook Implementation

type metricsMessageHook struct {
	plugin *MetricsPlugin
	crushsdk.NilMessageHook
}

func (h *metricsMessageHook) OnMessageCreated(ctx context.Context, msg message.Message) error {
	h.plugin.metrics.mu.Lock()
	defer h.plugin.metrics.mu.Unlock()

	h.plugin.metrics.MessagesCreated++
	h.plugin.metrics.MessagesByRole[msg.Role]++
	h.plugin.metrics.LastActivity = time.Now()

	return nil
}

// Tool Hook Implementation

type metricsToolHook struct {
	plugin *MetricsPlugin
	crushsdk.NilToolHook
}

func (h *metricsToolHook) OnToolExecuteBefore(ctx context.Context, input crushsdk.ToolExecuteInput) (map[string]any, error) {
	h.plugin.metrics.mu.Lock()
	defer h.plugin.metrics.mu.Unlock()

	h.plugin.metrics.ToolExecutions++
	h.plugin.metrics.ToolsByName[input.ToolName]++
	h.plugin.metrics.LastActivity = time.Now()

	return nil, nil
}

func (h *metricsToolHook) OnToolExecuteAfter(ctx context.Context, input crushsdk.ToolExecuteInput, result crushsdk.ToolExecuteResult) (*crushsdk.ToolExecuteResult, error) {
	if result.Error != nil {
		h.plugin.metrics.mu.Lock()
		h.plugin.metrics.ToolErrors++
		h.plugin.metrics.mu.Unlock()
	}
	return nil, nil
}

// Agent Hook Implementation

type metricsAgentHook struct {
	plugin *MetricsPlugin
	crushsdk.NilAgentHook
}

func (h *metricsAgentHook) OnAgentStart(ctx context.Context, input crushsdk.AgentStartInput) error {
	h.plugin.metrics.mu.Lock()
	defer h.plugin.metrics.mu.Unlock()

	h.plugin.metrics.AgentRuns++
	h.plugin.metrics.LastActivity = time.Now()

	return nil
}

func (h *metricsAgentHook) OnAgentStep(ctx context.Context, input crushsdk.AgentStepInput) error {
	h.plugin.metrics.mu.Lock()
	defer h.plugin.metrics.mu.Unlock()

	h.plugin.metrics.TotalSteps++
	h.plugin.metrics.LastActivity = time.Now()

	return nil
}

func (h *metricsAgentHook) OnAgentFinish(ctx context.Context, input crushsdk.AgentFinishInput) error {
	if input.Error != nil {
		h.plugin.metrics.mu.Lock()
		h.plugin.metrics.AgentErrors++
		h.plugin.metrics.mu.Unlock()
	}
	return nil
}
