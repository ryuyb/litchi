package valueobject

import (
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/ryuyb/litchi/internal/pkg/errors"
)

// Stage represents the workflow stage enum.
// The core workflow stages are: Clarification → Design → TaskBreakdown → Execution → PullRequest → Completed
type Stage string

// Core workflow stages (6 stages as per T2.1.1 requirement)
const (
	StageClarification  Stage = "clarification"
	StageDesign         Stage = "design"
	StageTaskBreakdown  Stage = "task_breakdown"
	StageExecution      Stage = "execution"
	StagePullRequest    Stage = "pull_request"
	StageCompleted      Stage = "completed"
)

// AllStages returns all valid workflow stages in order.
func AllStages() []Stage {
	return []Stage{
		StageClarification,
		StageDesign,
		StageTaskBreakdown,
		StageExecution,
		StagePullRequest,
		StageCompleted,
	}
}

// Parse parses a string into a Stage.
// Returns an error if the string is not a valid stage.
func Parse(stageStr string) (Stage, error) {
	stage := Stage(strings.ToLower(strings.TrimSpace(stageStr)))
	for _, s := range AllStages() {
		if s == stage {
			return stage, nil
		}
	}
	return "", errors.New(errors.ErrInvalidStage).WithDetail(
		fmt.Sprintf("invalid stage: %s, valid stages are: %v", stageStr, AllStages()),
	)
}

// MustParse parses a string into a Stage, panics on invalid input.
// Use this only when you are certain the input is valid.
func MustParse(stageStr string) Stage {
	stage, err := Parse(stageStr)
	if err != nil {
		panic(err)
	}
	return stage
}

// IsValidStage checks if the given stage is a valid workflow stage.
// This is a helper function for convenient value-type usage.
func IsValidStage(s Stage) bool {
	return (&s).IsValid()
}

// StageOrder returns the ordinal position of the stage in the workflow (0-based).
// Returns -1 for invalid stages. This is a helper function for convenient value-type usage.
func StageOrder(s Stage) int {
	return (&s).Order()
}

// IsTerminalStage checks if the stage is a terminal stage (Completed).
// This is a helper function for convenient value-type usage.
func IsTerminalStage(s Stage) bool {
	return (&s).IsTerminal()
}

// IsFirstStage checks if the stage is the first stage (Clarification).
// This is a helper function for convenient value-type usage.
func IsFirstStage(s Stage) bool {
	return (&s).IsFirst()
}

// NextStage returns the next stage in the workflow.
// Returns empty Stage if current stage is terminal or invalid.
// This is a helper function for convenient value-type usage.
func NextStage(s Stage) Stage {
	return (&s).Next()
}

// PrevStage returns the previous stage in the workflow.
// Returns empty Stage if current stage is first or invalid.
// This is a helper function for convenient value-type usage.
func PrevStage(s Stage) Stage {
	return (&s).Prev()
}

// CanTransition checks if forward transition from source to target stage is allowed.
// Forward transitions must be sequential (one stage at a time).
// This is a helper function for convenient value-type usage.
func CanTransition(source, target Stage) bool {
	return (&source).CanTransitionTo(target)
}

// CanRollback checks if rollback from source to target stage is allowed.
// This is a helper function for convenient value-type usage.
func CanRollback(source, target Stage) bool {
	return (&source).CanRollbackTo(target)
}

// StageDisplayName returns the user-friendly display name for the stage.
// This is a helper function for convenient value-type usage.
func StageDisplayName(s Stage) string {
	return (&s).DisplayName()
}

// Methods with pointer receivers (consistent with Go best practices)

// String returns the string representation of the stage.
func (s *Stage) String() string {
	if s == nil {
		return ""
	}
	return string(*s)
}

// IsValid checks if the stage is a valid workflow stage.
func (s *Stage) IsValid() bool {
	if s == nil {
		return false
	}
	for _, stage := range AllStages() {
		if *s == stage {
			return true
		}
	}
	return false
}

// Order returns the ordinal position of the stage in the workflow (0-based).
// Returns -1 for invalid stages.
func (s *Stage) Order() int {
	if s == nil {
		return -1
	}
	for i, stage := range AllStages() {
		if *s == stage {
			return i
		}
	}
	return -1
}

// IsTerminal checks if the stage is a terminal stage (Completed).
func (s *Stage) IsTerminal() bool {
	if s == nil {
		return false
	}
	return *s == StageCompleted
}

// IsFirst checks if the stage is the first stage (Clarification).
func (s *Stage) IsFirst() bool {
	if s == nil {
		return false
	}
	return *s == StageClarification
}

// Next returns the next stage in the workflow.
// Returns empty Stage if current stage is terminal or invalid.
func (s *Stage) Next() Stage {
	if s == nil {
		return ""
	}
	order := s.Order()
	if order < 0 || order >= len(AllStages())-1 {
		return ""
	}
	return AllStages()[order+1]
}

// Prev returns the previous stage in the workflow.
// Returns empty Stage if current stage is first or invalid.
func (s *Stage) Prev() Stage {
	if s == nil {
		return ""
	}
	order := s.Order()
	if order <= 0 {
		return ""
	}
	return AllStages()[order-1]
}

// CanTransitionTo checks if forward transition to target stage is allowed.
// Forward transitions must be sequential (one stage at a time).
func (s *Stage) CanTransitionTo(target Stage) bool {
	if s == nil {
		return false
	}
	// Cannot transition from terminal or invalid stage
	if s.IsTerminal() || !s.IsValid() {
		return false
	}
	// Target must be the immediate next stage
	return s.Next() == target
}

// CanRollbackTo checks if rollback to target stage is allowed.
// Rollback rules as per state-machine design:
// - Execution can rollback to Design or Clarification
// - Design can rollback to Clarification
// - TaskBreakdown can rollback to Design or Clarification
// - PullRequest can rollback to Execution, Design, or Clarification
// - Clarification cannot rollback
// - Completed cannot rollback
func (s *Stage) CanRollbackTo(target Stage) bool {
	if s == nil {
		return false
	}
	if !s.IsValid() || !(&target).IsValid() {
		return false
	}

	// Clarification stage cannot rollback
	if *s == StageClarification {
		return false
	}

	// Completed stage cannot rollback
	if *s == StageCompleted {
		return false
	}

	// Rollback rules per stage
	switch *s {
	case StageDesign:
		return target == StageClarification
	case StageTaskBreakdown:
		return target == StageDesign || target == StageClarification
	case StageExecution:
		return target == StageDesign || target == StageClarification
	case StagePullRequest:
		// PR stage supports three-level rollback (R4, R5, R6)
		return target == StageExecution || target == StageDesign || target == StageClarification
	default:
		return false
	}
}

// DisplayName returns the user-friendly display name for the stage.
func (s *Stage) DisplayName() string {
	if s == nil {
		return "未知阶段"
	}
	switch *s {
	case StageClarification:
		return "需求澄清中"
	case StageDesign:
		return "设计方案中"
	case StageTaskBreakdown:
		return "任务拆解中"
	case StageExecution:
		return "任务执行中"
	case StagePullRequest:
		return "创建 PR"
	case StageCompleted:
		return "已完成"
	default:
		return "未知阶段"
	}
}

// GORM database serialization implementation

// Value implements driver.Valuer for database serialization.
func (s *Stage) Value() (driver.Value, error) {
	if s == nil {
		return nil, errors.New(errors.ErrInvalidStage).WithDetail("stage cannot be nil")
	}
	if !s.IsValid() {
		return nil, errors.New(errors.ErrInvalidStage).WithDetail(
			fmt.Sprintf("cannot serialize invalid stage: %s", *s),
		)
	}
	return s.String(), nil
}

// Scan implements sql.Scanner for database deserialization.
func (s *Stage) Scan(value any) error {
	if value == nil {
		return errors.New(errors.ErrInvalidStage).WithDetail("stage cannot be null")
	}

	var str string
	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return errors.New(errors.ErrInvalidStage).WithDetail(
			fmt.Sprintf("cannot scan stage from type: %T", value),
		)
	}

	stage, err := Parse(str)
	if err != nil {
		return err
	}

	*s = stage
	return nil
}

// MarshalJSON implements json.Marshaler for JSON serialization.
// nil Stage serializes to "null" (consistent with Go standard behavior).
// Business validation for required Stage fields should be done at the
// Application/Domain layer, not at serialization layer.
func (s *Stage) MarshalJSON() ([]byte, error) {
	// nil Stage serializes to "null" (consistent with Go standard behavior)
	if s == nil {
		return []byte("null"), nil
	}
	if !s.IsValid() {
		return nil, errors.New(errors.ErrInvalidStage).WithDetail(
			fmt.Sprintf("cannot marshal invalid stage: %s", *s),
		)
	}
	return []byte(fmt.Sprintf(`"%s"`, s.String())), nil
}

// UnmarshalJSON implements json.Unmarshaler for JSON deserialization.
func (s *Stage) UnmarshalJSON(data []byte) error {
	// Handle JSON null explicitly
	if string(data) == "null" {
		return errors.New(errors.ErrInvalidStage).WithDetail("stage cannot be null")
	}
	// Remove quotes
	str := strings.Trim(string(data), `"`)
	stage, err := Parse(str)
	if err != nil {
		return err
	}
	*s = stage
	return nil
}