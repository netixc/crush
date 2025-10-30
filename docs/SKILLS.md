# Crush Skills

Crush includes built-in support for [Anthropic's Agent Skills Specification v1.0](https://docs.anthropic.com/en/docs/build-with-claude/agent-skills).

## What are Skills?

Skills are reusable AI agent capabilities that provide:
- **Specialized Knowledge**: Domain-specific instructions and workflows
- **Bundled Resources**: Scripts, templates, documentation, and reference materials
- **Project Context**: Project-specific guidelines and conventions
- **Portable Instructions**: Shareable skill packages compatible with OpenCode and Claude Desktop

## Quick Start

### 1. Create a Skill

```bash
mkdir -p .crush/skills/my-skill
cat > .crush/skills/my-skill/SKILL.md <<'EOF'
---
name: my-skill
description: A helpful skill that provides specialized knowledge
---

# My Skill Instructions

Your instructions for the AI agent go here...
EOF
```

### 2. Use the Skill

Start Crush - skills are automatically discovered and registered as tools:

```
Skills Plugin: Loaded 1 skill(s)
  - skills_my_skill: A helpful skill that provides specialized knowledge
```

The agent can now invoke `skills_my_skill` like any other tool to access the skill's content.

## Skill Discovery Locations

Skills are discovered from three locations (priority: low → high):

1. **`~/.config/crush/skills/`** - Global skills (XDG config)
2. **`~/.crush/skills/`** - Alternative global location
3. **`.crush/skills/`** - Project-local skills (highest priority, overrides global)

## Skill Format

### SKILL.md Structure

```markdown
---
name: skill-name              # Required: lowercase alphanumeric with hyphens
description: What this skill does  # Required: min 20 characters
license: MIT                  # Optional
allowed-tools:                # Optional (suggestive, not enforced)
  - read
  - write
metadata:                     # Optional custom fields
  version: "1.0"
  author: "Your Name"
---

# Skill Content

Your instructions, examples, and documentation in Markdown...
```

### Skill Directory Structure

```
my-skill/
├── SKILL.md              # Required
├── scripts/              # Optional: executable code
│   └── helper.py
├── references/           # Optional: documentation
│   └── api-docs.md
└── assets/               # Optional: files for output
    └── template.html
```

### Validation Rules

- ✅ Name matches `^[a-z0-9-]+$` pattern
- ✅ Name matches directory name exactly
- ✅ Description is at least 20 characters
- ✅ Valid YAML frontmatter format

## Tool Naming

Skills are registered as tools with the `skills_` prefix:

- `my-skill/` → `skills_my_skill`
- `brand-guidelines/` → `skills_brand_guidelines`
- `nested/path/skill/` → `skills_nested_path_skill`

## When Skills Are Invoked

When the agent invokes a skill tool, it receives:

```
Launching skill: my-skill

Base directory for this skill: /path/to/.crush/skills/my-skill

[Full skill content from SKILL.md...]
```

The base directory allows the skill to reference local files using relative paths.

## Examples

### Code Review Skill

`.crush/skills/code-review/SKILL.md`:

```markdown
---
name: code-review
description: Comprehensive code review guidelines and checklist for this project
---

# Code Review Guidelines

When reviewing code in this project:

## Architecture
- Follow layered architecture (internal/app, internal/agent, internal/services)
- Proper dependency injection

## Testing
- Unit tests for new functionality
- Edge case coverage

## Style
- Go best practices
- Error wrapping with context
- Structured logging with slog
```

### API Documentation Skill

`.crush/skills/api-docs/SKILL.md`:

```markdown
---
name: api-docs
description: Complete API documentation and endpoint reference for the project
metadata:
  version: "2.1"
  last-updated: "2025-10-30"
---

# API Documentation

## Endpoints

### POST /api/sessions
Creates a new chat session...

### GET /api/sessions/:id
Retrieves session details...

[See references/openapi.yaml for full spec]
```

## Best Practices

### ✅ Do

- Keep skills focused on one domain/task
- Use descriptive names (`api-testing` not `test`)
- Include concrete examples
- Document any scripts or resources
- Version your skills using metadata
- Test that agents can follow your instructions

### ❌ Don't

- Make skills too broad (split into multiple skills)
- Use absolute paths (use base directory + relative paths)
- Hardcode project-specific paths
- Forget YAML validation
- Duplicate content across skills

## Compatibility

### Portable Between Systems

SKILL.md files work with:
- ✅ Crush (built-in)
- ✅ OpenCode (via opencode-skills plugin)
- ✅ Claude Desktop (via MCP skills)

All implement the same Agent Skills Specification v1.0.

### Differences from OpenCode

| Feature | OpenCode | Crush |
|---------|----------|-------|
| Installation | npm package | Built-in |
| Discovery | Same paths | Same paths |
| Format | Same | Same |
| Tool names | `skills:name` | `skills_name` |
| Delivery | Message insertion | Tool response |

## Troubleshooting

### Skill Not Discovered

Check:
1. SKILL.md exists in skill directory
2. YAML frontmatter is valid (test with `yamllint`)
3. Skill name matches directory name
4. Run with `--debug` to see warnings

### Validation Errors

Common issues:
- **Name contains uppercase**: Use lowercase only
- **Name has spaces**: Use hyphens instead
- **Description too short**: Min 20 characters required
- **Name/directory mismatch**: Must match exactly

### Tool Name Conflicts

If two skills generate the same tool name:
```
Warning: Duplicate tool name 'skills_my_skill' for skills at path1 and path2
```

Solution: Rename one skill directory. Higher-priority locations override lower ones.

## Implementation Details

Skills are implemented as a built-in plugin in `internal/skills/skills.go`:

- Discovers SKILL.md files on startup
- Parses and validates frontmatter
- Registers each skill as a dynamic tool
- Tools deliver skill content when invoked

The plugin integrates with Crush's plugin system and forwards skills to the agent coordinator for tool registration.

## See Also

- [Anthropic Agent Skills Specification](https://docs.anthropic.com/en/docs/build-with-claude/agent-skills)
- [Crush Plugin Development](PLUGIN_DEVELOPMENT.md)
- [OpenCode Skills](https://github.com/opencode-ai/opencode-skills)
