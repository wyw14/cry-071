package domain

import (
	"strings"
	"time"
)

type EventKind string

const (
	EventSubmitted          EventKind = "submitted"
	EventStatusChanged      EventKind = "status_changed"
	EventReplyAdded         EventKind = "reply_added"
	EventInternalNoteAdded  EventKind = "internal_note_added"
	EventTransferred        EventKind = "transferred"
	EventMaterialAdded      EventKind = "material_added"
	EventMerged             EventKind = "merged"
	EventReopened           EventKind = "reopened"
	EventSatisfactionGiven  EventKind = "satisfaction_given"
	EventAnnouncementLinked EventKind = "announcement_linked"
)

type TimelineVisibility string

const (
	VisibilityPublic   TimelineVisibility = "public"
	VisibilityInternal TimelineVisibility = "internal"
)

type TimelineEvent struct {
	ID         string
	FeedbackID string
	Sequence   int64
	Kind       EventKind
	ActorID    string
	Visibility TimelineVisibility
	Summary    string
	Details    map[string]string
	OccurredAt time.Time
}

func (e TimelineEvent) Validate() error {
	if strings.TrimSpace(e.ID) == "" || strings.TrimSpace(e.FeedbackID) == "" {
		return ValidationError{Field: "timeline_event", Message: "event and feedback ids are required"}
	}
	if e.Sequence <= 0 {
		return ValidationError{Field: "sequence", Message: "event sequence must be positive"}
	}
	if e.Visibility != VisibilityPublic && e.Visibility != VisibilityInternal {
		return ValidationError{Field: "visibility", Message: "timeline visibility is invalid"}
	}
	if strings.TrimSpace(e.Summary) == "" {
		return ValidationError{Field: "summary", Message: "timeline summary is required"}
	}
	if e.OccurredAt.IsZero() {
		return ValidationError{Field: "occurred_at", Message: "timeline time is required"}
	}
	return nil
}

// Clone returns a deep copy of the event so that callers cannot mutate a saved
// timeline entry through a value handed out for display. The Details map is the
// only reference field and must be copied by value, not shared.
func (e TimelineEvent) Clone() TimelineEvent {
	clone := e
	clone.Details = cloneStringMap(e.Details)
	return clone
}

func PublicTimeline(events []TimelineEvent) []TimelineEvent {
	public := make([]TimelineEvent, 0, len(events))
	for _, event := range events {
		if event.Visibility != VisibilityPublic {
			continue
		}
		public = append(public, event.Clone())
	}
	return public
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
