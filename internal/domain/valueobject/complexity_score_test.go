package valueobject

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ryuyb/litchi/internal/pkg/errors"
)

func TestNewComplexityScore(t *testing.T) {
	tests := []struct {
		value    int
		hasError bool
	}{
		{0, false},
		{50, false},
		{100, false},
		{-1, true},
		{101, true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("value_%d", tt.value), func(t *testing.T) {
			score, err := NewComplexityScore(tt.value)
			if tt.hasError {
				if err == nil {
					t.Errorf("NewComplexityScore(%d) expected error, got none", tt.value)
				}
				if !errors.Is(err, errors.ErrInvalidComplexityScore) {
					t.Errorf("NewComplexityScore(%d) error should be ErrInvalidComplexityScore", tt.value)
				}
			} else {
				if err != nil {
					t.Errorf("NewComplexityScore(%d) unexpected error: %v", tt.value, err)
				}
				if score.Value() != tt.value {
					t.Errorf("NewComplexityScore(%d).Value() = %d, expected %d", tt.value, score.Value(), tt.value)
				}
			}
		})
	}
}

func TestNewComplexityScoreFromDimensions(t *testing.T) {
	// Valid dimensions
	tests := []struct {
		name     string
		dims     ComplexityDimensions
		expected int
		hasError bool
	}{
		{
			name: "all_zero",
			dims: ComplexityDimensions{
				EstimatedCodeChange:    0,
				AffectedModules:        0,
				BreakingChanges:        0,
				TestCoverageDifficulty: 0,
			},
			expected: 0,
			hasError: false,
		},
		{
			name: "all_100",
			dims: ComplexityDimensions{
				EstimatedCodeChange:    100,
				AffectedModules:        100,
				BreakingChanges:        100,
				TestCoverageDifficulty: 100,
			},
			expected: 100,
			hasError: false,
		},
		{
			name: "mixed_values",
			dims: ComplexityDimensions{
				EstimatedCodeChange:    80,  // 80 * 30% = 24
				AffectedModules:        60,  // 60 * 25% = 15
				BreakingChanges:        40,  // 40 * 25% = 10
				TestCoverageDifficulty: 50,  // 50 * 20% = 10
			},
			expected: 59, // 24 + 15 + 10 + 10 = 59
			hasError: false,
		},
		{
			name: "invalid_negative",
			dims: ComplexityDimensions{
				EstimatedCodeChange:    -1,
				AffectedModules:        50,
				BreakingChanges:        50,
				TestCoverageDifficulty: 50,
			},
			hasError: true,
		},
		{
			name: "invalid_over_100",
			dims: ComplexityDimensions{
				EstimatedCodeChange:    101,
				AffectedModules:        50,
				BreakingChanges:        50,
				TestCoverageDifficulty: 50,
			},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := NewComplexityScoreFromDimensions(tt.dims)
			if tt.hasError {
				if err == nil {
					t.Errorf("NewComplexityScoreFromDimensions(%s) expected error", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("NewComplexityScoreFromDimensions(%s) unexpected error: %v", tt.name, err)
				}
				if score.Value() != tt.expected {
					t.Errorf("NewComplexityScoreFromDimensions(%s).Value() = %d, expected %d", tt.name, score.Value(), tt.expected)
				}
			}
		})
	}
}

func TestComplexityScoreIsHigh(t *testing.T) {
	threshold := 70

	tests := []struct {
		value    int
		expected bool
	}{
		{70, true},
		{80, true},
		{100, true},
		{69, false},
		{50, false},
		{0, false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("value_%d", tt.value), func(t *testing.T) {
			score, _ := NewComplexityScore(tt.value)
			if score.IsHigh(threshold) != tt.expected {
				t.Errorf("ComplexityScore(%d).IsHigh(%d) = %v, expected %v", tt.value, threshold, score.IsHigh(threshold), tt.expected)
			}
		})
	}
}

func TestComplexityScoreGrade(t *testing.T) {
	tests := []struct {
		value    int
		expected string
	}{
		{0, "低复杂度"},
		{39, "低复杂度"},
		{40, "中复杂度"},
		{69, "中复杂度"},
		{70, "高复杂度"},
		{89, "高复杂度"},
		{90, "极高复杂度"},
		{100, "极高复杂度"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("value_%d", tt.value), func(t *testing.T) {
			score, _ := NewComplexityScore(tt.value)
			if score.Grade() != tt.expected {
				t.Errorf("ComplexityScore(%d).Grade() = %s, expected %s", tt.value, score.Grade(), tt.expected)
			}
		})
	}
}

func TestComplexityScoreRequiresConfirmation(t *testing.T) {
	threshold := 70

	tests := []struct {
		value    int
		expected bool
	}{
		{70, true},  // At threshold, requires confirmation
		{80, true},  // Above threshold
		{69, false}, // Below threshold
		{50, false}, // Below threshold
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("value_%d", tt.value), func(t *testing.T) {
			score, _ := NewComplexityScore(tt.value)
			if score.RequiresConfirmation(threshold) != tt.expected {
				t.Errorf("ComplexityScore(%d).RequiresConfirmation(%d) = %v, expected %v", tt.value, threshold, score.RequiresConfirmation(threshold), tt.expected)
			}
		})
	}
}

func TestComplexityScoreString(t *testing.T) {
	score, _ := NewComplexityScore(75)
	if score.String() != "75" {
		t.Errorf("ComplexityScore(75).String() = %s, expected '75'", score.String())
	}
}

func TestComplexityScoreDisplayName(t *testing.T) {
	tests := []struct {
		value    int
		expected string
	}{
		{30, "30 (低复杂度)"},
		{50, "50 (中复杂度)"},
		{75, "75 (高复杂度)"},
		{95, "95 (极高复杂度)"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("value_%d", tt.value), func(t *testing.T) {
			score, _ := NewComplexityScore(tt.value)
			if score.DisplayName() != tt.expected {
				t.Errorf("ComplexityScore(%d).DisplayName() = %s, expected %s", tt.value, score.DisplayName(), tt.expected)
			}
		})
	}
}

func TestComplexityScoreJSONSerialization(t *testing.T) {
	score, _ := NewComplexityScore(75)
	data, err := json.Marshal(score)
	if err != nil {
		t.Errorf("Marshal(75) unexpected error: %v", err)
	}
	if string(data) != "75" {
		t.Errorf("Marshal(75) = %s, expected '75'", data)
	}
}

func TestComplexityScoreJSONDeserialization(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		hasError bool
	}{
		{"0", 0, false},
		{"50", 50, false},
		{"100", 100, false},
		{"-1", 0, true},
		{"101", 0, true},
		{"null", 0, true},
		{"\"invalid\"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var score ComplexityScore
			err := json.Unmarshal([]byte(tt.input), &score)

			if tt.hasError {
				if err == nil {
					t.Errorf("Unmarshal(%s) expected error", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unmarshal(%s) unexpected error: %v", tt.input, err)
				}
				if score.Value() != tt.expected {
					t.Errorf("Unmarshal(%s).Value() = %d, expected %d", tt.input, score.Value(), tt.expected)
				}
			}
		})
	}
}

func TestComplexityScoreScan(t *testing.T) {
	tests := []struct {
		input    any
		expected int
		hasError bool
	}{
		{0, 0, false},
		{50, 50, false},
		{int64(75), 75, false},
		{int32(60), 60, false},
		{float64(80), 80, false},
		{-1, 0, true},
		{101, 0, true},
		{nil, 0, true},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v_%d", tt.input, tt.expected), func(t *testing.T) {
			var score ComplexityScore
			err := score.Scan(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("Scan(%v) expected error", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Scan(%v) unexpected error: %v", tt.input, err)
				}
				if score.Value() != tt.expected {
					t.Errorf("Scan(%v).Value() = %d, expected %d", tt.input, score.Value(), tt.expected)
				}
			}
		})
	}
}