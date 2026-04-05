package service

import (
	"testing"

	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
)

func TestComplexityEvaluator_EvaluateWithDefaultWeights(t *testing.T) {
	// Create evaluator with static analyzer
	dimensions := valueobject.ComplexityDimensions{
		EstimatedCodeChange:    60,
		AffectedModules:        50,
		BreakingChanges:        40,
		TestCoverageDifficulty: 30,
	}

	analyzer := StaticComplexityAnalyzer(dimensions)
	evaluator := NewDefaultComplexityEvaluator(analyzer)

	design := entity.NewDesign("Test design content")
	codebaseInfo := &CodebaseInfo{
		AffectedModules: []string{"auth", "user"},
		TechStack:       []string{"go", "postgres"},
	}

	score, dims, err := evaluator.EvaluateWithDefaultWeights(design, codebaseInfo)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// Expected: (60*30 + 50*25 + 40*25 + 30*20) / 100 = (1800 + 1250 + 1000 + 600) / 100 = 46.5 ≈ 46
	expectedScore := (60*30 + 50*25 + 40*25 + 30*20) / 100
	if score.Value() != expectedScore {
		t.Errorf("expected score %d, got %d", expectedScore, score.Value())
	}

	// Verify dimensions match input
	if dims.EstimatedCodeChange != 60 {
		t.Errorf("expected EstimatedCodeChange 60, got %d", dims.EstimatedCodeChange)
	}
}

func TestComplexityEvaluator_EvaluateWithCustomWeights(t *testing.T) {
	dimensions := valueobject.ComplexityDimensions{
		EstimatedCodeChange:    100,
		AffectedModules:        100,
		BreakingChanges:        100,
		TestCoverageDifficulty: 100,
	}

	analyzer := StaticComplexityAnalyzer(dimensions)
	evaluator := NewDefaultComplexityEvaluator(analyzer)

	// Custom weights: all equal
	weights := ComplexityWeights{
		CodeChangeWeight:     25,
		ModulesWeight:        25,
		BreakingChangeWeight: 25,
		TestingWeight:        25,
	}

	design := entity.NewDesign("Test design")

	score, _, err := evaluator.Evaluate(design, nil, &weights)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// With all dimensions at 100 and equal weights, score should be 100
	if score.Value() != 100 {
		t.Errorf("expected score 100, got %d", score.Value())
	}
}

func TestComplexityEvaluator_InvalidWeights(t *testing.T) {
	dimensions := valueobject.ComplexityDimensions{
		EstimatedCodeChange:    50,
		AffectedModules:        50,
		BreakingChanges:        50,
		TestCoverageDifficulty: 50,
	}

	analyzer := StaticComplexityAnalyzer(dimensions)
	evaluator := NewDefaultComplexityEvaluator(analyzer)

	// Weights that don't sum to 100
	weights := ComplexityWeights{
		CodeChangeWeight:     30,
		ModulesWeight:        30,
		BreakingChangeWeight: 30,
		TestingWeight:        30, // Sum = 120
	}

	design := entity.NewDesign("Test design")

	_, _, err := evaluator.Evaluate(design, nil, &weights)
	if err == nil {
		t.Error("expected error for invalid weights")
	}
}

func TestComplexityEvaluator_Threshold(t *testing.T) {
	analyzer := MockComplexityAnalyzer

	evaluator := NewDefaultComplexityEvaluator(analyzer)
	if evaluator.GetThreshold() != 70 {
		t.Errorf("expected default threshold 70, got %d", evaluator.GetThreshold())
	}

	// Set custom threshold
	err := evaluator.SetThreshold(80)
	if err != nil {
		t.Fatalf("SetThreshold failed: %v", err)
	}
	if evaluator.GetThreshold() != 80 {
		t.Errorf("expected threshold 80, got %d", evaluator.GetThreshold())
	}

	// Invalid threshold
	err = evaluator.SetThreshold(-1)
	if err == nil {
		t.Error("expected error for negative threshold")
	}

	err = evaluator.SetThreshold(101)
	if err == nil {
		t.Error("expected error for threshold > 100")
	}
}

func TestComplexityEvaluator_NewWithThreshold(t *testing.T) {
	analyzer := MockComplexityAnalyzer

	evaluator, err := NewComplexityEvaluatorWithThreshold(50, analyzer)
	if err != nil {
		t.Fatalf("NewComplexityEvaluatorWithThreshold failed: %v", err)
	}
	if evaluator.GetThreshold() != 50 {
		t.Errorf("expected threshold 50, got %d", evaluator.GetThreshold())
	}

	// Invalid threshold
	_, err = NewComplexityEvaluatorWithThreshold(150, analyzer)
	if err == nil {
		t.Error("expected error for invalid threshold")
	}
}

func TestComplexityEvaluator_NilDesign(t *testing.T) {
	analyzer := MockComplexityAnalyzer
	evaluator := NewDefaultComplexityEvaluator(analyzer)

	_, _, err := evaluator.Evaluate(nil, nil, nil)
	if err == nil {
		t.Error("expected error for nil design")
	}
}

func TestComplexityEvaluator_NoAnalyzer(t *testing.T) {
	evaluator := &DefaultComplexityEvaluator{
		threshold: 70,
		analyzer:  nil, // No analyzer configured
	}

	design := entity.NewDesign("Test")

	_, _, err := evaluator.Evaluate(design, nil, nil)
	if err == nil {
		t.Error("expected error when analyzer not configured")
	}
}

func TestComplexityWeights_Validate(t *testing.T) {
	// Valid weights
	valid := ComplexityWeights{
		CodeChangeWeight:     30,
		ModulesWeight:        25,
		BreakingChangeWeight: 25,
		TestingWeight:        20,
	}
	if !valid.Validate() {
		t.Error("valid weights should pass validation")
	}

	// Sum not 100
	invalidSum := ComplexityWeights{
		CodeChangeWeight:     30,
		ModulesWeight:        30,
		BreakingChangeWeight: 30,
		TestingWeight:        5, // Sum = 95
	}
	if invalidSum.Validate() {
		t.Error("weights with sum != 100 should fail validation")
	}

	// Negative weight
	negative := ComplexityWeights{
		CodeChangeWeight:     -10,
		ModulesWeight:        40,
		BreakingChangeWeight: 40,
		TestingWeight:        30,
	}
	if negative.Validate() {
		t.Error("negative weight should fail validation")
	}

	// Weight > 100
	overLimit := ComplexityWeights{
		CodeChangeWeight:     110,
		ModulesWeight:        0,
		BreakingChangeWeight: 0,
		TestingWeight:        0,
	}
	if overLimit.Validate() {
		t.Error("weight > 100 should fail validation")
	}
}

func TestDefaultComplexityWeights(t *testing.T) {
	weights := DefaultComplexityWeights()
	if !weights.Validate() {
		t.Error("default weights should be valid")
	}

	// Verify expected values
	if weights.CodeChangeWeight != 30 {
		t.Errorf("expected CodeChangeWeight 30, got %d", weights.CodeChangeWeight)
	}
	if weights.ModulesWeight != 25 {
		t.Errorf("expected ModulesWeight 25, got %d", weights.ModulesWeight)
	}
	if weights.BreakingChangeWeight != 25 {
		t.Errorf("expected BreakingChangeWeight 25, got %d", weights.BreakingChangeWeight)
	}
	if weights.TestingWeight != 20 {
		t.Errorf("expected TestingWeight 20, got %d", weights.TestingWeight)
	}
}

func TestMockComplexityAnalyzer(t *testing.T) {
	design := entity.NewDesign("Test")
	codebaseInfo := &CodebaseInfo{
		AffectedModules: []string{"user"},
	}

	dims, err := MockComplexityAnalyzer(design, codebaseInfo)
	if err != nil {
		t.Fatalf("MockComplexityAnalyzer failed: %v", err)
	}

	// Verify mock returns fixed values
	if dims.EstimatedCodeChange != 50 {
		t.Errorf("expected EstimatedCodeChange 50, got %d", dims.EstimatedCodeChange)
	}
	if dims.AffectedModules != 40 {
		t.Errorf("expected AffectedModules 40, got %d", dims.AffectedModules)
	}
	if dims.BreakingChanges != 30 {
		t.Errorf("expected BreakingChanges 30, got %d", dims.BreakingChanges)
	}
	if dims.TestCoverageDifficulty != 60 {
		t.Errorf("expected TestCoverageDifficulty 60, got %d", dims.TestCoverageDifficulty)
	}
}

func TestStaticComplexityAnalyzer(t *testing.T) {
	dimensions := valueobject.ComplexityDimensions{
		EstimatedCodeChange:    80,
		AffectedModules:        70,
		BreakingChanges:        60,
		TestCoverageDifficulty: 50,
	}

	analyzer := StaticComplexityAnalyzer(dimensions)
	design := entity.NewDesign("Test")

	resultDims, err := analyzer(design, nil)
	if err != nil {
		t.Fatalf("StaticComplexityAnalyzer failed: %v", err)
	}

	// Should return exact dimensions provided
	if resultDims.EstimatedCodeChange != 80 {
		t.Errorf("expected EstimatedCodeChange 80, got %d", resultDims.EstimatedCodeChange)
	}
	if resultDims.AffectedModules != 70 {
		t.Errorf("expected AffectedModules 70, got %d", resultDims.AffectedModules)
	}
}
