package valueobject

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/ryuyb/litchi/internal/pkg/errors"
)

// ClarityDimensions represents the clarity evaluation dimensions for requirements.
// Each dimension is scored 0-maxScore, and the total is calculated with weights.
//
// Dimensions and weights:
// - Completeness: 30% weight (max 30 points)
// - Clarity (明确性): 25% weight (max 25 points)
// - Consistency: 20% weight (max 20 points)
// - Feasibility: 15% weight (max 15 points)
// - Testability: 10% weight (max 10 points)
type ClarityDimensions struct {
	Completeness DimensionDetail `json:"completeness"` // Max 30, weight 30%
	Clarity      DimensionDetail `json:"clarity"`      // Max 25, weight 25% (明确性)
	Consistency  DimensionDetail `json:"consistency"`  // Max 20, weight 20%
	Feasibility  DimensionDetail `json:"feasibility"`  // Max 15, weight 15%
	Testability  DimensionDetail `json:"testability"`  // Max 10, weight 10%
	totalScore   int             // Calculated total score (0-100)
}

// DimensionDetail represents a single dimension's score and detailed checks.
type DimensionDetail struct {
	Score    int              `json:"score"`    // Current score
	MaxScore int              `json:"maxScore"` // Maximum possible score
	Checks   map[string]Check `json:"checks"`   // Individual check items
}

// Check represents a single check item within a dimension.
type Check struct {
	Score  int    `json:"score"`            // Score for this check
	Passed bool   `json:"passed"`           // Whether the check passed
	Detail string `json:"detail,omitempty"` // Optional detail/reason
}

// Dimension constants for max scores
const (
	CompletenessMaxScore = 30
	ClarityMaxScore      = 25
	ConsistencyMaxScore  = 20
	FeasibilityMaxScore  = 15
	TestabilityMaxScore  = 10
)

// ClarityGrade constants
const (
	GradeHighClarity   = "高清晰度" // 80-100
	GradeMediumClarity = "中清晰度" // 60-79
	GradeLowClarity    = "低清晰度" // 40-59
	GradeNotClear      = "不清晰"  // 0-39
)

// NewClarityDimensions creates a new ClarityDimensions with all dimension scores.
func NewClarityDimensions(
	completeness, clarity, consistency, feasibility, testability int,
) (ClarityDimensions, error) {
	cd := ClarityDimensions{
		Completeness: NewDimensionDetail(completeness, CompletenessMaxScore),
		Clarity:      NewDimensionDetail(clarity, ClarityMaxScore),
		Consistency:  NewDimensionDetail(consistency, ConsistencyMaxScore),
		Feasibility:  NewDimensionDetail(feasibility, FeasibilityMaxScore),
		Testability:  NewDimensionDetail(testability, TestabilityMaxScore),
	}

	// Validate all scores
	if err := cd.Validate(); err != nil {
		return ClarityDimensions{}, err
	}

	// Calculate total
	cd.totalScore = cd.CalculateTotal()
	return cd, nil
}

// NewDimensionDetail creates a DimensionDetail with the given score and max score.
func NewDimensionDetail(score, maxScore int) DimensionDetail {
	return DimensionDetail{
		Score:    score,
		MaxScore: maxScore,
		Checks:   make(map[string]Check),
	}
}

// Validate checks that all dimension scores are within valid ranges.
func (cd ClarityDimensions) Validate() error {
	dims := []struct {
		name     string
		score    int
		maxScore int
	}{
		{"Completeness", cd.Completeness.Score, CompletenessMaxScore},
		{"Clarity", cd.Clarity.Score, ClarityMaxScore},
		{"Consistency", cd.Consistency.Score, ConsistencyMaxScore},
		{"Feasibility", cd.Feasibility.Score, FeasibilityMaxScore},
		{"Testability", cd.Testability.Score, TestabilityMaxScore},
	}

	for _, dim := range dims {
		if dim.score < 0 || dim.score > dim.maxScore {
			return errors.New(errors.ErrInvalidClarityScore).WithDetail(
				fmt.Sprintf("%s score must be between 0 and %d, got: %d", dim.name, dim.maxScore, dim.score),
			)
		}
	}
	return nil
}

// CalculateTotal calculates the weighted total score (0-100).
// Since max scores already represent weights, sum them directly.
func (cd ClarityDimensions) CalculateTotal() int {
	return cd.Completeness.Score +
		cd.Clarity.Score +
		cd.Consistency.Score +
		cd.Feasibility.Score +
		cd.Testability.Score
}

// TotalScore returns the calculated total clarity score.
func (cd ClarityDimensions) TotalScore() int {
	return cd.totalScore
}

// Grade returns the clarity grade based on total score.
// - 高清晰度: 80-100 (auto proceed, no confirmation needed)
// - 中清晰度: 60-79 (auto proceed, but design needs confirmation)
// - 低清晰度: 40-59 (requires manual confirmation to proceed)
// - 不清晰: 0-39 (must continue clarification, cannot proceed)
func (cd ClarityDimensions) Grade() string {
	total := cd.TotalScore()
	switch {
	case total >= 80:
		return GradeHighClarity
	case total >= 60:
		return GradeMediumClarity
	case total >= 40:
		return GradeLowClarity
	default:
		return GradeNotClear
	}
}

// CanAutoProceed returns true if the score is high enough to auto proceed to design.
// >= 80: Auto proceed without any confirmation
// >= 60: Auto proceed but design needs confirmation
func (cd ClarityDimensions) CanAutoProceed(threshold int) bool {
	return cd.TotalScore() >= threshold
}

// NeedsManualConfirmation returns true if manual confirmation is needed before proceeding.
// 40-59: Need manual confirmation to proceed
// 0-39: Cannot proceed, must continue clarification
func (cd ClarityDimensions) NeedsManualConfirmation(threshold int) bool {
	total := cd.TotalScore()
	return total >= 40 && total < threshold
}

// MustContinueClarification returns true if the score is too low to proceed.
// Must continue clarification and answer more questions.
func (cd ClarityDimensions) MustContinueClarification() bool {
	return cd.TotalScore() < 40
}

// CanEnterDesign returns true if the session can enter design stage.
// >= threshold (default 60): Can enter design
func (cd ClarityDimensions) CanEnterDesign(threshold int) bool {
	return cd.TotalScore() >= threshold
}

// String returns the string representation.
func (cd ClarityDimensions) String() string {
	return fmt.Sprintf("Total: %d (%s)", cd.TotalScore(), cd.Grade())
}

// DisplayName returns the user-friendly display name with grade.
func (cd ClarityDimensions) DisplayName() string {
	return fmt.Sprintf("清晰度: %d分 (%s)", cd.TotalScore(), cd.Grade())
}

// SetCheck sets a check item for a specific dimension.
func (cd *ClarityDimensions) SetCheck(dimension, checkName string, score int, passed bool, detail string) {
	var dimDetail *DimensionDetail
	switch dimension {
	case "completeness":
		dimDetail = &cd.Completeness
	case "clarity":
		dimDetail = &cd.Clarity
	case "consistency":
		dimDetail = &cd.Consistency
	case "feasibility":
		dimDetail = &cd.Feasibility
	case "testability":
		dimDetail = &cd.Testability
	default:
		return
	}

	dimDetail.Checks[checkName] = Check{
		Score:  score,
		Passed: passed,
		Detail: detail,
	}
}

// RecalculateFromChecks recalculates dimension scores from individual checks.
func (cd *ClarityDimensions) RecalculateFromChecks() {
	// Recalculate each dimension score from its checks
	cd.Completeness.Score = sumChecks(cd.Completeness.Checks)
	cd.Clarity.Score = sumChecks(cd.Clarity.Checks)
	cd.Consistency.Score = sumChecks(cd.Consistency.Checks)
	cd.Feasibility.Score = sumChecks(cd.Feasibility.Checks)
	cd.Testability.Score = sumChecks(cd.Testability.Checks)

	// Recalculate total
	cd.totalScore = cd.CalculateTotal()
}

// sumChecks sums the scores of all checks in a map.
func sumChecks(checks map[string]Check) int {
	total := 0
	for _, check := range checks {
		total += check.Score
	}
	return total
}

// GORM database serialization implementation

// Value implements driver.Valuer for database serialization (as JSON).
func (cd ClarityDimensions) Value() (driver.Value, error) {
	return json.Marshal(cd)
}

// Scan implements sql.Scanner for database deserialization (from JSON).
func (cd *ClarityDimensions) Scan(value any) error {
	if value == nil {
		return errors.New(errors.ErrInvalidClarityScore).WithDetail("clarity dimensions cannot be null")
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New(errors.ErrInvalidClarityScore).WithDetail(
			fmt.Sprintf("cannot scan clarity dimensions from type: %T", value),
		)
	}

	if err := json.Unmarshal(bytes, cd); err != nil {
		return errors.New(errors.ErrInvalidClarityScore).WithDetail(
			fmt.Sprintf("cannot unmarshal clarity dimensions: %v", err),
		)
	}

	// Validate after scanning
	if err := cd.Validate(); err != nil {
		return err
	}

	// Calculate total after scanning
	cd.totalScore = cd.CalculateTotal()
	return nil
}

// MarshalJSON implements json.Marshaler for JSON serialization.
func (cd ClarityDimensions) MarshalJSON() ([]byte, error) {
	// Create a proxy struct for JSON serialization with totalScore included
	type Alias struct {
		Completeness   DimensionDetail `json:"completeness"`
		Clarity        DimensionDetail `json:"clarity"`
		Consistency    DimensionDetail `json:"consistency"`
		Feasibility    DimensionDetail `json:"feasibility"`
		Testability    DimensionDetail `json:"testability"`
		TotalScore     int             `json:"totalScore"`
		Grade          string          `json:"grade"`
		CanAutoProceed bool            `json:"canAutoProceed"`
	}

	alias := Alias{
		Completeness:   cd.Completeness,
		Clarity:        cd.Clarity,
		Consistency:    cd.Consistency,
		Feasibility:    cd.Feasibility,
		Testability:    cd.Testability,
		TotalScore:     cd.TotalScore(),
		Grade:          cd.Grade(),
		CanAutoProceed: cd.CanAutoProceed(60), // Default threshold
	}

	return json.Marshal(alias)
}

// UnmarshalJSON implements json.Unmarshaler for JSON deserialization.
func (cd *ClarityDimensions) UnmarshalJSON(data []byte) error {
	// Use proxy struct to extract fields
	type Alias struct {
		Completeness DimensionDetail `json:"completeness"`
		Clarity      DimensionDetail `json:"clarity"`
		Consistency  DimensionDetail `json:"consistency"`
		Feasibility  DimensionDetail `json:"feasibility"`
		Testability  DimensionDetail `json:"testability"`
		TotalScore   int             `json:"totalScore"` // Ignored, recalculated
	}

	var alias Alias
	if err := json.Unmarshal(data, &alias); err != nil {
		return errors.New(errors.ErrInvalidClarityScore).WithDetail(
			fmt.Sprintf("cannot unmarshal clarity dimensions: %v", err),
		)
	}

	cd.Completeness = alias.Completeness
	cd.Clarity = alias.Clarity
	cd.Consistency = alias.Consistency
	cd.Feasibility = alias.Feasibility
	cd.Testability = alias.Testability

	// Validate after unmarshaling
	if err := cd.Validate(); err != nil {
		return err
	}

	// Recalculate total (ignore JSON totalScore, compute from dimensions)
	cd.totalScore = cd.CalculateTotal()
	return nil
}

// DimensionDetail MarshalJSON for proper maxScore serialization
func (dd DimensionDetail) MarshalJSON() ([]byte, error) {
	type Alias struct {
		Score    int              `json:"score"`
		MaxScore int              `json:"maxScore"`
		Checks   map[string]Check `json:"checks"`
	}
	return json.Marshal(Alias(dd))
}

// DimensionDetail UnmarshalJSON
func (dd *DimensionDetail) UnmarshalJSON(data []byte) error {
	type Alias struct {
		Score    int              `json:"score"`
		MaxScore int              `json:"maxScore"`
		Checks   map[string]Check `json:"checks"`
	}
	var alias Alias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	dd.Score = alias.Score
	dd.MaxScore = alias.MaxScore
	dd.Checks = alias.Checks
	return nil
}
