package valueobject

import (
	"database/sql/driver"
	"fmt"

	"github.com/ryuyb/litchi/internal/pkg/errors"
)

// ComplexityScore represents the complexity score (0-100).
// Used to determine if a design requires manual confirmation.
// Threshold: score >= 70 requires confirmation (configurable).
//
// Evaluation dimensions:
// - EstimatedCodeChange: 30% weight
// - AffectedModules: 25% weight
// - BreakingChanges: 25% weight
// - TestCoverageDifficulty: 20% weight
type ComplexityScore struct {
	value int // 0-100
}

// Complexity dimensions with their scores and weights
type ComplexityDimensions struct {
	EstimatedCodeChange    int `json:"estimatedCodeChange"`    // 0-100, weight 30%
	AffectedModules        int `json:"affectedModules"`        // 0-100, weight 25%
	BreakingChanges        int `json:"breakingChanges"`        // 0-100, weight 25%
	TestCoverageDifficulty int `json:"testCoverageDifficulty"` // 0-100, weight 20%
}

// NewComplexityScore creates a new ComplexityScore with the given value (0-100).
func NewComplexityScore(value int) (ComplexityScore, error) {
	if value < 0 || value > 100 {
		return ComplexityScore{}, errors.New(errors.ErrInvalidComplexityScore).WithDetail(
			fmt.Sprintf("complexity score must be between 0 and 100, got: %d", value),
		)
	}
	return ComplexityScore{value: value}, nil
}

// NewComplexityScoreFromDimensions creates a ComplexityScore from dimension scores.
// The final score is calculated using weighted average.
func NewComplexityScoreFromDimensions(dimensions ComplexityDimensions) (ComplexityScore, error) {
	// Validate each dimension score is in valid range
	if err := validateDimensionScores(dimensions); err != nil {
		return ComplexityScore{}, err
	}

	// Calculate weighted score
	score := calculateWeightedScore(dimensions)
	return ComplexityScore{value: score}, nil
}

// validateDimensionScores ensures all dimension scores are within 0-100 range.
func validateDimensionScores(dimensions ComplexityDimensions) error {
	dims := []struct {
		name  string
		value int
	}{
		{"EstimatedCodeChange", dimensions.EstimatedCodeChange},
		{"AffectedModules", dimensions.AffectedModules},
		{"BreakingChanges", dimensions.BreakingChanges},
		{"TestCoverageDifficulty", dimensions.TestCoverageDifficulty},
	}

	for _, dim := range dims {
		if dim.value < 0 || dim.value > 100 {
			return errors.New(errors.ErrInvalidComplexityScore).WithDetail(
				fmt.Sprintf("%s score must be between 0 and 100, got: %d", dim.name, dim.value),
			)
		}
	}
	return nil
}

// calculateWeightedScore calculates the weighted average of dimension scores.
// Weight distribution: 30% + 25% + 25% + 20% = 100%
func calculateWeightedScore(dimensions ComplexityDimensions) int {
	score := dimensions.EstimatedCodeChange*30 +
		dimensions.AffectedModules*25 +
		dimensions.BreakingChanges*25 +
		dimensions.TestCoverageDifficulty*20
	return score / 100
}

// Value returns the complexity score value (0-100).
func (c ComplexityScore) Value() int {
	return c.value
}

// IsHigh returns true if the score exceeds the threshold (default 70).
// High complexity requires manual confirmation before proceeding.
func (c ComplexityScore) IsHigh(threshold int) bool {
	return c.value >= threshold
}

// IsLow returns true if the score is below the threshold.
// Low complexity can proceed without manual confirmation.
func (c ComplexityScore) IsLow(threshold int) bool {
	return c.value < threshold
}

// Grade returns the complexity grade based on the score.
// - Low: 0-39
// - Medium: 40-69
// - High: 70-89
// - VeryHigh: 90-100
func (c ComplexityScore) Grade() string {
	switch {
	case c.value >= 90:
		return "极高复杂度"
	case c.value >= 70:
		return "高复杂度"
	case c.value >= 40:
		return "中复杂度"
	default:
		return "低复杂度"
	}
}

// RequiresConfirmation returns true if the design requires manual confirmation.
// Uses the default threshold of 70 if not specified.
func (c ComplexityScore) RequiresConfirmation(threshold int) bool {
	return c.IsHigh(threshold)
}

// String returns the string representation of the complexity score.
func (c ComplexityScore) String() string {
	return fmt.Sprintf("%d", c.value)
}

// DisplayName returns the user-friendly display name with grade.
func (c ComplexityScore) DisplayName() string {
	return fmt.Sprintf("%d (%s)", c.value, c.Grade())
}

// GORM database serialization implementation

// GormValue implements driver.Valuer for database serialization.
// Note: Using GormValue to avoid conflict with Value() method.
func (c ComplexityScore) GormValue() (driver.Value, error) {
	return c.value, nil
}

// Scan implements sql.Scanner for database deserialization.
func (c *ComplexityScore) Scan(value any) error {
	if value == nil {
		return errors.New(errors.ErrInvalidComplexityScore).WithDetail("complexity score cannot be null")
	}

	var intValue int
	switch v := value.(type) {
	case int:
		intValue = v
	case int64:
		intValue = int(v)
	case int32:
		intValue = int(v)
	case float64:
		intValue = int(v)
	default:
		return errors.New(errors.ErrInvalidComplexityScore).WithDetail(
			fmt.Sprintf("cannot scan complexity score from type: %T", value),
		)
	}

	if intValue < 0 || intValue > 100 {
		return errors.New(errors.ErrInvalidComplexityScore).WithDetail(
			fmt.Sprintf("complexity score must be between 0 and 100, got: %d", intValue),
		)
	}

	c.value = intValue
	return nil
}

// MarshalJSON implements json.Marshaler for JSON serialization.
func (c ComplexityScore) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`%d`, c.value)), nil
}

// UnmarshalJSON implements json.Unmarshaler for JSON deserialization.
func (c *ComplexityScore) UnmarshalJSON(data []byte) error {
	// Parse integer from JSON
	var intValue int
	if err := parseJSONInt(data, &intValue); err != nil {
		return err
	}

	if intValue < 0 || intValue > 100 {
		return errors.New(errors.ErrInvalidComplexityScore).WithDetail(
			fmt.Sprintf("complexity score must be between 0 and 100, got: %d", intValue),
		)
	}

	c.value = intValue
	return nil
}

// parseJSONInt helper function to parse integer from JSON bytes.
func parseJSONInt(data []byte, result *int) error {
	str := string(data)

	// Handle null
	if str == "null" {
		return errors.New(errors.ErrInvalidComplexityScore).WithDetail("complexity score cannot be null")
	}

	// Parse the integer
	var parsed int
	_, err := fmt.Sscanf(str, "%d", &parsed)
	if err != nil {
		return errors.New(errors.ErrInvalidComplexityScore).WithDetail(
			fmt.Sprintf("cannot parse complexity score: %s", str),
		)
	}

	*result = parsed
	return nil
}
