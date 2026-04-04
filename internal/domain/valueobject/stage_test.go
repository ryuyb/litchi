package valueobject

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ryuyb/litchi/internal/pkg/errors"
)

func TestAllStages(t *testing.T) {
	stages := AllStages()
	expected := []Stage{
		StageClarification,
		StageDesign,
		StageTaskBreakdown,
		StageExecution,
		StagePullRequest,
		StageCompleted,
	}

	if len(stages) != len(expected) {
		t.Errorf("AllStages() returned %d stages, expected %d", len(stages), len(expected))
	}

	for i, stage := range stages {
		if stage != expected[i] {
			t.Errorf("AllStages()[%d] = %s, expected %s", i, stage, expected[i])
		}
	}
}

func TestStageString(t *testing.T) {
	tests := []struct {
		stage    Stage
		expected string
	}{
		{StageClarification, "clarification"},
		{StageDesign, "design"},
		{StageTaskBreakdown, "task_breakdown"},
		{StageExecution, "execution"},
		{StagePullRequest, "pull_request"},
		{StageCompleted, "completed"},
	}

	for _, tt := range tests {
		t.Run(string(tt.stage), func(t *testing.T) {
			// Test pointer receiver method
			stage := tt.stage // Create variable to take address
			if got := (&stage).String(); got != tt.expected {
				t.Errorf("(&Stage).String() = %s, expected %s", got, tt.expected)
			}
		})
	}

	// Test nil safety
	t.Run("nil", func(t *testing.T) {
		var nilStage *Stage
		if got := nilStage.String(); got != "" {
			t.Errorf("nilStage.String() = %s, expected empty string", got)
		}
	})
}

func TestParse(t *testing.T) {
	tests := []struct {
		input    string
		expected Stage
		hasError bool
	}{
		{"clarification", StageClarification, false},
		{"design", StageDesign, false},
		{"task_breakdown", StageTaskBreakdown, false},
		{"execution", StageExecution, false},
		{"pull_request", StagePullRequest, false},
		{"completed", StageCompleted, false},
		// Case insensitive and whitespace tolerant
		{"CLARIFICATION", StageClarification, false},
		{"  design  ", StageDesign, false},
		{"DESIGN", StageDesign, false},
		// Invalid inputs
		{"invalid", "", true},
		{"", "", true},
		{"unknown_stage", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := Parse(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("Parse(%s) expected error, got none", tt.input)
				}
				if !errors.Is(err, errors.ErrInvalidStage) {
					t.Errorf("Parse(%s) error should be ErrInvalidStage, got %v", tt.input, err)
				}
			} else {
				if err != nil {
					t.Errorf("Parse(%s) unexpected error: %v", tt.input, err)
				}
				if got != tt.expected {
					t.Errorf("Parse(%s) = %s, expected %s", tt.input, got, tt.expected)
				}
			}
		})
	}
}

func TestMustParse(t *testing.T) {
	// Valid input
	stage := MustParse("clarification")
	if stage != StageClarification {
		t.Errorf("MustParse(clarification) = %s, expected %s", stage, StageClarification)
	}

	// Invalid input should panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("MustParse(invalid) should have panicked")
		}
	}()
	MustParse("invalid")
}

func TestStageIsValid(t *testing.T) {
	// Test pointer receiver method
	validStages := AllStages()
	for _, stage := range validStages {
		if !(&stage).IsValid() {
			t.Errorf("(&Stage(%s)).IsValid() should be true", stage)
		}
		// Test helper function
		if !IsValidStage(stage) {
			t.Errorf("IsValidStage(%s) should be true", stage)
		}
	}

	invalidStages := []Stage{"invalid", "", "unknown"}
	for _, stage := range invalidStages {
		if (&stage).IsValid() {
			t.Errorf("(&Stage(%s)).IsValid() should be false", stage)
		}
		if IsValidStage(stage) {
			t.Errorf("IsValidStage(%s) should be false", stage)
		}
	}

	// Test nil safety
	t.Run("nil", func(t *testing.T) {
		var nilStage *Stage
		if nilStage.IsValid() {
			t.Errorf("nilStage.IsValid() should be false")
		}
	})
}

func TestStageOrder(t *testing.T) {
	tests := []struct {
		stage    Stage
		expected int
	}{
		{StageClarification, 0},
		{StageDesign, 1},
		{StageTaskBreakdown, 2},
		{StageExecution, 3},
		{StagePullRequest, 4},
		{StageCompleted, 5},
		{Stage("invalid"), -1},
	}

	for _, tt := range tests {
		t.Run(string(tt.stage), func(t *testing.T) {
			// Test pointer receiver method
			stage := tt.stage // Create variable to take address
			if got := (&stage).Order(); got != tt.expected {
				t.Errorf("(&Stage).Order() = %d, expected %d", got, tt.expected)
			}
			// Test helper function
			if got := StageOrder(tt.stage); got != tt.expected {
				t.Errorf("StageOrder(%s) = %d, expected %d", tt.stage, got, tt.expected)
			}
		})
	}

	// Test nil safety
	t.Run("nil", func(t *testing.T) {
		var nilStage *Stage
		if nilStage.Order() != -1 {
			t.Errorf("nilStage.Order() should be -1")
		}
	})
}

func TestStageIsTerminal(t *testing.T) {
	// Test pointer receiver method
	completed := StageCompleted
	if !(&completed).IsTerminal() {
		t.Errorf("(&StageCompleted).IsTerminal() should be true")
	}
	// Test helper function
	if !IsTerminalStage(StageCompleted) {
		t.Errorf("IsTerminalStage(StageCompleted) should be true")
	}

	nonTerminalStages := []Stage{
		StageClarification,
		StageDesign,
		StageTaskBreakdown,
		StageExecution,
		StagePullRequest,
	}

	for _, s := range nonTerminalStages {
		stage := s // Create variable to take address
		if (&stage).IsTerminal() {
			t.Errorf("(&Stage(%s)).IsTerminal() should be false", stage)
		}
		if IsTerminalStage(s) {
			t.Errorf("IsTerminalStage(%s) should be false", s)
		}
	}

	// Test nil safety
	t.Run("nil", func(t *testing.T) {
		var nilStage *Stage
		if nilStage.IsTerminal() {
			t.Errorf("nilStage.IsTerminal() should be false")
		}
	})
}

func TestStageIsFirst(t *testing.T) {
	// Test pointer receiver method
	clarification := StageClarification
	if !(&clarification).IsFirst() {
		t.Errorf("(&StageClarification).IsFirst() should be true")
	}
	// Test helper function
	if !IsFirstStage(StageClarification) {
		t.Errorf("IsFirstStage(StageClarification) should be true")
	}

	otherStages := []Stage{
		StageDesign,
		StageTaskBreakdown,
		StageExecution,
		StagePullRequest,
		StageCompleted,
	}

	for _, s := range otherStages {
		stage := s // Create variable to take address
		if (&stage).IsFirst() {
			t.Errorf("(&Stage(%s)).IsFirst() should be false", stage)
		}
		if IsFirstStage(s) {
			t.Errorf("IsFirstStage(%s) should be false", s)
		}
	}

	// Test nil safety
	t.Run("nil", func(t *testing.T) {
		var nilStage *Stage
		if nilStage.IsFirst() {
			t.Errorf("nilStage.IsFirst() should be false")
		}
	})
}

func TestStageNext(t *testing.T) {
	tests := []struct {
		stage    Stage
		expected Stage
	}{
		{StageClarification, StageDesign},
		{StageDesign, StageTaskBreakdown},
		{StageTaskBreakdown, StageExecution},
		{StageExecution, StagePullRequest},
		{StagePullRequest, StageCompleted},
		{StageCompleted, ""}, // Terminal stage has no next
		{Stage("invalid"), ""}, // Invalid stage has no next
	}

	for _, tt := range tests {
		t.Run(string(tt.stage), func(t *testing.T) {
			// Test pointer receiver method
			stage := tt.stage // Create variable to take address
			if got := (&stage).Next(); got != tt.expected {
				t.Errorf("(&Stage).Next() = %s, expected %s", got, tt.expected)
			}
			// Test helper function
			if got := NextStage(tt.stage); got != tt.expected {
				t.Errorf("NextStage(%s) = %s, expected %s", tt.stage, got, tt.expected)
			}
		})
	}

	// Test nil safety
	t.Run("nil", func(t *testing.T) {
		var nilStage *Stage
		if nilStage.Next() != "" {
			t.Errorf("nilStage.Next() should be empty")
		}
	})
}

func TestStagePrev(t *testing.T) {
	tests := []struct {
		stage    Stage
		expected Stage
	}{
		{StageClarification, ""}, // First stage has no prev
		{StageDesign, StageClarification},
		{StageTaskBreakdown, StageDesign},
		{StageExecution, StageTaskBreakdown},
		{StagePullRequest, StageExecution},
		{StageCompleted, StagePullRequest},
		{Stage("invalid"), ""}, // Invalid stage has no prev
	}

	for _, tt := range tests {
		t.Run(string(tt.stage), func(t *testing.T) {
			// Test pointer receiver method
			stage := tt.stage // Create variable to take address
			if got := (&stage).Prev(); got != tt.expected {
				t.Errorf("(&Stage).Prev() = %s, expected %s", got, tt.expected)
			}
			// Test helper function
			if got := PrevStage(tt.stage); got != tt.expected {
				t.Errorf("PrevStage(%s) = %s, expected %s", tt.stage, got, tt.expected)
			}
		})
	}

	// Test nil safety
	t.Run("nil", func(t *testing.T) {
		var nilStage *Stage
		if nilStage.Prev() != "" {
			t.Errorf("nilStage.Prev() should be empty")
		}
	})
}

func TestStageCanTransitionTo(t *testing.T) {
	// Valid forward transitions (must be sequential)
	validTransitions := []struct {
		from, to Stage
	}{
		{StageClarification, StageDesign},
		{StageDesign, StageTaskBreakdown},
		{StageTaskBreakdown, StageExecution},
		{StageExecution, StagePullRequest},
		{StagePullRequest, StageCompleted},
	}

	for _, tt := range validTransitions {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			// Test pointer receiver method
			if !(&tt.from).CanTransitionTo(tt.to) {
				t.Errorf("(&Stage).CanTransitionTo(%s -> %s) should be true", tt.from, tt.to)
			}
			// Test helper function
			if !CanTransition(tt.from, tt.to) {
				t.Errorf("CanTransition(%s -> %s) should be true", tt.from, tt.to)
			}
		})
	}

	// Invalid forward transitions (skip stages)
	invalidForwardTransitions := []struct {
		from, to Stage
	}{
		{StageClarification, StageTaskBreakdown}, // Skip Design
		{StageClarification, StageExecution},      // Skip Design + TaskBreakdown
		{StageDesign, StageExecution},             // Skip TaskBreakdown
		{StageCompleted, StageClarification},      // Terminal stage
	}

	for _, tt := range invalidForwardTransitions {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			if (&tt.from).CanTransitionTo(tt.to) {
				t.Errorf("(&Stage).CanTransitionTo(%s -> %s) should be false", tt.from, tt.to)
			}
			if CanTransition(tt.from, tt.to) {
				t.Errorf("CanTransition(%s -> %s) should be false", tt.from, tt.to)
			}
		})
	}

	// Test nil safety
	t.Run("nil", func(t *testing.T) {
		var nilStage *Stage
		if nilStage.CanTransitionTo(StageDesign) {
			t.Errorf("nilStage.CanTransitionTo() should be false")
		}
	})
}

func TestStageCanRollbackTo(t *testing.T) {
	// Valid rollback rules as per state-machine design
	validRollbacks := []struct {
		from, to Stage
	}{
		// Design -> Clarification
		{StageDesign, StageClarification},
		// TaskBreakdown -> Design, Clarification
		{StageTaskBreakdown, StageDesign},
		{StageTaskBreakdown, StageClarification},
		// Execution -> Design, Clarification
		{StageExecution, StageDesign},
		{StageExecution, StageClarification},
		// PullRequest -> Execution (R4), Design (R5), Clarification (R6)
		{StagePullRequest, StageExecution},
		{StagePullRequest, StageDesign},
		{StagePullRequest, StageClarification},
	}

	for _, tt := range validRollbacks {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			// Test pointer receiver method
			if !(&tt.from).CanRollbackTo(tt.to) {
				t.Errorf("(&Stage).CanRollbackTo(%s -> %s) should be true", tt.from, tt.to)
			}
			// Test helper function
			if !CanRollback(tt.from, tt.to) {
				t.Errorf("CanRollback(%s -> %s) should be true", tt.from, tt.to)
			}
		})
	}

	// Invalid rollback rules
	invalidRollbacks := []struct {
		from, to Stage
	}{
		// Clarification cannot rollback
		{StageClarification, StageClarification},
		// Completed cannot rollback
		{StageCompleted, StageExecution},
		{StageCompleted, StageDesign},
		// Cannot rollback to later stages
		{StageDesign, StageExecution},
		{StageExecution, StagePullRequest},
		// Cannot rollback to itself
		{StageDesign, StageDesign},
		{StageExecution, StageExecution},
	}

	for _, tt := range invalidRollbacks {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			if (&tt.from).CanRollbackTo(tt.to) {
				t.Errorf("(&Stage).CanRollbackTo(%s -> %s) should be false", tt.from, tt.to)
			}
			if CanRollback(tt.from, tt.to) {
				t.Errorf("CanRollback(%s -> %s) should be false", tt.from, tt.to)
			}
		})
	}

	// Test nil safety
	t.Run("nil", func(t *testing.T) {
		var nilStage *Stage
		if nilStage.CanRollbackTo(StageClarification) {
			t.Errorf("nilStage.CanRollbackTo() should be false")
		}
	})
}

func TestStageDisplayName(t *testing.T) {
	tests := []struct {
		stage    Stage
		expected string
	}{
		{StageClarification, "需求澄清中"},
		{StageDesign, "设计方案中"},
		{StageTaskBreakdown, "任务拆解中"},
		{StageExecution, "任务执行中"},
		{StagePullRequest, "创建 PR"},
		{StageCompleted, "已完成"},
		{Stage("invalid"), "未知阶段"},
	}

	for _, tt := range tests {
		t.Run(string(tt.stage), func(t *testing.T) {
			// Test pointer receiver method
			stage := tt.stage // Create variable to take address
			if got := (&stage).DisplayName(); got != tt.expected {
				t.Errorf("(&Stage).DisplayName() = %s, expected %s", got, tt.expected)
			}
			// Test helper function
			if got := StageDisplayName(tt.stage); got != tt.expected {
				t.Errorf("StageDisplayName(%s) = %s, expected %s", tt.stage, got, tt.expected)
			}
		})
	}

	// Test nil safety
	t.Run("nil", func(t *testing.T) {
		var nilStage *Stage
		if nilStage.DisplayName() != "未知阶段" {
			t.Errorf("nilStage.DisplayName() should be '未知阶段'")
		}
	})
}

func TestStageJSONSerialization(t *testing.T) {
	tests := []struct {
		stage    Stage
		expected string
	}{
		{StageClarification, `"clarification"`},
		{StageDesign, `"design"`},
		{StageTaskBreakdown, `"task_breakdown"`},
		{StageExecution, `"execution"`},
		{StagePullRequest, `"pull_request"`},
		{StageCompleted, `"completed"`},
	}

	for _, tt := range tests {
		t.Run(string(tt.stage), func(t *testing.T) {
			stage := tt.stage // Create variable to take address
			data, err := json.Marshal(&stage)
			if err != nil {
				t.Errorf("Marshal(&%s) unexpected error: %v", tt.stage, err)
			}
			if string(data) != tt.expected {
				t.Errorf("Marshal(&%s) = %s, expected %s", tt.stage, data, tt.expected)
			}
		})
	}

	// Invalid stage should not marshal
	invalidStage := Stage("invalid")
	_, err := json.Marshal(&invalidStage)
	if err == nil {
		t.Errorf("Marshal(&invalid) should return error")
	}
	if !errors.Is(err, errors.ErrInvalidStage) {
		t.Errorf("Marshal(&invalid) error should be ErrInvalidStage, got %v", err)
	}

	// Test nil behavior: nil Stage serializes to "null" (consistent with Go standard behavior)
	// Business validation for required Stage fields should be done at Application/Domain layer
	t.Run("nil", func(t *testing.T) {
		var nilStage *Stage
		data, err := json.Marshal(nilStage)
		if err != nil {
			t.Errorf("Marshal(nil) unexpected error: %v", err)
		}
		if string(data) != "null" {
			t.Errorf("Marshal(nil) = %s, expected 'null'", data)
		}
	})
}

// Test that nil Stage in nested struct doesn't block serialization
func TestStageNilInNestedStruct(t *testing.T) {
	type FilterRequest struct {
		Stage *Stage `json:"stage,omitempty"`
		Name  string `json:"name"`
	}

	// nil Stage should not cause serialization to fail
	req := FilterRequest{Stage: nil, Name: "test"}
	data, err := json.Marshal(req)
	if err != nil {
		t.Errorf("Marshal(FilterRequest with nil Stage) unexpected error: %v", err)
	}

	// With omitempty, nil Stage should be omitted from output
	expected := `{"name":"test"}`
	if string(data) != expected {
		t.Errorf("Marshal(FilterRequest) = %s, expected %s", data, expected)
	}

	// Without omitempty, nil Stage should serialize to null
	type RequiredRequest struct {
		Stage *Stage `json:"stage"`
		Name  string `json:"name"`
	}

	req2 := RequiredRequest{Stage: nil, Name: "test"}
	data2, err := json.Marshal(req2)
	if err != nil {
		t.Errorf("Marshal(RequiredRequest with nil Stage) unexpected error: %v", err)
	}

	expected2 := `{"stage":null,"name":"test"}`
	if string(data2) != expected2 {
		t.Errorf("Marshal(RequiredRequest) = %s, expected %s", data2, expected2)
	}
}

func TestStageJSONDeserialization(t *testing.T) {
	tests := []struct {
		input    string
		expected Stage
		hasError bool
	}{
		{`"clarification"`, StageClarification, false},
		{`"design"`, StageDesign, false},
		{`"task_breakdown"`, StageTaskBreakdown, false},
		{`"execution"`, StageExecution, false},
		{`"pull_request"`, StagePullRequest, false},
		{`"completed"`, StageCompleted, false},
		{`"invalid"`, "", true},
		{`""`, "", true},
		{`null`, "", true}, // JSON null should be rejected with clear error
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var stage Stage
			err := json.Unmarshal([]byte(tt.input), &stage)

			if tt.hasError {
				if err == nil {
					t.Errorf("Unmarshal(%s) expected error, got none", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unmarshal(%s) unexpected error: %v", tt.input, err)
				}
				if stage != tt.expected {
					t.Errorf("Unmarshal(%s) = %s, expected %s", tt.input, stage, tt.expected)
				}
			}
		})
	}
}

func TestStageValue(t *testing.T) {
	// Valid stages
	for _, stage := range AllStages() {
		value, err := (&stage).Value()
		if err != nil {
			t.Errorf("Value(&%s) unexpected error: %v", stage, err)
		}
		if value != string(stage) {
			t.Errorf("Value(&%s) = %v, expected %s", stage, value, stage)
		}
	}

	// Invalid stage
	invalidStage := Stage("invalid")
	_, err := (&invalidStage).Value()
	if err == nil {
		t.Errorf("Value(&invalid) should return error")
	}

	// Test nil safety
	t.Run("nil", func(t *testing.T) {
		var nilStage *Stage
		_, err := nilStage.Value()
		if err == nil {
			t.Errorf("nilStage.Value() should return error")
		}
	})
}

func TestStageScan(t *testing.T) {
	tests := []struct {
		input    any
		expected Stage
		hasError bool
	}{
		{"clarification", StageClarification, false},
		{"design", StageDesign, false},
		{[]byte("execution"), StageExecution, false},
		{"invalid", "", true},
		{[]byte("invalid"), "", true},
		{nil, "", true},
		{123, "", true}, // Invalid type
	}

	for _, tt := range tests {
		t.Run(fmtInput(tt.input), func(t *testing.T) {
			var stage Stage
			err := (&stage).Scan(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("Scan(%v) expected error, got none", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Scan(%v) unexpected error: %v", tt.input, err)
				}
				if stage != tt.expected {
					t.Errorf("Scan(%v) = %s, expected %s", tt.input, stage, tt.expected)
				}
			}
		})
	}
}

func fmtInput(input any) string {
	if input == nil {
		return "nil"
	}
	return fmt.Sprintf("%v", input)
}