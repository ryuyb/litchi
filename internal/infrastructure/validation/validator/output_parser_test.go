package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultOutputParser_ParseFormatOutput(t *testing.T) {
	parser := NewDefaultOutputParser()

	// Test gofmt - silent on success
	success, msg := parser.ParseFormatOutput("", "gofmt")
	assert.True(t, success)
	assert.Equal(t, "formatting successful", msg)

	// Test prettier error
	success, msg = parser.ParseFormatOutput("error: syntax error", "prettier")
	assert.False(t, success)

	// Test prettier success
	success, msg = parser.ParseFormatOutput("", "prettier")
	assert.True(t, success)

	// Test cargo fmt
	success, msg = parser.ParseFormatOutput("", "cargo fmt")
	assert.True(t, success)
	assert.Equal(t, "formatting successful", msg)
}

func TestDefaultOutputParser_ParseLintOutput(t *testing.T) {
	parser := NewDefaultOutputParser()

	// Golangci-lint
	issues, fixed, _ := parser.ParseLintOutput("", "golangci-lint")
	assert.Equal(t, 0, issues)
	assert.Equal(t, 0, fixed)

	output := "main.go:10: unused variable\nmain.go:20: syntax error"
	issues, fixed, msg := parser.ParseLintOutput(output, "golangci-lint")
	assert.Equal(t, 2, issues)
	assert.Equal(t, 0, fixed)
	assert.NotEmpty(t, msg)

	// Go vet
	issues, fixed, _ = parser.ParseLintOutput("main.go:10: printf format error", "go vet")
	assert.Equal(t, 1, issues)
	assert.Equal(t, 0, fixed)

	// Ruff
	output = "main.py:10: unused import\nmain.py:20: syntax error"
	issues, fixed, _ = parser.ParseLintOutput(output, "ruff")
	assert.Equal(t, 2, issues)
	assert.Equal(t, 0, fixed)

	// Clippy
	output = "warning: unused variable\nerror: syntax error"
	issues, fixed, _ = parser.ParseLintOutput(output, "clippy")
	assert.Equal(t, 2, issues)
	assert.Equal(t, 0, fixed)
}

func TestDefaultOutputParser_ParseEslintOutput(t *testing.T) {
	parser := NewDefaultOutputParser()

	// Test ESLint parsing directly
	issues := parser.parseEslintIssues("0 problems")
	assert.Equal(t, 0, issues)

	issues = parser.parseEslintIssues("5 problems (2 errors, 3 warnings)")
	assert.Equal(t, 5, issues)
}

func TestDefaultOutputParser_ParseTestOutput_GoTest(t *testing.T) {
	parser := NewDefaultOutputParser()

	// Test parseGoTestOutput directly
	output := "PASS\nok  github.com/test  0.123s"
	passed, failed, failures := parser.parseGoTestOutput(output)
	assert.Equal(t, 2, passed) // PASS line + ok line
	assert.Equal(t, 0, failed)
	assert.Empty(t, failures)
}

func TestDefaultOutputParser_ParseTestOutput_Jest(t *testing.T) {
	parser := NewDefaultOutputParser()

	// Test parseJestOutput directly
	output := "Tests: 5 passed"
	passed, failed, _ := parser.parseJestOutput(output)
	assert.Equal(t, 5, passed)
	assert.Equal(t, 0, failed)

	output = "Tests: 3 passed, 2 failed"
	passed, failed, _ = parser.parseJestOutput(output)
	assert.Equal(t, 3, passed)
	assert.Equal(t, 2, failed)
}

func TestDefaultOutputParser_ParseTestOutput_Pytest(t *testing.T) {
	parser := NewDefaultOutputParser()

	// Test parsePytestOutput directly
	output := "5 passed"
	passed, failed, _ := parser.parsePytestOutput(output)
	assert.Equal(t, 5, passed)
	assert.Equal(t, 0, failed)

	output = "3 passed, 2 failed"
	passed, failed, _ = parser.parsePytestOutput(output)
	assert.Equal(t, 3, passed)
	assert.Equal(t, 2, failed)
}

func TestDefaultOutputParser_ParseTestOutput_CargoDirect(t *testing.T) {
	parser := NewDefaultOutputParser()

	// Test parseCargoTestOutput directly
	output := "test result: ok. 5 passed"
	passed, failed, _ := parser.parseCargoTestOutput(output)
	assert.Equal(t, 5, passed)
	assert.Equal(t, 0, failed)

	output = "3 passed; 2 failed"
	passed, failed, _ = parser.parseCargoTestOutput(output)
	assert.Equal(t, 3, passed)
	assert.Equal(t, 2, failed)
}

func TestParseInt(t *testing.T) {
	assert.Equal(t, 123, parseInt("123"))
	assert.Equal(t, 456, parseInt("  456  "))
	assert.Equal(t, 78, parseInt("78 problems"))
	assert.Equal(t, 0, parseInt(""))
	assert.Equal(t, 0, parseInt("abc"))
}

func TestDefaultOutputParser_SupportsTool(t *testing.T) {
	parser := NewDefaultOutputParser()
	assert.True(t, parser.SupportsTool("gofmt"))
	assert.True(t, parser.SupportsTool("eslint"))
	assert.True(t, parser.SupportsTool("unknown-tool"))
}

func TestDefaultOutputParser_countLines(t *testing.T) {
	parser := NewDefaultOutputParser()

	// Empty output
	assert.Equal(t, 0, parser.countLines("", "", false))

	// Count non-empty lines
	output := "line1\nline2\n\nline3"
	assert.Equal(t, 3, parser.countLines(output, "", false))

	// Count lines with pattern (required)
	output = "main.go:10: error\nmain.go:20: warning\nno match"
	assert.Equal(t, 2, parser.countLines(output, ":", true))
}

func TestDefaultOutputParser_parseGoTestOutput(t *testing.T) {
	parser := NewDefaultOutputParser()

	// PASS line
	output := "PASS\nok  pkg  0.1s"
	passed, failed, failures := parser.parseGoTestOutput(output)
	assert.Equal(t, 2, passed) // PASS + ok line
	assert.Equal(t, 0, failed)
	assert.Empty(t, failures)

	// FAIL line with test name
	output = "FAIL TestName\nPASS"
	passed, failed, failures = parser.parseGoTestOutput(output)
	assert.Equal(t, 1, passed)
	assert.Equal(t, 1, failed)
	require.NotEmpty(t, failures)
	assert.Equal(t, "TestName", failures[0].Name)
}