// Package skills implements Anthropic's Agent Skills Specification for Crush.
// It discovers SKILL.md files from standard locations and registers them as
// dynamic tools that the agent can invoke to access specialized knowledge.
package skills

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/plugin"
	"gopkg.in/yaml.v3"
)

// SkillFrontmatter represents the YAML frontmatter in SKILL.md files
type SkillFrontmatter struct {
	Name         string            `yaml:"name"`
	Description  string            `yaml:"description"`
	License      string            `yaml:"license,omitempty"`
	AllowedTools []string          `yaml:"allowed-tools,omitempty"`
	Metadata     map[string]string `yaml:"metadata,omitempty"`
}

// Skill represents a parsed skill with its metadata and content
type Skill struct {
	Name         string
	FullPath     string
	ToolName     string
	Description  string
	AllowedTools []string
	Metadata     map[string]string
	License      string
	Content      string
	Path         string
}

// Plugin implements the Crush plugin interface for skills
type Plugin struct {
	info   plugin.PluginInfo
	hooks  *plugin.BaseHooks
	skills []Skill
	tools  []plugin.PluginTool
}

// NewPlugin creates a new skills plugin instance
func NewPlugin() *Plugin {
	return &Plugin{
		info: plugin.PluginInfo{
			Name:        "crush-skills",
			Version:     "1.0.0",
			Description: "Implements Anthropic's Agent Skills Specification for Crush",
			Author:      "Crush Team",
		},
		hooks:  plugin.NewBaseHooks(),
		skills: []Skill{},
		tools:  []plugin.PluginTool{},
	}
}

// Info returns metadata about the plugin
func (p *Plugin) Info() plugin.PluginInfo {
	return p.info
}

// Init is called when the plugin is loaded
func (p *Plugin) Init(ctx context.Context, pluginCtx plugin.PluginContext) error {
	// Get skill discovery paths
	basePaths := getSkillBasePaths(pluginCtx.WorkingDir)

	// Discover skills
	skills, err := discoverSkills(basePaths)
	if err != nil {
		return fmt.Errorf("failed to discover skills: %w", err)
	}

	p.skills = skills

	// Register each skill as a tool
	for _, skill := range skills {
		// Capture skill in closure
		s := skill

		tool := &skillTool{
			name:        s.ToolName,
			description: s.Description,
			skill:       s,
		}

		p.tools = append(p.tools, tool)
	}

	if len(skills) > 0 {
		fmt.Fprintf(os.Stderr, "Skills Plugin: Loaded %d skill(s)\n", len(skills))
		for _, skill := range skills {
			fmt.Fprintf(os.Stderr, "  - %s: %s\n", skill.ToolName, skill.Description)
		}
	}

	return nil
}

// Hooks returns the hook implementations provided by this plugin
func (p *Plugin) Hooks() plugin.Hooks {
	return p.hooks
}

// Shutdown is called when the application is shutting down
func (p *Plugin) Shutdown(ctx context.Context) error {
	return nil
}

// GetTools returns the custom tools provided by this plugin
func (p *Plugin) GetTools() []plugin.PluginTool {
	return p.tools
}

// skillTool implements plugin.PluginTool for a single skill
type skillTool struct {
	name        string
	description string
	skill       Skill
}

func (t *skillTool) Info() fantasy.ToolInfo {
	return fantasy.ToolInfo{
		Name:        t.name,
		Description: t.description,
		Parameters:  map[string]any{}, // No parameters needed
	}
}

func (t *skillTool) Run(ctx context.Context, params fantasy.ToolCall) (fantasy.ToolResponse, error) {
	// Format the skill content with base directory
	output := fmt.Sprintf("Launching skill: %s\n\nBase directory for this skill: %s\n\n%s",
		t.skill.Name,
		t.skill.FullPath,
		t.skill.Content,
	)

	return fantasy.NewTextResponse(output), nil
}

func (t *skillTool) ProviderOptions() fantasy.ProviderOptions {
	return fantasy.ProviderOptions{}
}

// validateSkillName checks if the skill name matches the expected format
func validateSkillName(name string) bool {
	match, _ := regexp.MatchString(`^[a-z0-9-]+$`, name)
	return match
}

// generateToolName converts a skill path to a tool name
// Example: "brand-guidelines" -> "skills_brand_guidelines"
// Example: "tools/analyzer" -> "skills_tools_analyzer"
func generateToolName(skillPath string) string {
	// Clean the path and convert to tool name
	cleaned := strings.TrimPrefix(skillPath, "./")
	cleaned = strings.TrimSuffix(cleaned, "/")

	// Replace path separators and non-alphanumeric with underscores
	toolName := strings.ReplaceAll(cleaned, "/", "_")
	toolName = regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(toolName, "_")
	toolName = strings.ToLower(toolName)

	return "skills_" + toolName
}

// parseSkillMD parses a SKILL.md file and returns a Skill struct
func parseSkillMD(skillPath string) (*Skill, error) {
	// Read the file
	content, err := os.ReadFile(skillPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill file: %w", err)
	}

	// Parse frontmatter and content
	// Look for YAML frontmatter between --- markers
	contentStr := string(content)
	parts := strings.SplitN(contentStr, "---", 3)

	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid SKILL.md format: missing frontmatter")
	}

	// Parse YAML frontmatter
	var frontmatter SkillFrontmatter
	if err := yaml.Unmarshal([]byte(parts[1]), &frontmatter); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Validate required fields
	if frontmatter.Name == "" {
		return nil, fmt.Errorf("skill name is required in frontmatter")
	}
	if !validateSkillName(frontmatter.Name) {
		return nil, fmt.Errorf("invalid skill name format: %s (must be lowercase alphanumeric with hyphens)", frontmatter.Name)
	}
	if len(frontmatter.Description) < 20 {
		return nil, fmt.Errorf("skill description must be at least 20 characters")
	}

	// Get the skill directory name
	skillDir := filepath.Dir(skillPath)
	skillDirName := filepath.Base(skillDir)

	// Verify name matches directory name
	if frontmatter.Name != skillDirName {
		return nil, fmt.Errorf("skill name '%s' does not match directory name '%s'", frontmatter.Name, skillDirName)
	}

	// Get relative path from skills directory for tool name generation
	// Extract the relative path after "skills/"
	skillsIdx := strings.LastIndex(skillDir, "/skills/")
	var relPath string
	if skillsIdx >= 0 {
		relPath = skillDir[skillsIdx+8:] // +8 to skip "/skills/"
	} else {
		relPath = skillDirName
	}

	// Create skill object
	skill := &Skill{
		Name:         frontmatter.Name,
		FullPath:     skillDir,
		ToolName:     generateToolName(relPath),
		Description:  frontmatter.Description,
		AllowedTools: frontmatter.AllowedTools,
		Metadata:     frontmatter.Metadata,
		License:      frontmatter.License,
		Content:      strings.TrimSpace(parts[2]),
		Path:         skillPath,
	}

	return skill, nil
}

// discoverSkills scans directories for SKILL.md files
func discoverSkills(basePaths []string) ([]Skill, error) {
	var allSkills []Skill
	seenToolNames := make(map[string]string) // toolName -> skillPath

	for _, basePath := range basePaths {
		// Check if directory exists
		if _, err := os.Stat(basePath); os.IsNotExist(err) {
			continue // Skip missing directories
		}

		// Walk the directory tree looking for SKILL.md files
		err := filepath.WalkDir(basePath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil // Skip errors, continue walking
			}

			// Check if this is a SKILL.md file
			if !d.IsDir() && d.Name() == "SKILL.md" {
				skill, parseErr := parseSkillMD(path)
				if parseErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: Failed to parse skill at %s: %v\n", path, parseErr)
					return nil // Continue walking despite parse error
				}

				// Check for duplicate tool names
				if existingPath, exists := seenToolNames[skill.ToolName]; exists {
					fmt.Fprintf(os.Stderr, "Warning: Duplicate tool name '%s' for skills at %s and %s. Using the later one.\n",
						skill.ToolName, existingPath, path)
					// Remove the old skill
					for i, s := range allSkills {
						if s.ToolName == skill.ToolName {
							allSkills = append(allSkills[:i], allSkills[i+1:]...)
							break
						}
					}
				}

				seenToolNames[skill.ToolName] = path
				allSkills = append(allSkills, *skill)
			}

			return nil
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Error walking directory %s: %v\n", basePath, err)
		}
	}

	return allSkills, nil
}

// getSkillBasePaths returns the paths to search for skills in priority order (low to high)
func getSkillBasePaths(workingDir string) []string {
	var paths []string

	// 1. XDG config directory (or ~/.config/crush/skills/)
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			configDir = filepath.Join(homeDir, ".config")
		}
	}
	if configDir != "" {
		paths = append(paths, filepath.Join(configDir, "crush", "skills"))
	}

	// 2. Home directory ~/.crush/skills/
	homeDir, err := os.UserHomeDir()
	if err == nil {
		paths = append(paths, filepath.Join(homeDir, ".crush", "skills"))
	}

	// 3. Project-local .crush/skills/ (highest priority)
	paths = append(paths, filepath.Join(workingDir, ".crush", "skills"))

	return paths
}
