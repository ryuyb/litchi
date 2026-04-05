package claude

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// StreamHandler handles real-time output streams.
type StreamHandler struct {
	logger   *zap.Logger
	source   string
	lines    []StreamLine
	mu       sync.Mutex
	callback func(line StreamLine)
}

// StreamLine represents a line in the output stream.
type StreamLine struct {
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`    // stdout, stderr
	Content   string    `json:"content"`   // Raw content
	Type      string    `json:"type"`      // log, tool_use, tool_result, error, result, json
	Parsed    any       `json:"parsed"`    // Parsed JSON if applicable
}

// NewStreamHandler creates a new stream handler.
func NewStreamHandler(logger *zap.Logger, source string) *StreamHandler {
	return &StreamHandler{
		logger:   logger.Named("stream-" + source),
		source:   source,
		lines:    make([]StreamLine, 0),
		callback: nil,
	}
}

// NewStreamHandlerWithCallback creates a stream handler with a callback.
func NewStreamHandlerWithCallback(logger *zap.Logger, source string, callback func(line StreamLine)) *StreamHandler {
	return &StreamHandler{
		logger:   logger.Named("stream-" + source),
		source:   source,
		lines:    make([]StreamLine, 0),
		callback: callback,
	}
}

// Write implements io.Writer interface.
func (h *StreamHandler) Write(p []byte) (n int, err error) {
	scanner := bufio.NewScanner(bytes.NewReader(p))
	for scanner.Scan() {
		content := scanner.Text()

		line := StreamLine{
			Timestamp: time.Now(),
			Source:    h.source,
			Content:   content,
			Type:      h.parseLineType(content),
		}

		// Try to parse as JSON
		if line.Type == "json" {
			var parsed any
			if err := json.Unmarshal([]byte(content), &parsed); err == nil {
				line.Parsed = parsed
			}
		}

		h.mu.Lock()
		h.lines = append(h.lines, line)
		h.mu.Unlock()

		// Call callback if set
		if h.callback != nil {
			h.callback(line)
		}

		// Log the line
		h.logLine(line)
	}

	return len(p), scanner.Err()
}

// parseLineType determines the type of a line based on content.
func (h *StreamHandler) parseLineType(content string) string {
	if len(content) == 0 {
		return "empty"
	}

	// Check if valid JSON (not just starts with { or [)
	if (content[0] == '{' || content[0] == '[') && json.Valid([]byte(content)) {
		return "json"
	}

	// Check for tool use pattern
	if len(content) > 10 {
		prefix := content[:10]
		if prefix == "Tool use: " {
			return "tool_use"
		}
	}

	// Check for tool result pattern
	if len(content) > 12 {
		prefix := content[:12]
		if prefix == "Tool result:" {
			return "tool_result"
		}
	}

	// Check for separators
	if content == "---" || content == "===" || content == "..." {
		return "separator"
	}

	// Check for error indicators
	if containsError(content) {
		return "error"
	}

	// Default to log
	return "log"
}

// containsError checks if content contains error indicators.
func containsError(content string) bool {
	lower := strings.ToLower(content)
	return strings.Contains(lower, "error") ||
		strings.Contains(lower, "failed") ||
		strings.Contains(lower, "exception")
}

// logLine logs a stream line.
func (h *StreamHandler) logLine(line StreamLine) {
	switch line.Type {
	case "error":
		h.logger.Error("stream line",
			zap.String("type", line.Type),
			zap.String("content", truncate(line.Content, 200)),
		)
	case "tool_use", "tool_result":
		h.logger.Info("stream line",
			zap.String("type", line.Type),
			zap.String("content", truncate(line.Content, 200)),
		)
	default:
		h.logger.Debug("stream line",
			zap.String("type", line.Type),
			zap.String("content", truncate(line.Content, 200)),
		)
	}
}

// GetLines returns all captured lines.
func (h *StreamHandler) GetLines() []StreamLine {
	h.mu.Lock()
	defer h.mu.Unlock()

	result := make([]StreamLine, len(h.lines))
	copy(result, h.lines)
	return result
}

// GetLinesByType returns lines filtered by type.
func (h *StreamHandler) GetLinesByType(lineType string) []StreamLine {
	h.mu.Lock()
	defer h.mu.Unlock()

	var result []StreamLine
	for _, line := range h.lines {
		if line.Type == lineType {
			result = append(result, line)
		}
	}
	return result
}

// GetJSONLines returns all JSON lines.
func (h *StreamHandler) GetJSONLines() []StreamLine {
	return h.GetLinesByType("json")
}

// Clear clears all captured lines.
func (h *StreamHandler) Clear() {
	h.mu.Lock()
	h.lines = make([]StreamLine, 0)
	h.mu.Unlock()
}

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}