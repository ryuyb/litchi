package detector

import (
	"go.uber.org/fx"
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/ryuyb/litchi/internal/domain/service"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/validation/command"
	"go.uber.org/zap"
)

// NodeJSProjectDetector detects Node.js projects.
type NodeJSProjectDetector struct {
	executor *command.Executor
	logger   *zap.Logger
}

// NodeJSProjectDetectorParams contains dependencies for NodeJSProjectDetector.
type NodeJSProjectDetectorParams struct {
		fx.In
	Executor *command.Executor
	Logger   *zap.Logger
}

// NewNodeJSProjectDetector creates a new Node.js project detector.
func NewNodeJSProjectDetector(p NodeJSProjectDetectorParams) service.ProjectDetector {
	return &NodeJSProjectDetector{
		executor: p.Executor,
		logger:   p.Logger.Named("nodejs-detector"),
	}
}

// Detect detects Node.js project information.
func (d *NodeJSProjectDetector) Detect(ctx context.Context, worktreePath string) (*valueobject.DetectedProject, error) {
	// 1. Check package.json exists
	if !d.executor.CheckFileExists(worktreePath, "package.json") {
		return nil, nil // Not a Node.js project
	}

	project := valueobject.NewDetectedProject(valueobject.ProjectTypeNodeJS, "javascript", 90)

	// Parse package.json for dependencies
	pkg := d.parsePackageJSON(worktreePath)
	if pkg != nil {
		// Add language based on type field
		if pkg.Type == "module" || pkg.Type == "commonjs" {
			project.PrimaryLanguage = "javascript"
		}
		// Check for TypeScript
		if d.executor.CheckFileExists(worktreePath, "tsconfig.json") {
			project.PrimaryLanguage = "typescript"
			project.AddLanguage("typescript")
		}
	}

	// 2. Detect formatter
	d.detectFormatter(worktreePath, project, pkg)

	// 3. Detect linter
	d.detectLinter(worktreePath, project, pkg)

	// 4. Detect test framework
	d.detectTestFramework(worktreePath, project, pkg)

	// 5. Check Makefile (may override defaults)
	d.detectMakefileTargets(worktreePath, project)

	return project, nil
}

// SupportsLanguage returns true for "nodejs", "node", "javascript", "typescript" languages.
func (d *NodeJSProjectDetector) SupportsLanguage(language string) bool {
	lang := strings.ToLower(language)
	return lang == "nodejs" || lang == "node" || lang == "javascript" || lang == "typescript"
}

// Priority returns priority for Node.js projects.
func (d *NodeJSProjectDetector) Priority() int {
	return 90
}

// packageJSON represents package.json structure.
type packageJSON struct {
	Type          string            `json:"type"`
	Dependencies  map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	Scripts       map[string]string `json:"scripts"`
}

// parsePackageJSON parses package.json file.
func (d *NodeJSProjectDetector) parsePackageJSON(worktreePath string) *packageJSON {
	content, err := d.executor.ReadFile(worktreePath, "package.json")
	if err != nil {
		d.logger.Warn("failed to read package.json", zap.Error(err))
		return nil
	}

	var pkg packageJSON
	if err := json.Unmarshal(content, &pkg); err != nil {
		d.logger.Warn("failed to parse package.json", zap.Error(err))
		return nil
	}

	return &pkg
}

// hasDependency checks if a dependency exists in package.json.
func (d *NodeJSProjectDetector) hasDependency(pkg *packageJSON, name string) bool {
	if pkg == nil {
		return false
	}
	_, ok := pkg.Dependencies[name]
	if ok {
		return true
	}
	_, ok = pkg.DevDependencies[name]
	return ok
}

// detectFormatter detects formatting tool.
func (d *NodeJSProjectDetector) detectFormatter(worktreePath string, project *valueobject.DetectedProject, pkg *packageJSON) {
	// Check for prettier config files
	prettierConfigs := []string{".prettierrc", ".prettierrc.json", ".prettierrc.yaml", ".prettierrc.yml", "prettier.config.js"}

	for _, configFile := range prettierConfigs {
		if d.executor.CheckFileExists(worktreePath, configFile) {
			tool := valueobject.NewDetectedTool(
				valueobject.ToolTypeFormatter,
				"prettier",
				configFile+" exists",
				valueobject.NewToolCommand("prettier", "npx", []string{"prettier", "--write", "."}, 120).
					WithConfigCheck(configFile),
			).WithConfigFile(configFile)
			project.AddTool(tool)
			return
		}
	}

	// Check for prettier in dependencies
	if d.hasDependency(pkg, "prettier") {
		project.AddTool(valueobject.NewDetectedTool(
			valueobject.ToolTypeFormatter,
			"prettier",
			"prettier in dependencies",
			valueobject.NewToolCommand("prettier", "npx", []string{"prettier", "--write", "."}, 120),
		))
		return
	}

	// Check for biome
	if d.executor.CheckFileExists(worktreePath, "biome.json") {
		project.AddTool(valueobject.NewDetectedTool(
			valueobject.ToolTypeFormatter,
			"biome",
			"biome.json exists",
			valueobject.NewToolCommand("biome", "npx", []string{"biome", "check", "--apply", "."}, 120).
				WithConfigCheck("biome.json"),
		))
		return
	}
}

// detectLinter detects linting tool.
func (d *NodeJSProjectDetector) detectLinter(worktreePath string, project *valueobject.DetectedProject, pkg *packageJSON) {
	// Check for eslint config files
	eslintConfigs := []string{".eslintrc", ".eslintrc.json", ".eslintrc.yaml", ".eslintrc.yml", ".eslintrc.js"}

	for _, configFile := range eslintConfigs {
		if d.executor.CheckFileExists(worktreePath, configFile) {
			tool := valueobject.NewDetectedTool(
				valueobject.ToolTypeLinter,
				"eslint",
				configFile+" exists",
				valueobject.NewToolCommand("eslint", "npx", []string{"eslint", "--fix", "."}, 120).
					WithConfigCheck(configFile),
			).WithConfigFile(configFile)
			project.AddTool(tool)
			return
		}
	}

	// Check for eslint in dependencies
	if d.hasDependency(pkg, "eslint") {
		project.AddTool(valueobject.NewDetectedTool(
			valueobject.ToolTypeLinter,
			"eslint",
			"eslint in dependencies",
			valueobject.NewToolCommand("eslint", "npx", []string{"eslint", "--fix", "."}, 120),
		))
		return
	}

	// Biome as linter
	if d.executor.CheckFileExists(worktreePath, "biome.json") {
		// Already added as formatter, skip
		return
	}
}

// detectTestFramework detects test framework.
func (d *NodeJSProjectDetector) detectTestFramework(worktreePath string, project *valueobject.DetectedProject, pkg *packageJSON) {
	// Check for jest
	if d.hasDependency(pkg, "jest") {
		project.AddTool(valueobject.NewDetectedTool(
			valueobject.ToolTypeTester,
			"jest",
			"jest in dependencies",
			valueobject.NewToolCommand("jest", "npx", []string{"jest", "--passWithNoTests"}, 300),
		))
		return
	}

	// Check for vitest
	if d.hasDependency(pkg, "vitest") {
		project.AddTool(valueobject.NewDetectedTool(
			valueobject.ToolTypeTester,
			"vitest",
			"vitest in dependencies",
			valueobject.NewToolCommand("vitest", "npx", []string{"vitest", "run", "--passWithNoTests"}, 300),
		))
		return
	}

	// Check for mocha
	if d.hasDependency(pkg, "mocha") {
		project.AddTool(valueobject.NewDetectedTool(
			valueobject.ToolTypeTester,
			"mocha",
			"mocha in dependencies",
			valueobject.NewToolCommand("mocha", "npx", []string{"mocha"}, 300),
		))
		return
	}

	// Default npm test
	project.AddTool(valueobject.NewDetectedTool(
		valueobject.ToolTypeTester,
		"npm test",
		"nodejs project default",
		valueobject.NewToolCommand("npm test", "npm", []string{"test"}, 300),
	))
}

// detectMakefileTargets detects Makefile targets.
func (d *NodeJSProjectDetector) detectMakefileTargets(worktreePath string, project *valueobject.DetectedProject) {
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
func (d *NodeJSProjectDetector) parseMakefileTargets(content string) map[string]bool {
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
func (d *NodeJSProjectDetector) removeToolsByType(tools []valueobject.DetectedTool, toolType valueobject.ToolType) []valueobject.DetectedTool {
	return slices.DeleteFunc(tools, func(t valueobject.DetectedTool) bool {
		return t.Type == toolType
	})
}