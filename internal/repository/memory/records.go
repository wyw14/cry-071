package memory

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/wyw14/cry-071/internal/domain"
)

func (r *repositoryView) Append(ctx context.Context, event domain.TimelineEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := event.Validate(); err != nil {
		return err
	}
	events := r.state.timeline[event.FeedbackID]
	if len(events) > 0 && events[len(events)-1].Sequence >= event.Sequence {
		return domain.ConflictError{Resource: "timeline sequence", Key: event.FeedbackID}
	}
	r.state.timeline[event.FeedbackID] = append(events, event.Clone())
	return nil
}

func (r *repositoryView) ListTimeline(ctx context.Context, feedbackID string) ([]domain.TimelineEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, exists := r.state.feedbacks[feedbackID]; !exists {
		return nil, domain.ErrNotFound
	}
	return cloneEvents(r.state.timeline[feedbackID]), nil
}

func (r *repositoryView) CreateReply(ctx context.Context, reply domain.Reply) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := reply.Validate(); err != nil {
		return err
	}
	if _, exists := r.state.feedbacks[reply.FeedbackID]; !exists {
		return domain.ErrNotFound
	}
	r.state.replies[reply.FeedbackID] = append(r.state.replies[reply.FeedbackID], reply)
	return nil
}

func (r *repositoryView) ListReplies(ctx context.Context, feedbackID string) ([]domain.Reply, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, exists := r.state.feedbacks[feedbackID]; !exists {
		return nil, domain.ErrNotFound
	}
	return append([]domain.Reply(nil), r.state.replies[feedbackID]...), nil
}

func (r *repositoryView) CreateSupplement(ctx context.Context, supplement domain.Supplement) error {
	if err := supplement.Validate(); err != nil {
		return err
	}
	if _, exists := r.state.feedbacks[supplement.FeedbackID]; !exists {
		return domain.ErrNotFound
	}
	r.state.supplements[supplement.FeedbackID] = append(r.state.supplements[supplement.FeedbackID], supplement)
	return nil
}

func (r *repositoryView) SaveSatisfaction(ctx context.Context, satisfaction domain.Satisfaction) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, exists := r.state.satisfaction[satisfaction.FeedbackID]; exists {
		return domain.ConflictError{Resource: "satisfaction", Key: satisfaction.FeedbackID}
	}
	r.state.satisfaction[satisfaction.FeedbackID] = satisfaction
	return nil
}

func (r *repositoryView) GetSatisfaction(ctx context.Context, feedbackID string) (*domain.Satisfaction, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	result, exists := r.state.satisfaction[feedbackID]
	if !exists {
		return nil, domain.ErrNotFound
	}
	copyResult := result
	return &copyResult, nil
}

func (r *repositoryView) CreateAttachment(ctx context.Context, attachment domain.Attachment) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := attachment.Validate(); err != nil {
		return err
	}
	if _, exists := r.state.attachments[attachment.ID]; exists {
		return domain.ConflictError{Resource: "attachment", Key: attachment.ID}
	}
	r.state.attachments[attachment.ID] = attachment
	r.state.attachmentIDs[attachment.FeedbackID] = append(r.state.attachmentIDs[attachment.FeedbackID], attachment.ID)
	return nil
}

func (r *repositoryView) GetAttachment(ctx context.Context, id string) (*domain.Attachment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	attachment, exists := r.state.attachments[id]
	if !exists {
		return nil, domain.ErrNotFound
	}
	copyAttachment := attachment
	return &copyAttachment, nil
}

func (r *repositoryView) ListAttachments(ctx context.Context, feedbackID string) ([]domain.Attachment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	result := make([]domain.Attachment, 0, len(r.state.attachmentIDs[feedbackID]))
	for _, id := range r.state.attachmentIDs[feedbackID] {
		result = append(result, r.state.attachments[id])
	}
	return result, nil
}

func (r *repositoryView) CreateMerge(ctx context.Context, group *domain.MergeGroup) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, exists := r.state.merges[group.ID]; exists {
		return domain.ConflictError{Resource: "merge group", Key: group.ID}
	}
	copyGroup := *group
	copyGroup.MemberIDs = append([]string(nil), group.MemberIDs...)
	r.state.merges[group.ID] = &copyGroup
	return nil
}

func (r *repositoryView) GetMerge(ctx context.Context, id string) (*domain.MergeGroup, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	group, exists := r.state.merges[id]
	if !exists {
		return nil, domain.ErrNotFound
	}
	copyGroup := *group
	copyGroup.MemberIDs = append([]string(nil), group.MemberIDs...)
	return &copyGroup, nil
}

func (r *repositoryView) UpdateMerge(ctx context.Context, group *domain.MergeGroup) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, exists := r.state.merges[group.ID]; !exists {
		return domain.ErrNotFound
	}
	copyGroup := *group
	copyGroup.MemberIDs = append([]string(nil), group.MemberIDs...)
	r.state.merges[group.ID] = &copyGroup
	return nil
}

func (r *repositoryView) CreateAnnouncement(ctx context.Context, announcement *domain.Announcement) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, exists := r.state.announcements[announcement.ID]; exists {
		return domain.ConflictError{Resource: "announcement", Key: announcement.ID}
	}
	r.state.announcements[announcement.ID] = announcement.Clone()
	return nil
}

func (r *repositoryView) GetAnnouncement(ctx context.Context, id string) (*domain.Announcement, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	announcement, exists := r.state.announcements[id]
	if !exists {
		return nil, domain.ErrNotFound
	}
	return announcement.Clone(), nil
}

func (r *repositoryView) UpdateAnnouncement(ctx context.Context, announcement *domain.Announcement) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, exists := r.state.announcements[announcement.ID]; !exists {
		return domain.ErrNotFound
	}
	r.state.announcements[announcement.ID] = announcement.Clone()
	return nil
}

func (r *repositoryView) ListAnnouncements(ctx context.Context, areaID string, publishedOnly bool) ([]*domain.Announcement, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	result := make([]*domain.Announcement, 0)
	for _, announcement := range r.state.announcements {
		if areaID != "" && announcement.AreaID != areaID {
			continue
		}
		if publishedOnly && !announcement.Published {
			continue
		}
		result = append(result, announcement.Clone())
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return result, nil
}

func (r *repositoryView) AppendAudit(ctx context.Context, entry domain.AuditEntry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := entry.Validate(); err != nil {
		return err
	}
	r.state.audits = append(r.state.audits, entry.Clone())
	return nil
}

func (r *repositoryView) ListAudit(ctx context.Context, resource, resourceID string) ([]domain.AuditEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	result := make([]domain.AuditEntry, 0)
	for _, entry := range r.state.audits {
		if resource != "" && entry.Resource != resource {
			continue
		}
		if resourceID != "" && entry.ResourceID != resourceID {
			continue
		}
		result = append(result, entry.Clone())
	}
	return result, nil
}

func (r *repositoryView) SaveToken(ctx context.Context, digest, feedbackID string, expiresAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if digest == "" || feedbackID == "" {
		return domain.ValidationError{Field: "token", Message: "token is incomplete"}
	}
	if _, exists := r.state.tokens[digest]; exists {
		return domain.ConflictError{Resource: "query token", Key: digest}
	}
	r.state.tokens[digest] = tokenRecord{FeedbackID: feedbackID, ExpiresAt: expiresAt.UTC()}
	return nil
}

func (r *repositoryView) ResolveToken(ctx context.Context, digest string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	token, exists := r.state.tokens[strings.TrimSpace(digest)]
	if !exists || !r.now().UTC().Before(token.ExpiresAt) {
		return "", domain.ErrTokenInvalid
	}
	return token.FeedbackID, nil
}

func (r *repositoryView) GetIdempotency(ctx context.Context, scope, key, requestHash string) (string, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	record, exists := r.state.idempotency[scope+":"+key]
	if !exists || !r.now().UTC().Before(record.ExpiresAt) {
		return "", false, nil
	}
	if record.RequestHash != requestHash {
		return "", false, domain.ConflictError{Resource: "idempotency key", Key: key}
	}
	return record.ResourceID, true, nil
}

func (r *repositoryView) SaveIdempotency(ctx context.Context, scope, key, requestHash, resourceID string, expiresAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	identity := scope + ":" + key
	if existing, exists := r.state.idempotency[identity]; exists {
		if existing.RequestHash == requestHash && existing.ResourceID == resourceID {
			return nil
		}
		return domain.ConflictError{Resource: "idempotency key", Key: key}
	}
	r.state.idempotency[identity] = idempotencyRecord{RequestHash: requestHash, ResourceID: resourceID, ExpiresAt: expiresAt.UTC()}
	return nil
}
