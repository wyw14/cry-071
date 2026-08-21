package domain

import (
	"strings"
	"time"
)

type PublicArea struct {
	ID       string
	Name     string
	District string
	Active   bool
	Created  time.Time
}

func (a PublicArea) Validate() error {
	if strings.TrimSpace(a.ID) == "" {
		return ValidationError{Field: "area_id", Message: "area is required"}
	}
	if len([]rune(strings.TrimSpace(a.Name))) < 2 {
		return ValidationError{Field: "area_name", Message: "area name is too short"}
	}
	if strings.TrimSpace(a.District) == "" {
		return ValidationError{Field: "district", Message: "district is required"}
	}
	return nil
}

type FacilityCategory struct {
	ID          string
	Name        string
	Description string
	Active      bool
}

func (c FacilityCategory) Validate() error {
	if strings.TrimSpace(c.ID) == "" || strings.TrimSpace(c.Name) == "" {
		return ValidationError{Field: "facility_category", Message: "category id and name are required"}
	}
	return nil
}

type FeedbackSubject struct {
	Code              string
	Name              string
	DefaultPriority   Priority
	DefaultAssigneeID string
}

func (s FeedbackSubject) Validate() error {
	if strings.TrimSpace(s.Code) == "" || strings.TrimSpace(s.Name) == "" {
		return ValidationError{Field: "subject", Message: "subject code and name are required"}
	}
	if !s.DefaultPriority.Valid() {
		return ValidationError{Field: "default_priority", Message: "priority is invalid"}
	}
	return nil
}
