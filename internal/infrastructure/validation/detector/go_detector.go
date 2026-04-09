package detector

import (
	"go.uber.org/fx"
	"context"
	"slices"
	"strings"

	"github.com/ryuyb/litchi/internal/domain/service"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/validation/command"
	"go.uber.org/zap"
)

// GoProjectDetector detects Go projects.
type GoProjectDetector struct {
	executor *command.Executor
	logger   *zap.Logger
}

// GoProjectDetectorParams contains dependencies for GoProjectDetector.
type GoProjectDetectorParams struct {
		fx.In
	Executor *command.Executor
	Logger   *zap.Logger
}

// NewGoProjectDetector creates a new Go project detector.
func NewGoProjectDetector(p GoProjectDetectorParams) service.ProjectDetector {
	return &GoProjectDetector{
		executor: p.Executor,
		logger:   p.Logger.Named("go-detector"),
	}
}

// Detect detects Go project information.
func (d *GoProjectDetector) Detect(ctx context.Context, worktreePath string) (*valueobject.DetectedProject, error) {
	// 1. Check go.mod exists
	if !d.executor.CheckFileExists(worktreePath, "go.mod") {
		return nil, nil // Not a Go project
	}

	project := valueobject.NewDetectedProject(valueobject.ProjectTypeGo, "go", 95)

	// 2. Add default formatter (gofmt)
	project.AddTool(valueobject.NewDetectedTool(
		valueobject.ToolTypeFormatter,
		"gofmt",
		"go project default",
		valueobject.NewToolCommand("gofmt", "gofmt", []string{"-w", "."}, 60),
	))

	// 3. Detect lint tools
	d.detectLintTools(worktreePath, project)

	// 4. Detect test tools
	d.detectTestTools(worktreePath, project)

	// 5. Check Makefile (may override defaults)
	d.detectMakefileTargets(worktreePath, project)

	return project, nil
}

// SupportsLanguage returns true for "go" language.
func (d *GoProjectDetector) SupportsLanguage(language string) bool {
	return strings.ToLower(language) == "go"
}

// Priority returns high priority for Go projects.
func (d *GoProjectDetector) Priority() int {
	return 100
}

// detectLintTools detects lint configuration.
func (d *GoProjectDetector) detectLintTools(worktreePath string, project *valueobject.DetectedProject) {
	// Check for golangci-lint config files
	golangciConfigs := []string{
		".golangci.yml",
		".golangci.yaml",
		".golangci.toml",
		".golangci.json",
	}

	for _, configFile := range golangciConfigs {
		if d.executor.CheckFileExists(worktreePath, configFile) {
			tool := valueobject.NewDetectedTool(
				valueobject.ToolTypeLinter,
				"golangci-lint",
				configFile+" exists",
				valueobject.NewToolCommand("golangci-lint", "golangci-lint", []string{"run", "--fix"}, 120).
					WithConfigCheck(configFile),
			).WithConfigFile(configFile)
			project.AddTool(tool)
			return // Only one lint tool needed
		}
	}

	// Fallback to go vet if no golangci-lint config
	tool := valueobject.NewDetectedTool(
		valueobject.ToolTypeLinter,
		"go vet",
		"go project default",
		valueobject.NewToolCommand("go vet", "go", []string{"vet", "./..."}, 60),
	)
	project.AddTool(tool)
}

// detectTestTools detects test configuration.
func (d *GoProjectDetector) detectTestTools(worktreePath string, project *valueobject.DetectedProject) {
	// Check for test files
	testFiles, err := d.executor.FindFiles(worktreePath, "*_test.go")
	if err != nil {
		d.logger.Warn("failed to find test files", zap.Error(err))
	}

	basis := "go project default"
	if len(testFiles) > 0 {
		basis = "_test.go files found"
	}

	tool := valueobject.NewDetectedTool(
		valueobject.ToolTypeTester,
		"go test",
		basis,
		valueobject.NewToolCommand("go test", "go", []string{"test", "-v", "./..."}, 300),
	)
	project.AddTool(tool)
}

// detectMakefileTargets detects Makefile targets and may override defaults.
func (d *GoProjectDetector) detectMakefileTargets(worktreePath string, project *valueobject.DetectedProject) {
	if !d.executor.CheckFileExists(worktreePath, "Makefile") {
		return
	}

	// Read Makefile content
	content, err := d.executor.ReadFile(worktreePath, "Makefile")
	if err != nil {
		d.logger.Warn("failed to read Makefile", zap.Error(err))
		return
	}

	makefileContent := string(content)

	// Parse targets
	targets := d.parseMakefileTargets(makefileContent)

	// Override formatter if Makefile has format/fmt target
	if targets["format"] || targets["fmt"] {
		// Remove default gofmt and add make format
		project.DetectedTools = d.removeToolsByType(project.DetectedTools, valueobject.ToolTypeFormatter)
		project.AddTool(valueobject.NewDetectedTool(
			valueobject.ToolTypeFormatter,
			"make format",
			"Makefile has format/fmt target",
			valueobject.NewToolCommand("make format", "make", []string{"format"}, 120),
		))
	}

	// Override lint if Makefile has lint target
	if targets["lint"] {
		project.DetectedTools = d.removeToolsByType(project.DetectedTools, valueobject.ToolTypeLinter)
		project.AddTool(valueobject.NewDetectedTool(
			valueobject.ToolTypeLinter,
			"make lint",
			"Makefile has lint target",
			valueobject.NewToolCommand("make lint", "make", []string{"lint"}, 120),
		))
	}

	// Override test if Makefile has test/check target
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

// parseMakefileTargets parses Makefile to find target definitions.
func (d *GoProjectDetector) parseMakefileTargets(content string) map[string]bool {
	targets := map[string]bool{}
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Target lines start with a word followed by colon (no leading spaces)
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

// removeToolsByType removes tools of a specific type from the list.
func (d *GoProjectDetector) removeToolsByType(tools []valueobject.DetectedTool, toolType valueobject.ToolType) []valueobject.DetectedTool {
	return slices.DeleteFunc(tools, func(t valueobject.DetectedTool) bool {
		return t.Type == toolType
	})
}