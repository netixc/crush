// Package main provides a simple "hello world" plugin example for Crush.
//
// This plugin demonstrates:
// - Basic plugin structure
// - Registering a custom tool
// - Implementing a simple tool handler
//
// To build this plugin:
//   go build -buildmode=plugin -o hello-world.so main.go
//
// To use this plugin, add to your crush config:
//   {
//     "plugins": ["./examples/plugins/hello-world/hello-world.so"]
//   }
package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/crush/pkg/crushsdk"
	"github.com/charmbracelet/fantasy"
)

// Plugin is the exported symbol that Crush will load
var Plugin crushsdk.Plugin = &HelloWorldPlugin{}

// HelloWorldPlugin is a simple plugin that adds a "hello" tool
type HelloWorldPlugin struct {
	*crushsdk.SimplePlugin
}

func init() {
	// Create the base plugin
	base := crushsdk.NewSimplePlugin(crushsdk.PluginInfo{
		Name:        "hello-world",
		Version:     "1.0.0",
		Description: "A simple hello world plugin that demonstrates custom tool registration",
		Author:      "Crush Examples",
	})

	// Create and add the hello tool
	helloTool := crushsdk.NewSimpleTool(
		"hello",
		"Says hello to the specified person. This is a demonstration tool from the hello-world plugin.",
		map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "The name of the person to greet",
			},
			"formal": map[string]any{
				"type":        "boolean",
				"description": "Whether to use formal greeting (optional)",
			},
		},
		[]string{"name"}, // required parameters
		func(ctx context.Context, params fantasy.ToolCall) (fantasy.ToolResponse, error) {
			// Parse input
			var input struct {
				Name   string `json:"name"`
				Formal bool   `json:"formal"`
			}
			if err := json.Unmarshal(params.Input, &input); err != nil {
				return fantasy.ToolResponse{
					Error: fmt.Errorf("invalid input: %w", err),
				}, nil
			}

			// Generate greeting
			var greeting string
			if input.Formal {
				greeting = fmt.Sprintf("Good day, %s. It is a pleasure to make your acquaintance.", input.Name)
			} else {
				greeting = fmt.Sprintf("Hey %s! 👋", input.Name)
			}

			return fantasy.ToolResponse{
				Text: greeting,
			}, nil
		},
	)

	base.AddTool(helloTool)

	// Initialize the plugin
	Plugin = &HelloWorldPlugin{SimplePlugin: base}
}
