package memory

import (
	"time"

	"github.com/wyw14/cry-071/internal/domain"
)

type tokenRecord struct {
	FeedbackID string
	ExpiresAt  time.Time
}

type idempotencyRecord struct {
	RequestHash string
	ResourceID  string
	ExpiresAt   time.Time
}

type state struct {
	feedbacks     map[string]*domain.Feedback
	acceptanceIDs map[string]string
	timeline      map[string][]domain.TimelineEvent
	replies       map[string][]domain.Reply
	supplements   map[string][]domain.Supplement
	satisfaction  map[string]domain.Satisfaction
	attachments   map[string]domain.Attachment
	attachmentIDs map[string][]string
	merges        map[string]*domain.MergeGroup
	announcements map[string]*domain.Announcement
	audits        []domain.AuditEntry
	tokens        map[string]tokenRecord
	idempotency   map[string]idempotencyRecord
	areas         map[string]domain.PublicArea
	categories    map[string]domain.FacilityCategory
	subjects      map[string]domain.FeedbackSubject
}

func newState() *state {
	return &state{
		feedbacks: map[string]*domain.Feedback{}, acceptanceIDs: map[string]string{},
		timeline: map[string][]domain.TimelineEvent{}, replies: map[string][]domain.Reply{},
		supplements: map[string][]domain.Supplement{}, satisfaction: map[string]domain.Satisfaction{},
		attachments: map[string]domain.Attachment{}, attachmentIDs: map[string][]string{},
		merges: map[string]*domain.MergeGroup{}, announcements: map[string]*domain.Announcement{},
		audits: []domain.AuditEntry{}, tokens: map[string]tokenRecord{}, idempotency: map[string]idempotencyRecord{}, areas: map[string]domain.PublicArea{},
		categories: map[string]domain.FacilityCategory{}, subjects: map[string]domain.FeedbackSubject{},
	}
}

func (s *state) clone() *state {
	copyState := newState()
	for id, feedback := range s.feedbacks {
		copyState.feedbacks[id] = feedback.Clone()
	}
	for number, id := range s.acceptanceIDs {
		copyState.acceptanceIDs[number] = id
	}
	for id, events := range s.timeline {
		copyState.timeline[id] = cloneEvents(events)
	}
	for id, replies := range s.replies {
		copyState.replies[id] = append([]domain.Reply(nil), replies...)
	}
	for id, supplements := range s.supplements {
		copyState.supplements[id] = append([]domain.Supplement(nil), supplements...)
	}
	for id, satisfaction := range s.satisfaction {
		copyState.satisfaction[id] = satisfaction
	}
	for id, attachment := range s.attachments {
		copyState.attachments[id] = attachment
	}
	for id, ids := range s.attachmentIDs {
		copyState.attachmentIDs[id] = append([]string(nil), ids...)
	}
	for id, merge := range s.merges {
		copied := *merge
		copied.MemberIDs = append([]string(nil), merge.MemberIDs...)
		copyState.merges[id] = &copied
	}
	for id, announcement := range s.announcements {
		copyState.announcements[id] = announcement.Clone()
	}
	copyState.audits = make([]domain.AuditEntry, len(s.audits))
	for index, audit := range s.audits {
		copyState.audits[index] = audit.Clone()
	}
	for digest, token := range s.tokens {
		copyState.tokens[digest] = token
	}
	for key, record := range s.idempotency {
		copyState.idempotency[key] = record
	}
	for id, area := range s.areas {
		copyState.areas[id] = area
	}
	for id, category := range s.categories {
		copyState.categories[id] = category
	}
	for id, subject := range s.subjects {
		copyState.subjects[id] = subject
	}
	return copyState
}

func cloneEvents(events []domain.TimelineEvent) []domain.TimelineEvent {
	cloned := make([]domain.TimelineEvent, len(events))
	for index, event := range events {
		cloned[index] = event
		if event.Details != nil {
			cloned[index].Details = make(map[string]string, len(event.Details))
			for key, value := range event.Details {
				cloned[index].Details[key] = value
			}
		}
	}
	return cloned
}
