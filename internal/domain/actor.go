package domain

import "strings"

type Role string

const (
	RoleSubmitter Role = "submitter"
	RoleAgent     Role = "agent"
	RoleManager   Role = "manager"
	RoleAuditor   Role = "auditor"
)

type Actor struct {
	ID          string
	DisplayName string
	Role        Role
	AreaIDs     []string
}

func (a Actor) Validate() error {
	if strings.TrimSpace(a.ID) == "" {
		return ValidationError{Field: "actor_id", Message: "actor id is required"}
	}
	switch a.Role {
	case RoleSubmitter, RoleAgent, RoleManager, RoleAuditor:
		return nil
	default:
		return ValidationError{Field: "role", Message: "unknown role"}
	}
}

func (a Actor) CanManage(areaID string) bool {
	if a.Role == RoleManager {
		return true
	}
	if a.Role != RoleAgent {
		return false
	}
	for _, assigned := range a.AreaIDs {
		if assigned == areaID {
			return true
		}
	}
	return false
}

func (a Actor) CanViewInternal() bool {
	return a.Role == RoleAgent || a.Role == RoleManager || a.Role == RoleAuditor
}

func (a Actor) CanPublish() bool { return a.Role == RoleManager }

func (a Actor) CanAudit() bool { return a.Role == RoleAuditor || a.Role == RoleManager }
