package valueobject

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ryuyb/litchi/internal/pkg/errors"
)

func TestNewClarityDimensions(t *testing.T) {
	tests := []struct {
		name       string
		comp, clar, cons, feas, test int
		expected   int
		hasError   bool
	}{
		{"all_zero", 0, 0, 0, 0, 0, 0, false},
		{"all_max", 30, 25, 20, 15, 10, 100, false},
		{"valid_mixed", 24, 20, 18, 12, 8, 82, false},
		{"invalid_completeness_high", 31, 20, 15, 10, 5, 0, true},
		{"invalid_clarity_high", 25, 26, 15, 10, 5, 0, true},
		{"invalid_negative", -1, 20, 15, 10, 5, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cd, err := NewClarityDimensions(tt.comp, tt.clar, tt.cons, tt.feas, tt.test)
			if tt.hasError {
				if err == nil {
					t.Errorf("NewClarityDimensions(%s) expected error", tt.name)
				}
				if !errors.Is(err, errors.ErrInvalidClarityScore) {
					t.Errorf("NewClarityDimensions(%s) error should be ErrInvalidClarityScore", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("NewClarityDimensions(%s) unexpected error: %v", tt.name, err)
				}
				if cd.TotalScore() != tt.expected {
					t.Errorf("NewClarityDimensions(%s).TotalScore() = %d, expected %d", tt.name, cd.TotalScore(), tt.expected)
				}
			}
		})
	}
}

func TestClarityDimensionsCalculateTotal(t *testing.T) {
	// The total is the sum of all dimension scores
	cd, _ := NewClarityDimensions(24, 20, 18, 12, 8)
	expected := 24 + 20 + 18 + 12 + 8 // = 82
	if cd.CalculateTotal() != expected {
		t.Errorf("CalculateTotal() = %d, expected %d", cd.CalculateTotal(), expected)
	}
}

func TestClarityDimensionsGrade(t *testing.T) {
	tests := []struct {
		total    int
		expected string
	}{
		{85, GradeHighClarity},
		{80, GradeHighClarity},
		{75, GradeMediumClarity},
		{60, GradeMediumClarity},
		{55, GradeLowClarity},
		{40, GradeLowClarity},
		{35, GradeNotClear},
		{0, GradeNotClear},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("total_%d", tt.total), func(t *testing.T) {
			// Create dimensions that sum to the total
			cd := createTestClarityDimensions(tt.total)
			if cd.Grade() != tt.expected {
				t.Errorf("Grade() = %s, expected %s", cd.Grade(), tt.expected)
			}
		})
	}
}

func TestClarityDimensionsCanAutoProceed(t *testing.T) {
	threshold := 60

	tests := []struct {
		total    int
		expected bool
	}{
		{80, true},
		{70, true},
		{60, true},
		{59, false},
		{40, false},
		{30, false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("total_%d", tt.total), func(t *testing.T) {
			cd := createTestClarityDimensions(tt.total)
			if cd.CanAutoProceed(threshold) != tt.expected {
				t.Errorf("CanAutoProceed(%d) = %v, expected %v", tt.total, cd.CanAutoProceed(threshold), tt.expected)
			}
		})
	}
}

func TestClarityDimensionsNeedsManualConfirmation(t *testing.T) {
	threshold := 60

	tests := []struct {
		total    int
		expected bool
	}{
		{80, false}, // Can auto proceed
		{60, false}, // Can auto proceed
		{55, true},  // Needs manual confirmation (40-59)
		{45, true},  // Needs manual confirmation
		{35, false}, // Must continue clarification (0-39)
		{20, false}, // Must continue clarification
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("total_%d", tt.total), func(t *testing.T) {
			cd := createTestClarityDimensions(tt.total)
			if cd.NeedsManualConfirmation(threshold) != tt.expected {
				t.Errorf("NeedsManualConfirmation(%d) = %v, expected %v", tt.total, cd.NeedsManualConfirmation(threshold), tt.expected)
			}
		})
	}
}

func TestClarityDimensionsMustContinueClarification(t *testing.T) {
	tests := []struct {
		total    int
		expected bool
	}{
		{80, false},
		{60, false},
		{50, false},
		{39, true},
		{30, true},
		{0, true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("total_%d", tt.total), func(t *testing.T) {
			cd := createTestClarityDimensions(tt.total)
			if cd.MustContinueClarification() != tt.expected {
				t.Errorf("MustContinueClarification(%d) = %v, expected %v", tt.total, cd.MustContinueClarification(), tt.expected)
			}
		})
	}
}

func TestClarityDimensionsCanEnterDesign(t *testing.T) {
	threshold := 60

	tests := []struct {
		total    int
		expected bool
	}{
		{80, true},
		{60, true},
		{59, false},
		{40, false},
		{30, false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("total_%d", tt.total), func(t *testing.T) {
			cd := createTestClarityDimensions(tt.total)
			if cd.CanEnterDesign(threshold) != tt.expected {
				t.Errorf("CanEnterDesign(%d) = %v, expected %v", tt.total, cd.CanEnterDesign(threshold), tt.expected)
			}
		})
	}
}

func TestClarityDimensionsString(t *testing.T) {
	cd, _ := NewClarityDimensions(24, 20, 18, 12, 8)
	expected := "Total: 82 (高清晰度)"
	if cd.String() != expected {
		t.Errorf("String() = %s, expected %s", cd.String(), expected)
	}
}

func TestClarityDimensionsDisplayName(t *testing.T) {
	cd, _ := NewClarityDimensions(24, 20, 18, 12, 8)
	expected := "清晰度: 82分 (高清晰度)"
	if cd.DisplayName() != expected {
		t.Errorf("DisplayName() = %s, expected %s", cd.DisplayName(), expected)
	}
}

func TestClarityDimensionsSetCheck(t *testing.T) {
	cd, _ := NewClarityDimensions(0, 0, 0, 0, 0)

	// Set check items
	cd.SetCheck("completeness", "functionalGoal", 8, true, "功能目标明确")
	cd.SetCheck("clarity", "noAmbiguousWords", 8, true, "无模糊词汇")

	// Verify check items
	if cd.Completeness.Checks["functionalGoal"].Score != 8 {
		t.Errorf("Completeness.Checks['functionalGoal'].Score = %d, expected 8", cd.Completeness.Checks["functionalGoal"].Score)
	}
	if !cd.Completeness.Checks["functionalGoal"].Passed {
		t.Errorf("Completeness.Checks['functionalGoal'].Passed should be true")
	}
	if cd.Completeness.Checks["functionalGoal"].Detail != "功能目标明确" {
		t.Errorf("Completeness.Checks['functionalGoal'].Detail = %s, expected '功能目标明确'", cd.Completeness.Checks["functionalGoal"].Detail)
	}
}

func TestClarityDimensionsRecalculateFromChecks(t *testing.T) {
	cd, _ := NewClarityDimensions(0, 0, 0, 0, 0)

	// Set check items for completeness
	cd.SetCheck("completeness", "functionalGoal", 8, true, "")
	cd.SetCheck("completeness", "inputOutput", 6, true, "")
	cd.SetCheck("completeness", "techConstraints", 4, false, "")
	cd.SetCheck("completeness", "boundaryConditions", 3, false, "")
	cd.SetCheck("completeness", "dependencies", 5, true, "")

	cd.RecalculateFromChecks()

	// Completeness score should be sum of checks: 8+6+4+3+5 = 26
	if cd.Completeness.Score != 26 {
		t.Errorf("Completeness.Score after recalculate = %d, expected 26", cd.Completeness.Score)
	}

	// Total score should be updated
	if cd.TotalScore() != 26 {
		t.Errorf("TotalScore() after recalculate = %d, expected 26", cd.TotalScore())
	}
}

func TestClarityDimensionsJSONSerialization(t *testing.T) {
	cd, _ := NewClarityDimensions(24, 20, 18, 12, 8)

	data, err := json.Marshal(cd)
	if err != nil {
		t.Errorf("Marshal unexpected error: %v", err)
	}

	// Verify JSON structure
	var result map[string]any
	json.Unmarshal(data, &result)

	// Check totalScore is included
	if result["totalScore"] != float64(82) {
		t.Errorf("JSON totalScore = %v, expected 82", result["totalScore"])
	}

	// Check grade is included
	if result["grade"] != GradeHighClarity {
		t.Errorf("JSON grade = %v, expected %s", result["grade"], GradeHighClarity)
	}

	// Check dimensions
	if result["completeness"] == nil {
		t.Errorf("JSON should include completeness dimension")
	}
}

func TestClarityDimensionsJSONDeserialization(t *testing.T) {
	jsonStr := `{
		"completeness": {"score": 24, "maxScore": 30, "checks": {}},
		"clarity": {"score": 20, "maxScore": 25, "checks": {}},
		"consistency": {"score": 18, "maxScore": 20, "checks": {}},
		"feasibility": {"score": 12, "maxScore": 15, "checks": {}},
		"testability": {"score": 8, "maxScore": 10, "checks": {}}
	}`

	var cd ClarityDimensions
	err := json.Unmarshal([]byte(jsonStr), &cd)
	if err != nil {
		t.Errorf("Unmarshal unexpected error: %v", err)
	}

	if cd.TotalScore() != 82 {
		t.Errorf("Unmarshal TotalScore() = %d, expected 82", cd.TotalScore())
	}

	if cd.Completeness.Score != 24 {
		t.Errorf("Unmarshal Completeness.Score = %d, expected 24", cd.Completeness.Score)
	}
}

func TestClarityDimensionsJSONDeserializationInvalid(t *testing.T) {
	tests := []struct {
		name     string
		jsonStr  string
		hasError bool
	}{
		{"invalid_completeness_high", `{"completeness": {"score": 31, "maxScore": 30}, "clarity": {"score": 20}, "consistency": {"score": 18}, "feasibility": {"score": 12}, "testability": {"score": 8}}`, true},
		{"invalid_negative", `{"completeness": {"score": -1}, "clarity": {"score": 20}, "consistency": {"score": 18}, "feasibility": {"score": 12}, "testability": {"score": 8}}`, true},
		{"malformed_json", `{invalid}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cd ClarityDimensions
			err := json.Unmarshal([]byte(tt.jsonStr), &cd)
			if tt.hasError {
				if err == nil {
					t.Errorf("Unmarshal(%s) expected error", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("Unmarshal(%s) unexpected error: %v", tt.name, err)
				}
			}
		})
	}
}

func TestClarityDimensionsScan(t *testing.T) {
	jsonStr := `{"completeness": {"score": 24, "maxScore": 30, "checks": {}}, "clarity": {"score": 20, "maxScore": 25, "checks": {}}, "consistency": {"score": 18, "maxScore": 20, "checks": {}}, "feasibility": {"score": 12, "maxScore": 15, "checks": {}}, "testability": {"score": 8, "maxScore": 10, "checks": {}}}`

	var cd ClarityDimensions
	err := cd.Scan([]byte(jsonStr))
	if err != nil {
		t.Errorf("Scan unexpected error: %v", err)
	}

	if cd.TotalScore() != 82 {
		t.Errorf("Scan TotalScore() = %d, expected 82", cd.TotalScore())
	}

	// Test nil
	err = cd.Scan(nil)
	if err == nil {
		t.Errorf("Scan(nil) should return error")
	}

	// Test invalid type
	err = cd.Scan(123)
	if err == nil {
		t.Errorf("Scan(123) should return error")
	}
}

func TestClarityDimensionsValue(t *testing.T) {
	cd, _ := NewClarityDimensions(24, 20, 18, 12, 8)

	value, err := cd.Value()
	if err != nil {
		t.Errorf("Value unexpected error: %v", err)
	}

	// Value should be JSON bytes
	var result map[string]any
	json.Unmarshal(value.([]byte), &result)

	if result["totalScore"] != float64(82) {
		t.Errorf("Value totalScore = %v, expected 82", result["totalScore"])
	}
}

// Helper function to create test ClarityDimensions with a specific total score
func createTestClarityDimensions(total int) ClarityDimensions {
	// Distribute the total score across dimensions proportionally
	// Default ratios: 30:25:20:15:10
	if total == 100 {
		cd, _ := NewClarityDimensions(30, 25, 20, 15, 10)
		return cd
	}

	// Simplified: split evenly across dimensions
	comp := min(total/5, 30)
	clar := min(total/5, 25)
	cons := min(total/5, 20)
	feas := min(total/5, 15)
	test := min(total/5, 10)

	// Adjust if total doesn't match
	remaining := total - (comp + clar + cons + feas + test)
	comp = min(comp + remaining, 30)

	cd, _ := NewClarityDimensions(comp, clar, cons, feas, test)
	return cd
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}