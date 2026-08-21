package domain

import (
	"strings"
	"time"
)

type Announcement struct {
	ID          string
	Title       string
	Content     string
	AreaID      string
	FeedbackIDs []string
	Published   bool
	PublishedBy string
	PublishedAt *time.Time
	CreatedAt   time.Time
}

func NewAnnouncement(id, title, content, areaID string, feedbackIDs []string, now time.Time) (*Announcement, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(areaID) == "" {
		return nil, ValidationError{Field: "announcement", Message: "announcement and area are required"}
	}
	if len([]rune(strings.TrimSpace(title))) < 4 {
		return nil, ValidationError{Field: "title", Message: "announcement title is too short"}
	}
	if len([]rune(strings.TrimSpace(content))) < 10 {
		return nil, ValidationError{Field: "content", Message: "announcement content is too short"}
	}
	if len(feedbackIDs) == 0 {
		return nil, ValidationError{Field: "feedback_ids", Message: "announcement must link feedback"}
	}
	return &Announcement{
		ID: id, Title: strings.TrimSpace(title), Content: strings.TrimSpace(content),
		AreaID: areaID, FeedbackIDs: append([]string(nil), feedbackIDs...), CreatedAt: now.UTC(),
	}, nil
}

func (a *Announcement) Publish(actor Actor, now time.Time) error {
	if !actor.CanPublish() {
		return ErrForbidden
	}
	if a.Published {
		return ConflictError{Resource: "announcement", Key: a.ID}
	}
	publishedAt := now.UTC()
	a.Published = true
	a.PublishedBy = actor.ID
	a.PublishedAt = &publishedAt
	return nil
}

func (a Announcement) Clone() *Announcement {
	copyAnnouncement := a
	copyAnnouncement.FeedbackIDs = append([]string(nil), a.FeedbackIDs...)
	if a.PublishedAt != nil {
		published := *a.PublishedAt
		copyAnnouncement.PublishedAt = &published
	}
	return &copyAnnouncement
}
