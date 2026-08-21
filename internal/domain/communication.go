package domain

import (
	"strings"
	"time"
)

type Reply struct {
	ID         string
	FeedbackID string
	ActorID    string
	Content    string
	Public     bool
	CreatedAt  time.Time
}

func (r Reply) Validate() error {
	if strings.TrimSpace(r.ID) == "" || strings.TrimSpace(r.FeedbackID) == "" {
		return ValidationError{Field: "reply", Message: "reply and feedback ids are required"}
	}
	length := len([]rune(strings.TrimSpace(r.Content)))
	if length < 2 || length > 2000 {
		return ValidationError{Field: "content", Message: "reply must contain 2 to 2000 characters"}
	}
	if r.CreatedAt.IsZero() {
		return ValidationError{Field: "created_at", Message: "reply time is required"}
	}
	return nil
}

type Supplement struct {
	ID          string
	FeedbackID  string
	SubmitterID string
	Content     string
	CreatedAt   time.Time
}

func (s Supplement) Validate() error {
	if strings.TrimSpace(s.ID) == "" || strings.TrimSpace(s.FeedbackID) == "" {
		return ValidationError{Field: "supplement", Message: "supplement identity is incomplete"}
	}
	if len([]rune(strings.TrimSpace(s.Content))) < 2 {
		return ValidationError{Field: "content", Message: "supplement content is required"}
	}
	return nil
}

type Satisfaction struct {
	FeedbackID string
	Score      int
	Comment    string
	CreatedAt  time.Time
}

func (s Satisfaction) Validate(status FeedbackStatus) error {
	if status != StatusPendingConfirm && status != StatusClosed {
		return ValidationError{Field: "status", Message: "feedback is not ready for satisfaction confirmation"}
	}
	if s.Score < 1 || s.Score > 5 {
		return ValidationError{Field: "score", Message: "score must be between 1 and 5"}
	}
	if len([]rune(s.Comment)) > 500 {
		return ValidationError{Field: "comment", Message: "comment is too long"}
	}
	return nil
}
