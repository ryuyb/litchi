package validator

import (
	"regexp"
	"strings"

	"github.com/ryuyb/litchi/internal/domain/valueobject"
)

// Precompiled regex patterns for performance
var (
	// ESLint patterns
	eslintProblemsRe = regexp.MustCompile(`(\d+)\s+problems?`)
	eslintDetailsRe  = regexp.MustCompile(`(\d+)\s+problems?\s+.*\((\d+)\s+errors?,\s*(\d+)\s+warnings?`)

	// Go test patterns
	goTestPassRe = regexp.MustCompile(`^ok\s+`)
	goTestFailRe = regexp.MustCompile(`^FAIL\s+`)

	// Jest/Vitest patterns
	jestTestsRe = regexp.MustCompile(`Tests?:\s*(\d+)\s*(?:passed|failed)?,?\s*(\d+)?\s*(?:passed|failed)?`)

	// pytest patterns
	pytestPassedRe = regexp.MustCompile(`(\d+)\s+passed`)
	pytestFailedRe = regexp.MustCompile(`(\d+)\s+failed`)

	// Cargo test patterns
	cargoPassedRe = regexp.MustCompile(`(\d+)\s+passed`)
	cargoFailedRe = regexp.MustCompile(`(\d+)\s+failed`)
)

// DefaultOutputParser parses output from common tools.
type DefaultOutputParser struct{}

// NewDefaultOutputParser creates a new default output parser.
func NewDefaultOutputParser() *DefaultOutputParser {
	return &DefaultOutputParser{}
}

// ParseFormatOutput parses formatting tool output.
func (p *DefaultOutputParser) ParseFormatOutput(output string, toolName string) (bool, string) {
	toolLower := strings.ToLower(toolName)

	switch {
	case strings.Contains(toolLower, "gofmt"):
		// gofmt -w outputs nothing on success
		return true, "formatting successful"

	case strings.Contains(toolLower, "prettier"):
		// prettier outputs file list on change
		if strings.Contains(output, "error") || strings.Contains(output, "Error") {
			return false, output
		}
		return true, "formatting successful"

	case strings.Contains(toolLower, "black"):
		// black outputs "reformatted X files"
		if strings.Contains(output, "error") || strings.Contains(output, "Error") {
			return false, output
		}
		return true, output

	case strings.Contains(toolLower, "cargo fmt"):
		// cargo fmt is silent on success
		return true, "formatting successful"

	default:
		// Generic check for error indicators
		if strings.Contains(output, "error") || strings.Contains(output, "Error") || strings.Contains(output, "failed") {
			return false, output
		}
		return true, "formatting completed"
	}
}

// ParseLintOutput parses lint tool output.
func (p *DefaultOutputParser) ParseLintOutput(output string, toolName string) (int, int, string) {
	toolLower := strings.ToLower(toolName)
	issuesFound := 0
	issuesFixed := 0

	switch {
	case strings.Contains(toolLower, "golangci-lint"):
		// Count issues in output
		issuesFound = p.countLines(output, ":", true) // Lines with : typically indicate issues
		if issuesFound == 0 && output != "" {
			// No issues found
			return 0, 0, "no issues found"
		}
		return issuesFound, issuesFixed, output

	case strings.Contains(toolLower, "go vet"):
		// go vet outputs issues
		issuesFound = p.countLines(output, ":", true)
		return issuesFound, 0, output

	case strings.Contains(toolLower, "eslint"):
		// ESLint outputs problems count
		issuesFound = p.parseEslintIssues(output)
		issuesFixed = p.parseEslintFixed(output)
		return issuesFound, issuesFixed, output

	case strings.Contains(toolLower, "ruff"):
		// Ruff outputs issues
		issuesFound = p.countLines(output, ":", true)
		return issuesFound, issuesFixed, output

	case strings.Contains(toolLower, "flake8"):
		issuesFound = p.countLines(output, ":", true)
		return issuesFound, 0, output

	case strings.Contains(toolLower, "clippy"):
		// Clippy outputs warnings/errors
		issuesFound = p.countLines(output, "warning:", false) + p.countLines(output, "error:", false)
		return issuesFound, 0, output

	default:
		// Generic counting
		if output != "" {
			issuesFound = p.countLines(output, "", false)
		}
		return issuesFound, issuesFixed, output
	}
}

// ParseTestOutput parses test tool output.
func (p *DefaultOutputParser) ParseTestOutput(output string, toolName string) (int, int, []valueobject.TestFailure) {
	toolLower := strings.ToLower(toolName)

	// Debug: check each condition
	if strings.Contains(toolLower, "go test") {
		return p.parseGoTestOutput(output)
	}
	if strings.Contains(toolLower, "jest") {
		return p.parseJestOutput(output)
	}
	if strings.Contains(toolLower, "vitest") {
		return p.parseVitestOutput(output)
	}
	if strings.Contains(toolLower, "pytest") {
		return p.parsePytestOutput(output)
	}
	if strings.Contains(toolLower, "cargo test") {
		return p.parseCargoTestOutput(output)
	}

	// Generic parsing
	passed := 0
	failed := 0
	if strings.Contains(output, "PASS") || strings.Contains(output, "passed") {
		passed = 1
	}
	if strings.Contains(output, "FAIL") || strings.Contains(output, "failed") {
		failed = 1
	}
	return passed, failed, nil
}

// SupportsTool checks if this parser supports the given tool.
func (p *DefaultOutputParser) SupportsTool(toolName string) bool {
	// Default parser supports all tools
	return true
}

// Helper methods

// countLines counts lines matching a pattern.
func (p *DefaultOutputParser) countLines(output string, pattern string, requirePattern bool) int {
	count := 0
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if pattern == "" {
			count++
		} else if strings.Contains(line, pattern) {
			count++
		} else if !requirePattern && len(line) > 0 {
			// Count non-empty lines if pattern is not required
		}
	}

	return count
}

// parseEslintIssues parses ESLint issues count.
func (p *DefaultOutputParser) parseEslintIssues(output string) int {
	// Look for "X problems (Y errors, Z warnings)"
	match := eslintProblemsRe.FindStringSubmatch(output)
	if len(match) > 1 {
		return parseInt(match[1])
	}
	return 0
}

// parseEslintFixed parses ESLint fixed count.
// Note: ESLint standard output doesn't include "fixed" count directly.
// This function looks for ESLint's --fix output format which may indicate fixed issues.
// Returns 0 if no fix information is found.
func (p *DefaultOutputParser) parseEslintFixed(output string) int {
	// ESLint with --fix outputs something like:
	// "Fixed X problems in Y files"
	// Without --fix, we can't determine how many were auto-fixed
	match := regexp.MustCompile(`Fixed\s+(\d+)\s+problems?`).FindStringSubmatch(output)
	if len(match) > 1 {
		return parseInt(match[1])
	}
	return 0
}

// parseGoTestOutput parses go test output.
func (p *DefaultOutputParser) parseGoTestOutput(output string) (int, int, []valueobject.TestFailure) {
	passed := 0
	failed := 0
	failures := []valueobject.TestFailure{}

	// Look for "PASS" and "FAIL" lines
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "PASS") {
			passed++
		}
		if strings.HasPrefix(line, "FAIL") {
			// Extract test name
			parts := strings.Fields(line)
			if len(parts) > 1 {
				failures = append(failures, valueobject.TestFailure{
					Name:    parts[1],
					Message: line,
				})
			}
			failed++
		}
	}

	// Also check summary line like "ok  github.com/...  0.123s"
	for _, line := range lines {
		if goTestPassRe.MatchString(line) {
			passed++
		}
		if goTestFailRe.MatchString(line) && !strings.Contains(line, "FAIL\t") {
			// Not a FAIL line from summary
		}
	}

	return passed, failed, failures
}

// parseJestOutput parses Jest output.
func (p *DefaultOutputParser) parseJestOutput(output string) (int, int, []valueobject.TestFailure) {
	// Look for "Tests: X passed, Y failed"
	match := jestTestsRe.FindStringSubmatch(output)

	if len(match) > 1 {
		passed := parseInt(match[1])
		failed := 0
		if len(match) > 2 && match[2] != "" {
			failed = parseInt(match[2])
		}
		return passed, failed, nil
	}

	return 0, 0, nil
}

// parseVitestOutput parses Vitest output.
func (p *DefaultOutputParser) parseVitestOutput(output string) (int, int, []valueobject.TestFailure) {
	// Similar to Jest
	return p.parseJestOutput(output)
}

// parsePytestOutput parses pytest output.
func (p *DefaultOutputParser) parsePytestOutput(output string) (int, int, []valueobject.TestFailure) {
	// Look for "X passed, Y failed"
	passed := 0
	match := pytestPassedRe.FindStringSubmatch(output)
	if len(match) > 1 {
		passed = parseInt(match[1])
	}

	failed := 0
	match = pytestFailedRe.FindStringSubmatch(output)
	if len(match) > 1 {
		failed = parseInt(match[1])
	}

	return passed, failed, nil
}

// parseCargoTestOutput parses cargo test output.
func (p *DefaultOutputParser) parseCargoTestOutput(output string) (int, int, []valueobject.TestFailure) {
	// Look for "test result: ok. X passed; Y failed"
	resultPassed := 0
	match := cargoPassedRe.FindStringSubmatch(output)
	if len(match) > 1 {
		resultPassed = parseInt(match[1])
	}

	resultFailed := 0
	match = cargoFailedRe.FindStringSubmatch(output)
	if len(match) > 1 {
		resultFailed = parseInt(match[1])
	}

	return resultPassed, resultFailed, nil
}

// parseInt safely parses an integer.
func parseInt(s string) int {
	var result int
	for _, c := range strings.TrimSpace(s) {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		} else {
			break
		}
	}
	return result
}