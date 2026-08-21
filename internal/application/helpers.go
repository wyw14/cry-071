package application

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wyw14/cry-071/internal/domain"
)

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	New(string) string
}

func newTimelineEvent(ids IDGenerator, feedback *domain.Feedback, kind domain.EventKind, actorID string, visibility domain.TimelineVisibility, summary string, details map[string]string, now time.Time) domain.TimelineEvent {
	return domain.TimelineEvent{
		ID: ids.New("evt"), FeedbackID: feedback.ID, Sequence: feedback.NextTimelineSequence(),
		Kind: kind, ActorID: actorID, Visibility: visibility, Summary: summary,
		Details: details, OccurredAt: now.UTC(),
	}
}

func requestHash(value any) string {
	payload, _ := json.Marshal(value)
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}

func newAudit(ids IDGenerator, actorID, action, resource, resourceID, requestID string, metadata map[string]string, now time.Time) domain.AuditEntry {
	return domain.AuditEntry{
		ID: ids.New("audit"), ActorID: actorID, Action: action, Resource: resource,
		ResourceID: resourceID, RequestID: requestID, Metadata: metadata, OccurredAt: now.UTC(),
	}
}

func acceptanceNumber(now time.Time, sequence string) string {
	return fmt.Sprintf("PSF-%s-%s", now.UTC().Format("20060102"), sequence)
}
