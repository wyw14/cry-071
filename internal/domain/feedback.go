package domain

import (
	"fmt"
	"strings"
	"time"
)

type Feedback struct {
	ID                 string
	AcceptanceNumber   string
	AreaID             string
	FacilityCategoryID string
	SubjectCode        string
	Priority           Priority
	Title              string
	Description        string
	Location           string
	Anonymous          bool
	SubmitterID        string
	SubmitterName      string
	SubmitterContact   string
	Status             FeedbackStatus
	AssigneeID         string
	MergedIntoID       string
	RelatedIDs         []string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DueAt              time.Time
	ClosedAt           *time.Time
	Version            int64
	TimelineSequence   int64
}

type NewFeedback struct {
	ID                 string
	AcceptanceNumber   string
	AreaID             string
	FacilityCategoryID string
	SubjectCode        string
	Priority           Priority
	Title              string
	Description        string
	Location           string
	Anonymous          bool
	SubmitterID        string
	SubmitterName      string
	SubmitterContact   string
	CreatedAt          time.Time
}

func CreateFeedback(input NewFeedback) (*Feedback, error) {
	input.ID = strings.TrimSpace(input.ID)
	input.AcceptanceNumber = strings.TrimSpace(input.AcceptanceNumber)
	input.AreaID = strings.TrimSpace(input.AreaID)
	input.FacilityCategoryID = strings.TrimSpace(input.FacilityCategoryID)
	input.SubjectCode = strings.TrimSpace(input.SubjectCode)
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.Location = strings.TrimSpace(input.Location)
	if input.ID == "" {
		return nil, ValidationError{Field: "id", Message: "feedback id is required"}
	}
	if input.AcceptanceNumber == "" {
		return nil, ValidationError{Field: "acceptance_number", Message: "acceptance number is required"}
	}
	if input.AreaID == "" {
		return nil, ValidationError{Field: "area_id", Message: "area is required"}
	}
	if input.FacilityCategoryID == "" {
		return nil, ValidationError{Field: "facility_category_id", Message: "facility category is required"}
	}
	if input.SubjectCode == "" {
		return nil, ValidationError{Field: "subject_code", Message: "subject is required"}
	}
	if !input.Priority.Valid() {
		return nil, ValidationError{Field: "priority", Message: "priority is invalid"}
	}
	if titleLength := len([]rune(input.Title)); titleLength < 5 || titleLength > 120 {
		return nil, ValidationError{Field: "title", Message: "title must contain 5 to 120 characters"}
	}
	if descriptionLength := len([]rune(input.Description)); descriptionLength < 10 || descriptionLength > 4000 {
		return nil, ValidationError{Field: "description", Message: "description must contain 10 to 4000 characters"}
	}
	if input.Location == "" {
		return nil, ValidationError{Field: "location", Message: "location description is required"}
	}
	if input.CreatedAt.IsZero() {
		return nil, ValidationError{Field: "created_at", Message: "creation time is required"}
	}
	if !input.Anonymous && strings.TrimSpace(input.SubmitterName) == "" {
		return nil, ValidationError{Field: "submitter_name", Message: "submitter name is required"}
	}
	feedback := &Feedback{
		ID:                 input.ID,
		AcceptanceNumber:   input.AcceptanceNumber,
		AreaID:             input.AreaID,
		FacilityCategoryID: input.FacilityCategoryID,
		SubjectCode:        input.SubjectCode,
		Priority:           input.Priority,
		Title:              input.Title,
		Description:        input.Description,
		Location:           input.Location,
		Anonymous:          input.Anonymous,
		SubmitterID:        strings.TrimSpace(input.SubmitterID),
		SubmitterName:      strings.TrimSpace(input.SubmitterName),
		SubmitterContact:   strings.TrimSpace(input.SubmitterContact),
		Status:             StatusPendingAcceptance,
		CreatedAt:          input.CreatedAt.UTC(),
		UpdatedAt:          input.CreatedAt.UTC(),
		DueAt:              input.CreatedAt.UTC().Add(time.Duration(input.Priority.SLAHours()) * time.Hour),
		Version:            1,
	}
	return feedback, nil
}

func (f *Feedback) Validate() error {
	if f == nil {
		return ValidationError{Field: "feedback", Message: "feedback is required"}
	}
	if strings.TrimSpace(f.ID) == "" || strings.TrimSpace(f.AcceptanceNumber) == "" {
		return ValidationError{Field: "feedback", Message: "feedback identity is incomplete"}
	}
	if !f.Status.Valid() {
		return ValidationError{Field: "status", Message: "status is invalid"}
	}
	if !f.Priority.Valid() {
		return ValidationError{Field: "priority", Message: "priority is invalid"}
	}
	if f.Version < 1 {
		return ValidationError{Field: "version", Message: "version must be positive"}
	}
	if f.Status == StatusClosed && f.ClosedAt == nil {
		return ValidationError{Field: "closed_at", Message: "closed feedback needs closure time"}
	}
	if f.Status != StatusClosed && f.ClosedAt != nil {
		return ValidationError{Field: "closed_at", Message: "only closed feedback can have closure time"}
	}
	return nil
}

func (f *Feedback) Transition(to FeedbackStatus, actor Actor, reason string, now time.Time) error {
	if f == nil {
		return ErrNotFound
	}
	if err := actor.Validate(); err != nil {
		return err
	}
	if !actor.CanManage(f.AreaID) {
		return ErrForbidden
	}
	if !to.Valid() || !CanTransition(f.Status, to) {
		return StateError{From: f.Status, To: to}
	}
	reason = strings.TrimSpace(reason)
	if transitionNeedsReason(f.Status, to) && len([]rune(reason)) < 5 {
		return ValidationError{Field: "reason", Message: "a specific transition reason is required"}
	}
	if to == StatusInProgress && f.AssigneeID == "" {
		return ValidationError{Field: "assignee_id", Message: "feedback must be assigned before processing"}
	}
	if now.IsZero() || now.Before(f.CreatedAt) {
		return ValidationError{Field: "occurred_at", Message: "transition time is invalid"}
	}
	from := f.Status
	f.Status = to
	f.UpdatedAt = now.UTC()
	f.Version++
	if to == StatusClosed {
		closed := now.UTC()
		f.ClosedAt = &closed
	} else if from == StatusClosed {
		f.ClosedAt = nil
	}
	return nil
}

func transitionNeedsReason(from, to FeedbackStatus) bool {
	if to == StatusRejected || to == StatusNeedsInformation {
		return true
	}
	return from == StatusClosed || from == StatusRejected
}

func (f *Feedback) Assign(actor Actor, assigneeID string, now time.Time) error {
	if !actor.CanManage(f.AreaID) {
		return ErrForbidden
	}
	assigneeID = strings.TrimSpace(assigneeID)
	if assigneeID == "" {
		return ValidationError{Field: "assignee_id", Message: "assignee is required"}
	}
	if f.Status.Terminal() {
		return ValidationError{Field: "status", Message: "terminal feedback cannot be assigned"}
	}
	f.AssigneeID = assigneeID
	f.UpdatedAt = now.UTC()
	f.Version++
	return nil
}

func (f *Feedback) AddRelation(otherID string, now time.Time) error {
	otherID = strings.TrimSpace(otherID)
	if otherID == "" || otherID == f.ID {
		return ValidationError{Field: "related_id", Message: "related feedback is invalid"}
	}
	for _, existing := range f.RelatedIDs {
		if existing == otherID {
			return nil
		}
	}
	f.RelatedIDs = append(f.RelatedIDs, otherID)
	f.UpdatedAt = now.UTC()
	f.Version++
	return nil
}

func (f *Feedback) MarkMerged(targetID string, now time.Time) error {
	targetID = strings.TrimSpace(targetID)
	if targetID == "" || targetID == f.ID {
		return ValidationError{Field: "target_id", Message: "merge target is invalid"}
	}
	if f.MergedIntoID != "" {
		return ConflictError{Resource: "feedback merge", Key: f.ID}
	}
	if f.Status == StatusClosed {
		return ValidationError{Field: "status", Message: "closed feedback cannot be merged"}
	}
	f.MergedIntoID = targetID
	f.UpdatedAt = now.UTC()
	f.Version++
	return nil
}

func (f *Feedback) NextTimelineSequence() int64 {
	f.TimelineSequence++
	f.Version++
	return f.TimelineSequence
}

func (f Feedback) IsOverdue(now time.Time) bool {
	return !f.Status.Terminal() && now.After(f.DueAt)
}

func (f Feedback) PublicSubmitter() string {
	if f.Anonymous {
		return "匿名提交者"
	}
	return f.SubmitterName
}

func (f Feedback) String() string {
	return fmt.Sprintf("%s[%s] %s", f.AcceptanceNumber, f.Status, f.Title)
}

func (f Feedback) Clone() *Feedback {
	copyFeedback := f
	copyFeedback.RelatedIDs = append([]string(nil), f.RelatedIDs...)
	if f.ClosedAt != nil {
		closed := *f.ClosedAt
		copyFeedback.ClosedAt = &closed
	}
	return &copyFeedback
}
