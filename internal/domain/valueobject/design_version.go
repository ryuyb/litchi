package valueobject

import "time"

// DesignVersion represents a single version of the design document.
// Used by Design entity for version management.
type DesignVersion struct {
	Version   int       `json:"version"`   // Version number (1, 2, 3, ...)
	Content   string    `json:"content"`   // Design document content
	Reason    string    `json:"reason"`    // Reason for this version (initial, rollback, update)
	CreatedAt time.Time `json:"createdAt"` // When this version was created
}

// NewDesignVersion creates a new design version.
func NewDesignVersion(version int, content, reason string) DesignVersion {
	return DesignVersion{
		Version:   version,
		Content:   content,
		Reason:    reason,
		CreatedAt: time.Now(),
	}
}

// IsInitial returns true if this is the initial design version.
func (dv DesignVersion) IsInitial() bool {
	return dv.Version == 1
}