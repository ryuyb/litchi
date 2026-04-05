package claude

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ryuyb/litchi/internal/domain/service"
	"github.com/ryuyb/litchi/internal/infrastructure/agent/permission"
)

// CommandBuilder builds Claude Code CLI commands.
type CommandBuilder struct {
	claudeBinary string
	defaultArgs  []string
}

// NewCommandBuilder creates a new command builder.
func NewCommandBuilder(claudeBinary string) *CommandBuilder {
	if claudeBinary == "" || claudeBinary == "claude-code" {
		claudeBinary = "claude" // default binary name
	}

	return &CommandBuilder{
		claudeBinary: claudeBinary,
		defaultArgs: []string{
			"--output-format", "json",
		},
	}
}

// ClaudeCommand represents a Claude Code command to execute.
type ClaudeCommand struct {
	Binary  string
	Args    []string
	WorkDir string
	Prompt  string
	Timeout time.Duration
	Env     map[string]string
}

// String returns the command string representation (for logging).
func (c *ClaudeCommand) String() string {
	// Truncate prompt for logging
	prompt := c.Prompt
	if len(prompt) > 100 {
		prompt = prompt[:100] + "..."
	}
	return fmt.Sprintf("%s %s --prompt '%s'", c.Binary, strings.Join(c.Args, " "), prompt)
}

// BuildCommand builds a complete execution command from the request.
func (b *CommandBuilder) BuildCommand(req *service.AgentRequest) *ClaudeCommand {
	args := make([]string, len(b.defaultArgs))
	copy(args, b.defaultArgs)

	// Add permission restrictions
	if len(req.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(req.AllowedTools, ","))
	}

	// Always add dangerous tool restrictions
	args = append(args, "--disallowedTools", strings.Join(permission.DefaultDangerousTools, ","))

	return &ClaudeCommand{
		Binary:  b.claudeBinary,
		Args:    args,
		WorkDir: req.WorktreePath,
		Prompt:  req.Prompt,
		Timeout: req.Timeout,
		Env:     b.buildEnvironment(req),
	}
}

// buildEnvironment builds environment variables for the command.
func (b *CommandBuilder) buildEnvironment(req *service.AgentRequest) map[string]string {
	env := map[string]string{
		"CLAUDE_SESSION_ID": req.SessionID.String(),
		"CLAUDE_STAGE":      string(req.Stage),
	}

	if req.Timeout > 0 {
		env["CLAUDE_TIMEOUT"] = req.Timeout.String()
	}

	return env
}

// BuildContextFile creates the context file path and content.
func (b *CommandBuilder) BuildContextFile(worktreePath string, ctx *service.AgentContext) (string, []byte, error) {
	// Create .litchi directory if not exists
	litchiDir := filepath.Join(worktreePath, ".litchi")
	if err := os.MkdirAll(litchiDir, 0755); err != nil {
		return "", nil, fmt.Errorf("failed to create .litchi directory: %w", err)
	}

	// Build context content
	content := b.buildContextContent(ctx)
	contextFile := filepath.Join(litchiDir, "agent_context.md")

	return contextFile, []byte(content), nil
}

// buildContextContent builds the context content for the Agent.
func (b *CommandBuilder) buildContextContent(ctx *service.AgentContext) string {
	var sb strings.Builder

	sb.WriteString("# Agent Execution Context\n\n")

	if ctx.IssueTitle != "" {
		sb.WriteString("## Issue\n")
		sb.WriteString(fmt.Sprintf("**Title:** %s\n\n", ctx.IssueTitle))
		if ctx.IssueBody != "" {
			sb.WriteString("**Description:**\n")
			sb.WriteString(ctx.IssueBody)
			sb.WriteString("\n\n")
		}
	}

	if len(ctx.ClarifiedPoints) > 0 {
		sb.WriteString("## Clarified Points\n")
		for _, point := range ctx.ClarifiedPoints {
			sb.WriteString(fmt.Sprintf("- %s\n", point))
		}
		sb.WriteString("\n")
	}

	if ctx.DesignContent != "" {
		sb.WriteString("## Design\n")
		sb.WriteString(ctx.DesignContent)
		sb.WriteString("\n\n")
	}

	if len(ctx.Tasks) > 0 {
		sb.WriteString("## Tasks\n")
		for _, task := range ctx.Tasks {
			sb.WriteString(fmt.Sprintf("- [%s] %s\n", task.Status, task.Description))
		}
		sb.WriteString("\n")
	}

	if ctx.Branch != "" {
		sb.WriteString(fmt.Sprintf("## Branch: %s\n", ctx.Branch))
	}

	return sb.String()
}

