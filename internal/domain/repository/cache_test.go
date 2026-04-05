package repository

import (
	"testing"

	"github.com/google/uuid"
)

func TestExecutionContextCache_Validate_Valid(t *testing.T) {
	cache := &ExecutionContextCache{
		SessionID:    uuid.New(),
		CurrentStage: "execution",
		Status:       "active",
	}

	if err := cache.Validate(); err != nil {
		t.Errorf("Expected no error for valid cache, got: %v", err)
	}
}

func TestExecutionContextCache_Validate_MissingSessionID(t *testing.T) {
	cache := &ExecutionContextCache{
		SessionID:    uuid.Nil,
		CurrentStage: "execution",
		Status:       "active",
	}

	if err := cache.Validate(); err == nil {
		t.Error("Expected error for missing session ID")
	}
}

func TestExecutionContextCache_Validate_MissingCurrentStage(t *testing.T) {
	cache := &ExecutionContextCache{
		SessionID:    uuid.New(),
		CurrentStage: "",
		Status:       "active",
	}

	if err := cache.Validate(); err == nil {
		t.Error("Expected error for missing current stage")
	}
}

func TestExecutionContextCache_Validate_MissingStatus(t *testing.T) {
	cache := &ExecutionContextCache{
		SessionID:    uuid.New(),
		CurrentStage: "execution",
		Status:       "",
	}

	if err := cache.Validate(); err == nil {
		t.Error("Expected error for missing status")
	}
}