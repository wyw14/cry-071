package domain

import (
	"strings"
	"time"
)

type AuditEntry struct {
	ID         string
	ActorID    string
	Action     string
	Resource   string
	ResourceID string
	RequestID  string
	Metadata   map[string]string
	OccurredAt time.Time
}

func (e AuditEntry) Validate() error {
	if strings.TrimSpace(e.ID) == "" || strings.TrimSpace(e.Action) == "" {
		return ValidationError{Field: "audit", Message: "audit identity and action are required"}
	}
	if strings.TrimSpace(e.Resource) == "" || strings.TrimSpace(e.ResourceID) == "" {
		return ValidationError{Field: "resource", Message: "audited resource is required"}
	}
	if e.OccurredAt.IsZero() {
		return ValidationError{Field: "occurred_at", Message: "audit time is required"}
	}
	return nil
}

func (e AuditEntry) Clone() AuditEntry {
	e.Metadata = cloneStringMap(e.Metadata)
	return e
}
