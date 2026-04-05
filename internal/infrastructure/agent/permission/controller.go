// Package permission provides tool permission control for Agent execution.
package permission

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ryuyb/litchi/internal/domain/service"
	"go.uber.org/zap"
)

// PermissionController defines the interface for tool permission control.
type PermissionController interface {
	// GetAllowedTools returns the allowed tools for a stage.
	GetAllowedTools(stage service.AgentStage) []string

	// IsToolAllowed checks if a tool is allowed for a stage.
	IsToolAllowed(tool string, stage service.AgentStage) bool

	// IsToolDangerous checks if a tool is dangerous.
	IsToolDangerous(tool string) bool

	// ValidateToolCall validates a tool call.
	ValidateToolCall(tool string, input string, stage service.AgentStage) error

	// GetBlockedTools returns tools blocked for a stage.
	GetBlockedTools(stage service.AgentStage) []string
}

// PermissionConfig contains permission configuration.
type PermissionConfig struct {
	// StageTools maps stages to allowed tools.
	StageTools map[service.AgentStage][]string

	// DangerousTools is the global dangerous tools list.
	DangerousTools []string

	// BlockedPatterns is the list of blocked tool patterns.
	BlockedPatterns []string
}

// DefaultDangerousTools is the list of dangerous tools that should always be blocked.
// These tools can cause data loss, system damage, or security issues.
var DefaultDangerousTools = []string{
	// Filesystem destruction
	"Bash(rm:*)",
	"Bash(rmdir:*)",
	"Bash(shred:*)",
	"Bash(wipe:*)",

	// Privilege escalation
	"Bash(sudo:*)",

	// Permission modification
	"Bash(chmod:*)",
	"Bash(chown:*)",

	// Disk operations
	"Bash(format:*)",
	"Bash(mkfs:*)",
	"Bash(dd:*)",
}

// DefaultPermissionConfig is the default permission configuration.
var DefaultPermissionConfig = PermissionConfig{
	StageTools: map[service.AgentStage][]string{
		service.AgentStageClarification: {
			"Read", "Glob", "Grep", "WebFetch", "WebSearch",
		},
		service.AgentStageDesign: {
			"Read", "Glob", "Grep", "WebFetch", "WebSearch",
		},
		service.AgentStageTaskBreakdown: {
			"Read", "Glob", "Grep", "WebFetch", "WebSearch",
		},
		service.AgentStageTaskExecution: {
			"Read", "Write", "Edit", "Bash", "Glob", "Grep", "WebFetch", "WebSearch",
		},
		service.AgentStagePRCreation: {
			"Read", "Bash(git:*)", "Glob", "Grep",
		},
	},
	DangerousTools: DefaultDangerousTools,
	BlockedPatterns: []string{
		// Privilege escalation patterns
		`Bash\(sudo:.*\)`,
		`Bash\(su:.*\)`,

		// Destructive file operations
		`Bash\(rm\s+-rf\s+/:.*\)`,     // Root directory deletion
		`Bash\(rm\s+-rf\s+.*\*:.*\)`,  // Wildcard root deletion

		// Dangerous permission changes
		`Bash\(chmod\s+777:.*\)`,      // World-writable permissions
		`Bash\(chmod\s+-R\s+777:.*\)`, // Recursive world-writable

		// System-critical file overwrite
		`Bash\(.*>\s*/etc/passwd.*\)`,
		`Bash\(.*>\s*/etc/shadow.*\)`,
		`Bash\(.*>\s*/boot/.*\)`,
		`Bash\(.*>\s*/dev/sd.*\)`,
		`Bash\(.*>\s*/dev/hd.*\)`,
	},
}

// DefaultPermissionController is the default implementation.
type DefaultPermissionController struct {
	config          PermissionConfig
	dangerousRegex  []*regexp.Regexp
	blockedRegex    []*regexp.Regexp
	allowedPatterns map[service.AgentStage][]*regexp.Regexp
	logger          *zap.Logger
}

// NewDefaultPermissionController creates a new default permission controller.
func NewDefaultPermissionController() *DefaultPermissionController {
	ctrl := &DefaultPermissionController{
		config:          DefaultPermissionConfig,
		dangerousRegex:  make([]*regexp.Regexp, 0),
		blockedRegex:    make([]*regexp.Regexp, 0),
		allowedPatterns: make(map[service.AgentStage][]*regexp.Regexp),
		logger:          zap.NewNop(), // Use nop logger for default controller
	}
	ctrl.compilePatterns()
	return ctrl
}

// NewPermissionController creates a permission controller with custom config.
func NewPermissionController(config PermissionConfig, logger *zap.Logger) *DefaultPermissionController {
	if logger == nil {
		logger = zap.NewNop()
	}
	ctrl := &DefaultPermissionController{
		config:          config,
		dangerousRegex:  make([]*regexp.Regexp, 0),
		blockedRegex:    make([]*regexp.Regexp, 0),
		allowedPatterns: make(map[service.AgentStage][]*regexp.Regexp),
		logger:          logger,
	}
	ctrl.compilePatterns()
	return ctrl
}

// compilePatterns compiles all regex patterns from configuration.
// Uses regexp.Compile to safely handle invalid patterns from user config.
func (c *DefaultPermissionController) compilePatterns() {
	// Compile dangerous tool patterns
	for _, tool := range c.config.DangerousTools {
		pattern := strings.ReplaceAll(regexp.QuoteMeta(tool), `\*`, `.*`)
		re, err := regexp.Compile("^" + pattern + "$")
		if err != nil {
			c.logger.Warn("invalid dangerous tool pattern, skipping",
				zap.String("tool", tool),
				zap.Error(err))
			continue
		}
		c.dangerousRegex = append(c.dangerousRegex, re)
	}

	// Compile blocked patterns
	for _, pattern := range c.config.BlockedPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			c.logger.Warn("invalid blocked pattern, skipping",
				zap.String("pattern", pattern),
				zap.Error(err))
			continue
		}
		c.blockedRegex = append(c.blockedRegex, re)
	}

	// Compile allowed patterns per stage
	for stage, tools := range c.config.StageTools {
		for _, tool := range tools {
			pattern := strings.ReplaceAll(regexp.QuoteMeta(tool), `\*`, `.*`)
			re, err := regexp.Compile("^" + pattern + "$")
			if err != nil {
				c.logger.Warn("invalid allowed tool pattern, skipping",
					zap.String("stage", string(stage)),
					zap.String("tool", tool),
					zap.Error(err))
				continue
			}
			c.allowedPatterns[stage] = append(c.allowedPatterns[stage], re)
		}
	}
}

// GetAllowedTools returns allowed tools for a stage.
func (c *DefaultPermissionController) GetAllowedTools(stage service.AgentStage) []string {
	if tools, ok := c.config.StageTools[stage]; ok {
		return tools
	}
	return []string{}
}

// IsToolAllowed checks if a tool is allowed for a stage.
func (c *DefaultPermissionController) IsToolAllowed(tool string, stage service.AgentStage) bool {
	// First check if dangerous
	if c.IsToolDangerous(tool) {
		return false
	}

	// Check against blocked patterns
	for _, re := range c.blockedRegex {
		if re.MatchString(tool) {
			return false
		}
	}

	// Check against allowed patterns
	if patterns, ok := c.allowedPatterns[stage]; ok {
		for _, re := range patterns {
			if re.MatchString(tool) {
				return true
			}
		}
	}

	// Default: not allowed
	return false
}

// IsToolDangerous checks if a tool is in the dangerous list.
func (c *DefaultPermissionController) IsToolDangerous(tool string) bool {
	for _, re := range c.dangerousRegex {
		if re.MatchString(tool) {
			return true
		}
	}
	return false
}

// ValidateToolCall validates a tool call and returns an error if invalid.
func (c *DefaultPermissionController) ValidateToolCall(tool string, input string, stage service.AgentStage) error {
	// Check if dangerous
	if c.IsToolDangerous(tool) {
		return fmt.Errorf("tool %s is blocked as dangerous", tool)
	}

	// Check against blocked patterns
	for _, re := range c.blockedRegex {
		if re.MatchString(tool) {
			return fmt.Errorf("tool %s matches blocked pattern", tool)
		}
	}

	// Check if allowed for stage
	if !c.IsToolAllowed(tool, stage) {
		return fmt.Errorf("tool %s is not allowed in stage %s", tool, stage)
	}

	return nil
}

// GetBlockedTools returns tools that would be blocked for a stage.
func (c *DefaultPermissionController) GetBlockedTools(stage service.AgentStage) []string {
	blocked := make([]string, 0)

	// All dangerous tools are blocked
	blocked = append(blocked, c.config.DangerousTools...)

	// Add tools not in allowed list for this stage
	if allowedTools, ok := c.config.StageTools[stage]; ok {
		allowedSet := make(map[string]bool)
		for _, t := range allowedTools {
			allowedSet[t] = true
		}

		// Add common tools that might be blocked
		allTools := []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep", "WebFetch", "WebSearch"}
		for _, tool := range allTools {
			if !allowedSet[tool] {
				blocked = append(blocked, tool)
			}
		}
	}

	return blocked
}

// FilterAllowedTools filters a list of tools to only allowed ones.
func (c *DefaultPermissionController) FilterAllowedTools(tools []string, stage service.AgentStage) []string {
	allowed := make([]string, 0)
	for _, tool := range tools {
		if c.IsToolAllowed(tool, stage) {
			allowed = append(allowed, tool)
		}
	}
	return allowed
}