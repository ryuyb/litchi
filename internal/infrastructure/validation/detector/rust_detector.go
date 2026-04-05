package detector

import (
	"context"
	"slices"
	"strings"

	"github.com/ryuyb/litchi/internal/domain/service"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/validation/command"
	"go.uber.org/zap"
)

// RustProjectDetector detects Rust projects.
type RustProjectDetector struct {
	executor *command.Executor
	logger   *zap.Logger
}

// RustProjectDetectorParams contains dependencies for RustProjectDetector.
type RustProjectDetectorParams struct {
	Executor *command.Executor
	Logger   *zap.Logger
}

// NewRustProjectDetector creates a new Rust project detector.
func NewRustProjectDetector(p RustProjectDetectorParams) service.ProjectDetector {
	return &RustProjectDetector{
		executor: p.Executor,
		logger:   p.Logger.Named("rust-detector"),
	}
}

// Detect detects Rust project information.
func (d *RustProjectDetector) Detect(ctx context.Context, worktreePath string) (*valueobject.DetectedProject, error) {
	// Check for Cargo.toml
	if !d.executor.CheckFileExists(worktreePath, "Cargo.toml") {
		return nil, nil
	}

	project := valueobject.NewDetectedProject(valueobject.ProjectTypeRust, "rust", 95)

	// Rust has standard tools: rustfmt, clippy, cargo test
	// Add formatter (rustfmt via cargo fmt)
	project.AddTool(valueobject.NewDetectedTool(
		valueobject.ToolTypeFormatter,
		"cargo fmt",
		"rust project default",
		valueobject.NewToolCommand("cargo fmt", "cargo", []string{"fmt"}, 120),
	))

	// Add linter (clippy)
	project.AddTool(valueobject.NewDetectedTool(
		valueobject.ToolTypeLinter,
		"cargo clippy",
		"rust project default",
		valueobject.NewToolCommand("cargo clippy", "cargo", []string{"clippy", "--fix", "--allow-dirty"}, 180),
	))

	// Add tester (cargo test)
	project.AddTool(valueobject.NewDetectedTool(
		valueobject.ToolTypeTester,
		"cargo test",
		"rust project default",
		valueobject.NewToolCommand("cargo test", "cargo", []string{"test"}, 300),
	))

	// Check Makefile (may override defaults)
	d.detectMakefileTargets(worktreePath, project)

	return project, nil
}

// SupportsLanguage returns true for "rust" language.
func (d *RustProjectDetector) SupportsLanguage(language string) bool {
	return strings.ToLower(language) == "rust"
}

// Priority returns priority for Rust projects.
func (d *RustProjectDetector) Priority() int {
	return 95
}

// detectMakefileTargets detects Makefile targets.
func (d *RustProjectDetector) detectMakefileTargets(worktreePath string, project *valueobject.DetectedProject) {
	if !d.executor.CheckFileExists(worktreePath, "Makefile") {
		return
	}

	content, err := d.executor.ReadFile(worktreePath, "Makefile")
	if err != nil {
		return
	}

	makefileContent := string(content)
	targets := d.parseMakefileTargets(makefileContent)

	if targets["format"] || targets["fmt"] {
		project.DetectedTools = d.removeToolsByType(project.DetectedTools, valueobject.ToolTypeFormatter)
		project.AddTool(valueobject.NewDetectedTool(
			valueobject.ToolTypeFormatter,
			"make format",
			"Makefile has format/fmt target",
			valueobject.NewToolCommand("make format", "make", []string{"format"}, 120),
		))
	}

	if targets["lint"] || targets["clippy"] {
		project.DetectedTools = d.removeToolsByType(project.DetectedTools, valueobject.ToolTypeLinter)
		project.AddTool(valueobject.NewDetectedTool(
			valueobject.ToolTypeLinter,
			"make lint",
			"Makefile has lint/clippy target",
			valueobject.NewToolCommand("make lint", "make", []string{"lint"}, 180),
		))
	}

	if targets["test"] || targets["check"] {
		project.DetectedTools = d.removeToolsByType(project.DetectedTools, valueobject.ToolTypeTester)
		project.AddTool(valueobject.NewDetectedTool(
			valueobject.ToolTypeTester,
			"make test",
			"Makefile has test/check target",
			valueobject.NewToolCommand("make test", "make", []string{"test"}, 300),
		))
	}
}

// parseMakefileTargets parses Makefile targets.
func (d *RustProjectDetector) parseMakefileTargets(content string) map[string]bool {
	targets := map[string]bool{}
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
			if idx := strings.Index(line, ":"); idx > 0 {
				targetName := strings.TrimSpace(line[:idx])
				if targetName != "" && !strings.HasPrefix(targetName, "#") {
					targets[strings.ToLower(targetName)] = true
				}
			}
		}
	}

	return targets
}

// removeToolsByType removes tools of a specific type.
func (d *RustProjectDetector) removeToolsByType(tools []valueobject.DetectedTool, toolType valueobject.ToolType) []valueobject.DetectedTool {
	return slices.DeleteFunc(tools, func(t valueobject.DetectedTool) bool {
		return t.Type == toolType
	})
}