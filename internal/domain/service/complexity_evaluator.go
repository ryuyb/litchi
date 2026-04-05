package service

import (
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/pkg/errors"
)

// DefaultComplexityEvaluator provides default implementation of ComplexityEvaluator.
// This implementation handles the scoring logic with configurable weights.
// The actual AI-based dimension analysis is delegated to an Analyzer function.
//
// Note: The Analyzer function needs to be injected from infrastructure layer
// (typically calling an Agent to analyze design content).
type DefaultComplexityEvaluator struct {
	threshold int
	analyzer  ComplexityAnalyzer
}

// ComplexityAnalyzer is the function type for analyzing design complexity dimensions.
// This function performs the actual content analysis (typically via Agent call)
// and returns raw dimension scores (0-100 for each dimension).
//
// Parameters:
// - design: the design entity to analyze
// - codebaseInfo: contextual information about existing codebase
//
// Returns:
// - ComplexityDimensions: raw dimension scores
// - error: analysis failure
type ComplexityAnalyzer func(design *entity.Design, codebaseInfo *CodebaseInfo) (valueobject.ComplexityDimensions, error)

// NewDefaultComplexityEvaluator creates a new evaluator with default threshold.
// The analyzer must be injected before use.
func NewDefaultComplexityEvaluator(analyzer ComplexityAnalyzer) *DefaultComplexityEvaluator {
	return &DefaultComplexityEvaluator{
		threshold: 70,
		analyzer:  analyzer,
	}
}

// NewComplexityEvaluatorWithThreshold creates an evaluator with custom threshold.
func NewComplexityEvaluatorWithThreshold(threshold int, analyzer ComplexityAnalyzer) (*DefaultComplexityEvaluator, error) {
	if threshold < 0 || threshold > 100 {
		return nil, errors.New(errors.ErrInvalidComplexityScore).WithDetail(
			"threshold must be between 0 and 100",
		)
	}
	return &DefaultComplexityEvaluator{
		threshold: threshold,
		analyzer:  analyzer,
	}, nil
}

// Evaluate analyzes design and returns complexity score with custom weights.
func (e *DefaultComplexityEvaluator) Evaluate(
	design *entity.Design,
	codebaseInfo *CodebaseInfo,
	weights *ComplexityWeights,
) (valueobject.ComplexityScore, valueobject.ComplexityDimensions, error) {
	if design == nil {
		return valueobject.ComplexityScore{}, valueobject.ComplexityDimensions{},
			errors.New(errors.ErrValidationFailed).WithDetail("design cannot be nil")
	}

	if e.analyzer == nil {
		return valueobject.ComplexityScore{}, valueobject.ComplexityDimensions{},
			errors.New(errors.ErrValidationFailed).WithDetail("complexity analyzer not configured")
	}

	// Analyze design to get raw dimension scores
	dimensions, err := e.analyzer(design, codebaseInfo)
	if err != nil {
		return valueobject.ComplexityScore{}, valueobject.ComplexityDimensions{}, err
	}

	// Use default weights if not provided
	if weights == nil {
		defaultWeights := DefaultComplexityWeights()
		weights = &defaultWeights
	}

	// Validate weights
	if !weights.Validate() {
		return valueobject.ComplexityScore{}, valueobject.ComplexityDimensions{},
			errors.New(errors.ErrValidationFailed).WithDetail(
				"invalid weights configuration (must sum to 100)",
			)
	}

	// Calculate weighted score
	score := e.calculateWeightedScore(dimensions, weights)

	// Create ComplexityScore value object
	complexityScore, err := valueobject.NewComplexityScore(score)
	if err != nil {
		return valueobject.ComplexityScore{}, valueobject.ComplexityDimensions{}, err
	}

	return complexityScore, dimensions, nil
}

// EvaluateWithDefaultWeights evaluates using default weight configuration.
func (e *DefaultComplexityEvaluator) EvaluateWithDefaultWeights(
	design *entity.Design,
	codebaseInfo *CodebaseInfo,
) (valueobject.ComplexityScore, valueobject.ComplexityDimensions, error) {
	return e.Evaluate(design, codebaseInfo, nil)
}

// GetThreshold returns the current complexity threshold.
func (e *DefaultComplexityEvaluator) GetThreshold() int {
	return e.threshold
}

// SetThreshold configures the complexity threshold.
func (e *DefaultComplexityEvaluator) SetThreshold(threshold int) error {
	if threshold < 0 || threshold > 100 {
		return errors.New(errors.ErrInvalidComplexityScore).WithDetail(
			"threshold must be between 0 and 100",
		)
	}
	e.threshold = threshold
	return nil
}

// calculateWeightedScore calculates the final score from dimensions and weights.
func (e *DefaultComplexityEvaluator) calculateWeightedScore(
	dimensions valueobject.ComplexityDimensions,
	weights *ComplexityWeights,
) int {
	// Weighted calculation: score = sum(dimension * weight) / 100
	score := dimensions.EstimatedCodeChange*weights.CodeChangeWeight +
		dimensions.AffectedModules*weights.ModulesWeight +
		dimensions.BreakingChanges*weights.BreakingChangeWeight +
		dimensions.TestCoverageDifficulty*weights.TestingWeight

	return score / 100
}

// MockComplexityAnalyzer provides a simple mock analyzer for testing.
// Returns fixed dimension scores for testing purposes.
func MockComplexityAnalyzer(design *entity.Design, codebaseInfo *CodebaseInfo) (valueobject.ComplexityDimensions, error) {
	// Simple mock: returns moderate scores for all dimensions
	return valueobject.ComplexityDimensions{
		EstimatedCodeChange:    50,
		AffectedModules:        40,
		BreakingChanges:        30,
		TestCoverageDifficulty: 60,
	}, nil
}

// StaticComplexityAnalyzer creates an analyzer that returns predefined scores.
// Useful for testing with specific complexity scenarios.
func StaticComplexityAnalyzer(dimensions valueobject.ComplexityDimensions) ComplexityAnalyzer {
	return func(design *entity.Design, codebaseInfo *CodebaseInfo) (valueobject.ComplexityDimensions, error) {
		return dimensions, nil
	}
}
