package valueobject

import "time"

// Branch represents Git branch information.
type Branch struct {
	Name         string     `json:"name"`                   // Branch name
	IsDeprecated bool       `json:"isDeprecated"`           // Whether the branch is deprecated
	DeprecatedAt *time.Time `json:"deprecatedAt,omitempty"` // When deprecated (if applicable)
}

// NewBranch creates a new active branch.
func NewBranch(name string) Branch {
	return Branch{
		Name:         name,
		IsDeprecated: false,
	}
}

// Deprecate marks the branch as deprecated.
func (b *Branch) Deprecate(reason string) {
	now := time.Now()
	b.IsDeprecated = true
	b.DeprecatedAt = &now
}

// IsActive returns true if the branch is still active.
func (b Branch) IsActive() bool {
	return !b.IsDeprecated
}

// DeprecatedBranch represents a deprecated branch record for rollback operations.
type DeprecatedBranch struct {
	Name            string    `json:"name"`                    // Branch name
	DeprecatedAt    time.Time `json:"deprecatedAt"`            // When deprecated
	Reason          string    `json:"reason"`                  // Why deprecated (e.g., "rollback to design")
	PRNumber        *int      `json:"prNumber,omitempty"`      // Associated PR number (if any)
	RollbackToStage string    `json:"rollbackToStage,omitempty"` // Target stage for rollback
}

// NewDeprecatedBranch creates a new deprecated branch record.
func NewDeprecatedBranch(name, reason string, prNumber *int, rollbackToStage string) DeprecatedBranch {
	return DeprecatedBranch{
		Name:            name,
		DeprecatedAt:    time.Now(),
		Reason:          reason,
		PRNumber:        prNumber,
		RollbackToStage: rollbackToStage,
	}
}