package domain

import (
	"strings"
	"time"
)

type MergeGroup struct {
	ID              string
	PrimaryID       string
	MemberIDs       []string
	Reason          string
	CreatedBy       string
	CreatedAt       time.Time
	AnnouncementID  string
	ResolutionState string
}

func NewMergeGroup(id, primaryID string, memberIDs []string, reason, actorID string, now time.Time) (*MergeGroup, error) {
	id = strings.TrimSpace(id)
	primaryID = strings.TrimSpace(primaryID)
	reason = strings.TrimSpace(reason)
	if id == "" || primaryID == "" {
		return nil, ValidationError{Field: "merge", Message: "merge group and primary feedback are required"}
	}
	if len(memberIDs) == 0 {
		return nil, ValidationError{Field: "member_ids", Message: "at least one feedback must be merged"}
	}
	if len([]rune(reason)) < 5 {
		return nil, ValidationError{Field: "reason", Message: "merge reason is too short"}
	}
	seen := map[string]struct{}{primaryID: {}}
	members := make([]string, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		memberID = strings.TrimSpace(memberID)
		if memberID == "" {
			return nil, ValidationError{Field: "member_ids", Message: "member id is empty"}
		}
		if _, exists := seen[memberID]; exists {
			return nil, ValidationError{Field: "member_ids", Message: "merge group contains duplicate feedback"}
		}
		seen[memberID] = struct{}{}
		members = append(members, memberID)
	}
	return &MergeGroup{
		ID: id, PrimaryID: primaryID, MemberIDs: members, Reason: reason,
		CreatedBy: actorID, CreatedAt: now.UTC(), ResolutionState: "open",
	}, nil
}

func (g *MergeGroup) LinkAnnouncement(announcementID string) error {
	announcementID = strings.TrimSpace(announcementID)
	if announcementID == "" {
		return ValidationError{Field: "announcement_id", Message: "announcement is required"}
	}
	if g.AnnouncementID != "" && g.AnnouncementID != announcementID {
		return ConflictError{Resource: "merge announcement", Key: g.ID}
	}
	g.AnnouncementID = announcementID
	return nil
}

func (g *MergeGroup) Resolve() {
	g.ResolutionState = "resolved"
}
