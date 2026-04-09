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

// PythonProjectDetector detects Python projects.
type PythonProjectDetector struct {
	executor *command.Executor
	logger   *zap.Logger
}

// PythonProjectDetectorParams contains dependencies for PythonProjectDetector.
type PythonProjectDetectorParams struct {
		fx.In
		Executor *command.Executor
	Logger   *zap.Logger
}

// NewPythonProjectDetector creates a new Python project detector.
func NewPythonProjectDetector(p PythonProjectDetectorParams) service.ProjectDetector {
	return &PythonProjectDetector{
		executor: p.Executor,
		logger:   p.Logger.Named("python-detector"),
	}
}

// Detect detects Python project information.
func (d *PythonProjectDetector) Detect(ctx context.Context, worktreePath string) (*valueobject.DetectedProject, error) {
	// Check for Python project files
	pythonFiles := []string{
		"pyproject.toml",
		"setup.py",
		"requirements.txt",
		"setup.cfg",
		"Pipfile",
	}

	isPythonProject := false
	for _, f := range pythonFiles {
		if d.executor.CheckFileExists(worktreePath, f) {
			isPythonProject = true
			break
		}
	}

	// Also check for .py files as fallback
	if !isPythonProject {
		pyFiles, err := d.executor.FindFiles(worktreePath, "*.py")
		if err == nil && len(pyFiles) > 0 {
			isPythonProject = true
		}
	}

	if !isPythonProject {
		return nil, nil
	}

	project := valueobject.NewDetectedProject(valueobject.ProjectTypePython, "python", 90)

	// Detect formatter
	d.detectFormatter(worktreePath, project)

	// Detect linter
	d.detectLinter(worktreePath, project)

	// Detect test framework
	d.detectTestFramework(worktreePath, project)

	// Check Makefile
	d.detectMakefileTargets(worktreePath, project)

	return project, nil
}

// SupportsLanguage returns true for "python" language.
func (d *PythonProjectDetector) SupportsLanguage(language string) bool {
	return strings.ToLower(language) == "python"
}

// Priority returns priority for Python projects.
func (d *PythonProjectDetector) Priority() int {
	return 90
}

// detectFormatter detects formatting tool.
func (d *PythonProjectDetector) detectFormatter(worktreePath string, project *valueobject.DetectedProject) {
	// Check pyproject.toml for [tool.black] or [tool.ruff]
	if d.executor.CheckFileExists(worktreePath, "pyproject.toml") {
		content, err := d.executor.ReadFile(worktreePath, "pyproject.toml")
		if err == nil {
			contentStr := string(content)
			if strings.Contains(contentStr, "[tool.black]") {
				project.AddTool(valueobject.NewDetectedTool(
					valueobject.ToolTypeFormatter,
					"black",
					"[tool.black] in pyproject.toml",
					valueobject.NewToolCommand("black", "black", []string{"."}, 120),
				))
				return
			}
			if strings.Contains(contentStr, "[tool.ruff]") {
				project.AddTool(valueobject.NewDetectedTool(
					valueobject.ToolTypeFormatter,
					"ruff format",
					"[tool.ruff] in pyproject.toml",
					valueobject.NewToolCommand("ruff format", "ruff", []string{"format", "."}, 120),
				))
				return
			}
		}
	}

	// Default to black if available
	project.AddTool(valueobject.NewDetectedTool(
		valueobject.ToolTypeFormatter,
		"black",
		"python project default",
		valueobject.NewToolCommand("black", "black", []string{"."}, 120),
	))
}

// detectLinter detects linting tool.
func (d *PythonProjectDetector) detectLinter(worktreePath string, project *valueobject.DetectedProject) {
	// Check for ruff first (modern choice)
	if d.executor.CheckFileExists(worktreePath, "pyproject.toml") {
		content, err := d.executor.ReadFile(worktreePath, "pyproject.toml")
		if err == nil && strings.Contains(string(content), "[tool.ruff]") {
			project.AddTool(valueobject.NewDetectedTool(
				valueobject.ToolTypeLinter,
				"ruff",
				"[tool.ruff] in pyproject.toml",
				valueobject.NewToolCommand("ruff", "ruff", []string{"check", "--fix", "."}, 120),
			))
			return
		}
	}

	// Check for flake8 config
	flake8Configs := []string{".flake8", "setup.cfg", "tox.ini"}
	for _, configFile := range flake8Configs {
		if d.executor.CheckFileExists(worktreePath, configFile) {
			content, err := d.executor.ReadFile(worktreePath, configFile)
			if err == nil && strings.Contains(string(content), "[flake8]") {
				project.AddTool(valueobject.NewDetectedTool(
					valueobject.ToolTypeLinter,
					"flake8",
					"[flake8] in "+configFile,
					valueobject.NewToolCommand("flake8", "flake8", []string{"."}, 120),
				))
				return
			}
		}
	}

	// Check for pylint
	if d.executor.CheckFileExists(worktreePath, ".pylintrc") || d.executor.CheckFileExists(worktreePath, "pyproject.toml") {
		content, err := d.executor.ReadFile(worktreePath, "pyproject.toml")
		if err == nil && strings.Contains(string(content), "[tool.pylint") {
			project.AddTool(valueobject.NewDetectedTool(
				valueobject.ToolTypeLinter,
				"pylint",
				"[tool.pylint] in pyproject.toml",
				valueobject.NewToolCommand("pylint", "pylint", []string{"."}, 120),
			))
			return
		}
	}

	// Default to ruff (modern, fast)
	project.AddTool(valueobject.NewDetectedTool(
		valueobject.ToolTypeLinter,
		"ruff",
		"python project default",
		valueobject.NewToolCommand("ruff", "ruff", []string{"check", "--fix", "."}, 120),
	))
}

// detectTestFramework detects test framework.
func (d *PythonProjectDetector) detectTestFramework(worktreePath string, project *valueobject.DetectedProject) {
	// Check for pytest config
	if d.executor.CheckFileExists(worktreePath, "pytest.ini") || d.executor.CheckFileExists(worktreePath, "pyproject.toml") {
		if d.executor.CheckFileExists(worktreePath, "pytest.ini") {
			project.AddTool(valueobject.NewDetectedTool(
				valueobject.ToolTypeTester,
				"pytest",
				"pytest.ini exists",
				valueobject.NewToolCommand("pytest", "pytest", []string{}, 300),
			))
			return
		}
		content, err := d.executor.ReadFile(worktreePath, "pyproject.toml")
		if err == nil && strings.Contains(string(content), "[tool.pytest") {
			project.AddTool(valueobject.NewDetectedTool(
				valueobject.ToolTypeTester,
				"pytest",
				"[tool.pytest] in pyproject.toml",
				valueobject.NewToolCommand("pytest", "pytest", []string{}, 300),
			))
			return
		}
	}

	// Check requirements.txt for pytest
	if d.executor.CheckFileExists(worktreePath, "requirements.txt") {
		content, err := d.executor.ReadFile(worktreePath, "requirements.txt")
		if err == nil && strings.Contains(string(content), "pytest") {
			project.AddTool(valueobject.NewDetectedTool(
				valueobject.ToolTypeTester,
				"pytest",
				"pytest in requirements.txt",
				valueobject.NewToolCommand("pytest", "pytest", []string{}, 300),
			))
			return
		}
	}

	// Default to pytest
	project.AddTool(valueobject.NewDetectedTool(
		valueobject.ToolTypeTester,
		"pytest",
		"python project default",
		valueobject.NewToolCommand("pytest", "pytest", []string{}, 300),
	))
}

// detectMakefileTargets detects Makefile targets.
func (d *PythonProjectDetector) detectMakefileTargets(worktreePath string, project *valueobject.DetectedProject) {
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

	if targets["lint"] {
		project.DetectedTools = d.removeToolsByType(project.DetectedTools, valueobject.ToolTypeLinter)
		project.AddTool(valueobject.NewDetectedTool(
			valueobject.ToolTypeLinter,
			"make lint",
			"Makefile has lint target",
			valueobject.NewToolCommand("make lint", "make", []string{"lint"}, 120),
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
func (d *PythonProjectDetector) parseMakefileTargets(content string) map[string]bool {
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
func (d *PythonProjectDetector) removeToolsByType(tools []valueobject.DetectedTool, toolType valueobject.ToolType) []valueobject.DetectedTool {
	return slices.DeleteFunc(tools, func(t valueobject.DetectedTool) bool {
		return t.Type == toolType
	})
}