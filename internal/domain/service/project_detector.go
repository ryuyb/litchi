package service

import (
	"context"

	"github.com/ryuyb/litchi/internal/domain/valueobject"
)

// ProjectDetector detects project type and tool configuration.
// Each detector is responsible for a specific language or project type.
type ProjectDetector interface {
	// Detect detects project information from the worktree path.
	// Returns nil if the project type is not detected (not an error).
	// Returns error only on detection failure (e.g., file read error).
	Detect(ctx context.Context, worktreePath string) (*valueobject.DetectedProject, error)

	// SupportsLanguage checks if this detector supports the given language.
	// Used for filtering detectors when only specific languages are needed.
	SupportsLanguage(language string) bool

	// Priority returns the detector priority for ordering.
	// Higher priority detectors are executed first.
	// Standard priorities: Go=100, NodeJS=90, Python=90, Rust=95
	Priority() int
}

// CompositeProjectDetector combines multiple language detectors.
// It executes detectors in priority order and returns the first successful result.
type CompositeProjectDetector interface {
	// DetectWithAll executes all registered detectors in priority order.
	// Returns the first successfully detected project.
	// Returns error if all detectors fail or no project is detected.
	DetectWithAll(ctx context.Context, worktreePath string) (*valueobject.DetectedProject, error)

	// DetectByLanguage executes only the detector for the specified language.
	// Returns nil if no detector supports the language or detection fails.
	DetectByLanguage(ctx context.Context, worktreePath string, language string) (*valueobject.DetectedProject, error)

	// RegisterDetector registers a language detector.
	// Detectors are automatically sorted by priority after registration.
	RegisterDetector(detector ProjectDetector)

	// GetDetectors returns all registered detectors sorted by priority.
	GetDetectors() []ProjectDetector

	// GetSupportedLanguages returns all supported language names.
	GetSupportedLanguages() []string
}