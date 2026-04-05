package valueobject

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolCommand_Validate(t *testing.T) {
	tests := []struct {
		name        string
		cmd         ToolCommand
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid command",
			cmd: ToolCommand{
				Name:    "gofmt",
				Command: "gofmt",
				Args:    []string{"-w", "."},
			},
			expectError: false,
		},
		{
			name: "empty command",
			cmd: ToolCommand{
				Name:    "empty",
				Command: "",
				Args:    []string{},
			},
			expectError: true,
			errorMsg:    "command cannot be empty",
		},
		{
			name: "dangerous command with semicolon",
			cmd: ToolCommand{
				Name:    "inject",
				Command: "echo;rm",
				Args:    []string{},
			},
			expectError: true,
			errorMsg:    "command contains dangerous characters",
		},
		{
			name: "dangerous command with pipe",
			cmd: ToolCommand{
				Name:    "pipe",
				Command: "cat|grep",
				Args:    []string{},
			},
			expectError: true,
			errorMsg:    "command contains dangerous characters",
		},
		{
			name: "dangerous command with backtick",
			cmd: ToolCommand{
				Name:    "backtick",
				Command: "echo`id`",
				Args:    []string{},
			},
			expectError: true,
			errorMsg:    "command contains dangerous characters",
		},
		{
			name: "dangerous arg with dollar",
			cmd: ToolCommand{
				Name:    "arg-inject",
				Command: "echo",
				Args:    []string{"$(rm -rf /)"},
			},
			expectError: true,
			errorMsg:    "arg contains dangerous characters",
		},
		{
			name: "dangerous arg with redirect",
			cmd: ToolCommand{
				Name:    "arg-redirect",
				Command: "echo",
				Args:    []string{"test>/etc/passwd"},
			},
			expectError: true,
			errorMsg:    "arg contains dangerous characters",
		},
		{
			name: "valid args with flags",
			cmd: ToolCommand{
				Name:    "go test",
				Command: "go",
				Args:    []string{"test", "-v", "./..."},
			},
			expectError: false,
		},
		{
			name: "path traversal in workingDir",
			cmd: ToolCommand{
				Name:       "path-traversal",
				Command:    "echo",
				WorkingDir: "../../../etc",
			},
			expectError: true,
			errorMsg:    "workingDir cannot contain path traversal",
		},
		{
			name: "absolute path in workingDir Unix",
			cmd: ToolCommand{
				Name:       "abs-path",
				Command:    "echo",
				WorkingDir: "/etc/passwd",
			},
			expectError: true,
			errorMsg:    "workingDir must be relative",
		},
		{
			name: "valid relative workingDir",
			cmd: ToolCommand{
				Name:       "relative-path",
				Command:    "echo",
				WorkingDir: "./subdir",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cmd.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestToolCommand_WithEnv(t *testing.T) {
	original := ToolCommand{
		Name:    "test",
		Command: "echo",
		Args:    []string{"hello"},
	}

	env := map[string]string{"FOO": "bar", "BAZ": "qux"}
	newCmd := original.WithEnv(env)

	// Verify new instance has env
	assert.Equal(t, env, newCmd.Env)

	// Verify original is unchanged
	assert.Empty(t, original.Env)

	// Verify other fields are preserved
	assert.Equal(t, original.Name, newCmd.Name)
	assert.Equal(t, original.Command, newCmd.Command)
	assert.Equal(t, original.Args, newCmd.Args)
}

func TestToolCommand_WithWorkingDir(t *testing.T) {
	original := ToolCommand{
		Name:    "test",
		Command: "echo",
		Args:    []string{"hello"},
	}

	newCmd := original.WithWorkingDir("/tmp/work")

	// Verify new instance has working dir
	assert.Equal(t, "/tmp/work", newCmd.WorkingDir)

	// Verify original is unchanged
	assert.Empty(t, original.WorkingDir)
}

func TestToolCommand_WithConfigCheck(t *testing.T) {
	original := ToolCommand{
		Name:    "test",
		Command: "echo",
		Args:    []string{"hello"},
	}

	newCmd := original.WithConfigCheck(".golangci.yml")

	// Verify new instance has config check
	assert.Equal(t, ".golangci.yml", newCmd.CheckConfigFile)

	// Verify original is unchanged
	assert.Empty(t, original.CheckConfigFile)
}

func TestNewToolCommand(t *testing.T) {
	cmd := NewToolCommand("gofmt", "gofmt", []string{"-w", "."}, 60)

	assert.Equal(t, "gofmt", cmd.Name)
	assert.Equal(t, "gofmt", cmd.Command)
	assert.Equal(t, []string{"-w", "."}, cmd.Args)
	assert.Equal(t, 60, cmd.Timeout)
	assert.Empty(t, cmd.Env)
	assert.Empty(t, cmd.WorkingDir)
	assert.Empty(t, cmd.CheckConfigFile)
}

func TestNewDetectedTool(t *testing.T) {
	cmd := NewToolCommand("gofmt", "gofmt", []string{"-w", "."}, 60)
	tool := NewDetectedTool(ToolTypeFormatter, "gofmt", ".go files detected", cmd)

	assert.Equal(t, ToolTypeFormatter, tool.Type)
	assert.Equal(t, "gofmt", tool.Name)
	assert.Equal(t, ".go files detected", tool.DetectionBasis)
	assert.Equal(t, cmd, tool.RecommendedCommand)
}

func TestDetectedTool_WithConfigFile(t *testing.T) {
	original := NewDetectedTool(ToolTypeFormatter, "golangci-lint", "detected", NewToolCommand("lint", "golangci-lint", []string{"run"}, 120))

	newTool := original.WithConfigFile(".golangci.yml")

	// Verify new instance has config file
	assert.Equal(t, ".golangci.yml", newTool.ConfigFile)

	// Verify original is unchanged
	assert.Empty(t, original.ConfigFile)
}

func TestNewDetectedProject(t *testing.T) {
	project := NewDetectedProject(ProjectTypeGo, "Go", 95)

	assert.Equal(t, ProjectTypeGo, project.Type)
	assert.Equal(t, "Go", project.PrimaryLanguage)
	assert.Equal(t, []string{"Go"}, project.Languages)
	assert.Empty(t, project.DetectedTools)
	assert.Equal(t, 95, project.Confidence)
	assert.False(t, project.DetectedAt.IsZero())
}

func TestDetectedProject_AddLanguage(t *testing.T) {
	project := NewDetectedProject(ProjectTypeMixed, "Go", 80)

	// Add new language
	project.AddLanguage("TypeScript")
	assert.Len(t, project.Languages, 2)
	assert.Contains(t, project.Languages, "TypeScript")

	// Add duplicate - should not add again
	project.AddLanguage("Go")
	assert.Len(t, project.Languages, 2)
}

func TestDetectedProject_AddTool(t *testing.T) {
	project := NewDetectedProject(ProjectTypeGo, "Go", 95)
	tool := NewDetectedTool(ToolTypeFormatter, "gofmt", "detected", NewToolCommand("fmt", "gofmt", []string{"-w"}, 30))

	project.AddTool(tool)
	assert.Len(t, project.DetectedTools, 1)
	assert.Equal(t, tool, project.DetectedTools[0])
}

func TestDetectedProject_GetToolsByType(t *testing.T) {
	project := NewDetectedProject(ProjectTypeGo, "Go", 95)

	formatTool := NewDetectedTool(ToolTypeFormatter, "gofmt", "detected", NewToolCommand("fmt", "gofmt", []string{}, 30))
	lintTool := NewDetectedTool(ToolTypeLinter, "golangci-lint", "detected", NewToolCommand("lint", "golangci-lint", []string{}, 60))
	testTool := NewDetectedTool(ToolTypeTester, "go test", "detected", NewToolCommand("test", "go", []string{"test"}, 120))

	project.AddTool(formatTool)
	project.AddTool(lintTool)
	project.AddTool(testTool)

	// Get by type
	formatTools := project.GetToolsByType(ToolTypeFormatter)
	assert.Len(t, formatTools, 1)
	assert.Equal(t, formatTool, formatTools[0])

	lintTools := project.GetToolsByType(ToolTypeLinter)
	assert.Len(t, lintTools, 1)

	testTools := project.GetToolsByType(ToolTypeTester)
	assert.Len(t, testTools, 1)

	// Non-existing type
	otherTools := project.GetToolsByType(ToolType("other"))
	assert.Empty(t, otherTools)
}

func TestDetectedProject_HasToolType(t *testing.T) {
	project := NewDetectedProject(ProjectTypeGo, "Go", 95)

	assert.False(t, project.HasToolType(ToolTypeFormatter))

	project.AddTool(NewDetectedTool(ToolTypeFormatter, "gofmt", "detected", NewToolCommand("fmt", "gofmt", []string{}, 30)))
	assert.True(t, project.HasToolType(ToolTypeFormatter))
	assert.False(t, project.HasToolType(ToolTypeLinter))
}

func TestNewValidationResult(t *testing.T) {
	result := NewValidationResult()

	assert.Equal(t, ValidationPassed, result.OverallStatus)
	assert.Empty(t, result.Warnings)
	assert.Nil(t, result.FormatResult)
	assert.Nil(t, result.LintResult)
	assert.Nil(t, result.TestResult)
}

func TestValidationResult_AddWarning(t *testing.T) {
	result := NewValidationResult()

	result.AddWarning("first warning")
	assert.Len(t, result.Warnings, 1)
	assert.Equal(t, "first warning", result.Warnings[0])

	result.AddWarning("second warning")
	assert.Len(t, result.Warnings, 2)
}

func TestValidationResult_IsPassed(t *testing.T) {
	result := NewValidationResult()
	assert.True(t, result.IsPassed())

	result.OverallStatus = ValidationFailed
	assert.False(t, result.IsPassed())
}

func TestValidationResult_IsFailed(t *testing.T) {
	result := NewValidationResult()
	assert.False(t, result.IsFailed())

	result.OverallStatus = ValidationFailed
	assert.True(t, result.IsFailed())
}

func TestValidationResult_HasWarnings(t *testing.T) {
	result := NewValidationResult()
	assert.False(t, result.HasWarnings())

	result.AddWarning("warning")
	assert.True(t, result.HasWarnings())
}

func TestNewFormatResult(t *testing.T) {
	result := NewFormatResult(true, "formatting done", "gofmt", 150)

	assert.True(t, result.Success)
	assert.Equal(t, "formatting done", result.Output)
	assert.Equal(t, "gofmt", result.ToolName)
	assert.Equal(t, int64(150), result.Duration)
}

func TestNewLintResult(t *testing.T) {
	result := NewLintResult(true, "lint passed", "golangci-lint", 0, 2, 300)

	assert.True(t, result.Success)
	assert.Equal(t, "lint passed", result.Output)
	assert.Equal(t, "golangci-lint", result.ToolName)
	assert.Equal(t, 0, result.IssuesFound)
	assert.Equal(t, 2, result.IssuesFixed)
	assert.Equal(t, int64(300), result.Duration)
}

func TestNewValidationTestResult(t *testing.T) {
	result := NewValidationTestResult(true, "tests passed", "go test", 10, 0, 500)

	assert.True(t, result.Success)
	assert.Equal(t, "tests passed", result.Output)
	assert.Equal(t, "go test", result.ToolName)
	assert.Equal(t, 10, result.Passed)
	assert.Equal(t, 0, result.Failed)
	assert.Equal(t, int64(500), result.Duration)
	assert.Empty(t, result.Failures)
}

func TestValidationTestResult_AddFailure(t *testing.T) {
	result := NewValidationTestResult(false, "tests failed", "go test", 5, 2, 300)

	result.AddFailure("TestFoo", "expected 5, got 3")
	assert.Len(t, result.Failures, 1)
	assert.Equal(t, "TestFoo", result.Failures[0].Name)
	assert.Equal(t, "expected 5, got 3", result.Failures[0].Message)

	result.AddFailure("TestBar", "nil pointer dereference")
	assert.Len(t, result.Failures, 2)
}