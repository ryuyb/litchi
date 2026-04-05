// Package parser provides output parsing for Agent responses.
package parser

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/service"
	"github.com/ryuyb/litchi/internal/pkg/errors"
)

// OutputParser defines the interface for parsing Agent output.
type OutputParser interface {
	// Parse parses raw output into an AgentResponse.
	Parse(rawOutput string, stage service.AgentStage) (*service.AgentResponse, error)

	// ParseJSON parses JSON format output.
	ParseJSON(jsonStr string) (*StructuredOutput, error)

	// ParseText parses plain text format output.
	ParseText(text string) (*TextOutput, error)

	// ParseError parses error output.
	ParseError(stderr string, exitCode int) *service.AgentErrorInfo
}

// StructuredOutput represents structured JSON output from Claude Code.
type StructuredOutput struct {
	Type           string                 `json:"type"`
	Content        string                 `json:"content"`
	StructuredData map[string]any         `json:"structured_data,omitempty"`
	ToolCalls      []ToolCallJSON         `json:"tool_calls,omitempty"`
	FilesChanged   []FileChangeJSON       `json:"files_changed,omitempty"`
	TestsRun       []TestResultJSON       `json:"tests_run,omitempty"`
	TokensUsed     int                    `json:"tokens_used,omitempty"`
	NeedsApproval  bool                   `json:"needs_approval,omitempty"`
}

// ToolCallJSON represents a tool call in JSON output.
type ToolCallJSON struct {
	Timestamp string `json:"timestamp"`
	ToolName  string `json:"tool_name"`
	Input     string `json:"input"`
	Output    string `json:"output"`
	Success   bool   `json:"success"`
	Blocked   bool   `json:"blocked"`
}

// FileChangeJSON represents a file change in JSON output.
type FileChangeJSON struct {
	Path        string `json:"path"`
	Action      string `json:"action"`
	LinesAdded  int    `json:"lines_added"`
	LinesDeleted int   `json:"lines_deleted"`
}

// TestResultJSON represents a test result in JSON output.
type TestResultJSON struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Message  string `json:"message"`
	Duration int64  `json:"duration"`
}

// TextOutput represents plain text output from Claude Code.
type TextOutput struct {
	Content     string
	FilesMentioned []string
	CommandsRun    []string
}

// DefaultOutputParser is the default implementation of OutputParser.
type DefaultOutputParser struct{}

// NewDefaultOutputParser creates a new default output parser.
func NewDefaultOutputParser() *DefaultOutputParser {
	return &DefaultOutputParser{}
}

// Parse parses raw output into an AgentResponse.
func (p *DefaultOutputParser) Parse(rawOutput string, stage service.AgentStage) (*service.AgentResponse, error) {
	response := &service.AgentResponse{
		Stage:   stage,
		Success: true,
		Output:  rawOutput,
	}

	// Try JSON parsing first
	if strings.HasPrefix(strings.TrimSpace(rawOutput), "{") {
		structured, err := p.ParseJSON(rawOutput)
		if err == nil {
			response.Result = service.AgentResult{
				Type:           structured.Type,
				Content:        structured.Content,
				StructuredData: structured.StructuredData,
			}
			response.TokensUsed = structured.TokensUsed
			response.NeedsApproval = structured.NeedsApproval

			// Convert tool calls
			if len(structured.ToolCalls) > 0 {
				response.ToolCalls = make([]service.ToolCallRecord, len(structured.ToolCalls))
				for i, tc := range structured.ToolCalls {
					t, _ := time.Parse(time.RFC3339, tc.Timestamp)
					response.ToolCalls[i] = service.ToolCallRecord{
						Timestamp:   t,
						ToolName:    tc.ToolName,
						Input:       tc.Input,
						Output:      tc.Output,
						Success:     tc.Success,
						Blocked:     tc.Blocked,
					}
				}
			}

			// Convert files changed
			if len(structured.FilesChanged) > 0 {
				response.Result.FilesChanged = make([]service.FileChange, len(structured.FilesChanged))
				for i, fc := range structured.FilesChanged {
					response.Result.FilesChanged[i] = service.FileChange{
						Path:         fc.Path,
						Action:       fc.Action,
						LinesAdded:   fc.LinesAdded,
						LinesDeleted: fc.LinesDeleted,
					}
				}
			}

			// Convert tests run
			if len(structured.TestsRun) > 0 {
				response.Result.TestsRun = make([]service.TestResult, len(structured.TestsRun))
				for i, tr := range structured.TestsRun {
					response.Result.TestsRun[i] = service.TestResult{
						Name:     tr.Name,
						Status:   tr.Status,
						Message:  tr.Message,
						Duration: time.Duration(tr.Duration) * time.Millisecond,
					}
				}
			}

			return response, nil
		}
	}

	// Fall back to text parsing
	textOutput, err := p.ParseText(rawOutput)
	if err == nil {
		response.Result.Content = textOutput.Content
	}

	return response, nil
}

// ParseJSON parses JSON format output.
func (p *DefaultOutputParser) ParseJSON(jsonStr string) (*StructuredOutput, error) {
	var output StructuredOutput
	if err := json.Unmarshal([]byte(jsonStr), &output); err != nil {
		return nil, err
	}
	return &output, nil
}

// ParseText parses plain text format output.
func (p *DefaultOutputParser) ParseText(text string) (*TextOutput, error) {
	output := &TextOutput{
		Content: text,
	}

	// Extract file mentions with improved accuracy
	// Pattern matches file paths with extensions, excluding:
	// - Version numbers (v1.0.0)
	// - URLs (http://...)
	// - Numbers with decimals
	// - Short extensions like single letters
	output.FilesMentioned = extractFileMentions(text)

	// Extract commands (e.g., "go test -v", "npm run build", "pip install requests")
	// Matches: command + arguments until line end or common terminators
	cmdRegex := regexp.MustCompile(`(?:go|npm|yarn|pnpm|cargo|python|pip)\s+[^\n\.;!?]+`)
	cmdMatches := cmdRegex.FindAllString(text, -1)
	output.CommandsRun = uniqueStrings(cmdMatches)

	return output, nil
}

// extractFileMentions extracts file paths from text with reduced false positives.
func extractFileMentions(text string) []string {
	// Common file extensions we care about
	commonExts := map[string]bool{
		// Programming languages
		".go": true, ".py": true, ".js": true, ".ts": true, ".jsx": true, ".tsx": true,
		".java": true, ".kt": true, ".rs": true, ".rb": true, ".php": true, ".c": true,
		".cpp": true, ".h": true, ".hpp": true, ".cs": true, ".swift": true, ".scala": true,
		// Config/Data
		".json": true, ".yaml": true, ".yml": true, ".toml": true, ".xml": true,
		".ini": true, ".cfg": true, ".conf": true, ".env": true,
		// Web/Markup
		".html": true, ".css": true, ".scss": true, ".sass": true, ".less": true,
		".md": true, ".markdown": true, ".rst": true, ".txt": true,
		// Shell/Scripts
		".sh": true, ".bash": true, ".zsh": true, ".fish": true,
		".ps1": true, ".bat": true, ".cmd": true,
		// Build/Package
		".mod": true, ".sum": true, ".lock": true, ".Dockerfile": true,
		".make": true, ".mk": true,
	}

	// Regex that matches potential file paths
	// Requires: word chars, dots, slashes, hyphens, underscores
	// Must end with a dot and extension (2-10 chars)
	fileRegex := regexp.MustCompile(`(?i)(?:^|[\s"'(\[{,])([\w][\w/.-]*\.[a-zA-Z]{2,10})(?:$|[\s"')\]},.!?;:])`)

	matches := fileRegex.FindAllStringSubmatch(text, -1)
	var files []string

	for _, match := range matches {
		if len(match) > 1 {
			candidate := match[1]

			// Skip version patterns like v1.0.0
			if regexp.MustCompile(`^v\d+\.\d+`).MatchString(candidate) {
				continue
			}

			// Skip URL patterns
			if regexp.MustCompile(`^https?://`).MatchString(candidate) {
				continue
			}

			// Skip decimal numbers
			if regexp.MustCompile(`^\d+\.\d+$`).MatchString(candidate) {
				continue
			}

			// Check if it has a known extension (optional filter for higher precision)
			ext := ""
			if idx := regexp.MustCompile(`\.[a-zA-Z]+$`).FindStringIndex(candidate); idx != nil {
				ext = candidate[idx[0]:]
			}

			// Accept if it's a known extension OR if it looks like a path
			if commonExts[ext] || strings.Contains(candidate, "/") || strings.Contains(candidate, "\\") {
				files = append(files, candidate)
			}
		}
	}

	return uniqueStrings(files)
}

// ParseError parses error output.
func (p *DefaultOutputParser) ParseError(stderr string, exitCode int) *service.AgentErrorInfo {
	errInfo := &service.AgentErrorInfo{
		Code:     errors.ErrAgentExecutionFail.Code,
		Category: "execution",
		Message:  errors.ErrAgentExecutionFail.Message,
		Detail:   stderr,
	}

	// Determine error category based on exit code
	switch exitCode {
	case 137: // SIGKILL
		errInfo.Category = "timeout"
		errInfo.Code = errors.ErrAgentTimeout.Code
		errInfo.Message = errors.ErrAgentTimeout.Message
		errInfo.Retryable = true
	case 143: // SIGTERM
		errInfo.Category = "process"
		errInfo.Code = errors.ErrAgentProcessCrash.Code
		errInfo.Message = errors.ErrAgentProcessCrash.Message
		errInfo.Retryable = true
	case 1:
		// General error, check stderr for details
		if strings.Contains(stderr, "permission") {
			errInfo.Category = "permission"
			errInfo.Code = errors.ErrAgentPermissionDenied.Code
			errInfo.Message = errors.ErrAgentPermissionDenied.Message
			errInfo.Retryable = false
		}
	}

	return errInfo
}

// uniqueStrings returns unique strings from a slice.
func uniqueStrings(strs []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, s := range strs {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// ParseJSONLines parses multiple JSON lines from output.
func ParseJSONLines(output string) []StructuredOutput {
	lines := strings.Split(output, "\n")
	var results []StructuredOutput

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "{") {
			var parsed StructuredOutput
			if err := json.Unmarshal([]byte(line), &parsed); err == nil {
				results = append(results, parsed)
			}
		}
	}

	return results
}

// ExtractSessionID extracts session ID from environment or output.
func ExtractSessionID(output string) uuid.UUID {
	// Try to find session ID in output
	re := regexp.MustCompile(`session[_-]?id[:\s]*([a-fA-F0-9-]{36})`)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		if id, err := uuid.Parse(matches[1]); err == nil {
			return id
		}
	}
	return uuid.Nil
}