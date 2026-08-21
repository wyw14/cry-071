package memory

import (
	"context"
	"time"

	"github.com/wyw14/cry-071/internal/domain"
)

func (s *Store) Append(ctx context.Context, event domain.TimelineEvent) error {
	return s.write(func(view *repositoryView) error { return view.Append(ctx, event) })
}

func (s *Store) ListTimeline(ctx context.Context, feedbackID string) (result []domain.TimelineEvent, err error) {
	err = s.read(func(view *repositoryView) error { result, err = view.ListTimeline(ctx, feedbackID); return err })
	return
}

func (s *Store) CreateReply(ctx context.Context, reply domain.Reply) error {
	return s.write(func(view *repositoryView) error { return view.CreateReply(ctx, reply) })
}

func (s *Store) ListReplies(ctx context.Context, feedbackID string) (result []domain.Reply, err error) {
	err = s.read(func(view *repositoryView) error { result, err = view.ListReplies(ctx, feedbackID); return err })
	return
}

func (s *Store) CreateSupplement(ctx context.Context, supplement domain.Supplement) error {
	return s.write(func(view *repositoryView) error { return view.CreateSupplement(ctx, supplement) })
}

func (s *Store) SaveSatisfaction(ctx context.Context, satisfaction domain.Satisfaction) error {
	return s.write(func(view *repositoryView) error { return view.SaveSatisfaction(ctx, satisfaction) })
}

func (s *Store) GetSatisfaction(ctx context.Context, feedbackID string) (result *domain.Satisfaction, err error) {
	err = s.read(func(view *repositoryView) error { result, err = view.GetSatisfaction(ctx, feedbackID); return err })
	return
}

func (s *Store) CreateAttachment(ctx context.Context, attachment domain.Attachment) error {
	return s.write(func(view *repositoryView) error { return view.CreateAttachment(ctx, attachment) })
}

func (s *Store) GetAttachment(ctx context.Context, id string) (result *domain.Attachment, err error) {
	err = s.read(func(view *repositoryView) error { result, err = view.GetAttachment(ctx, id); return err })
	return
}

func (s *Store) ListAttachments(ctx context.Context, feedbackID string) (result []domain.Attachment, err error) {
	err = s.read(func(view *repositoryView) error { result, err = view.ListAttachments(ctx, feedbackID); return err })
	return
}

func (s *Store) CreateMerge(ctx context.Context, group *domain.MergeGroup) error {
	return s.write(func(view *repositoryView) error { return view.CreateMerge(ctx, group) })
}

func (s *Store) GetMerge(ctx context.Context, id string) (result *domain.MergeGroup, err error) {
	err = s.read(func(view *repositoryView) error { result, err = view.GetMerge(ctx, id); return err })
	return
}

func (s *Store) UpdateMerge(ctx context.Context, group *domain.MergeGroup) error {
	return s.write(func(view *repositoryView) error { return view.UpdateMerge(ctx, group) })
}

func (s *Store) CreateAnnouncement(ctx context.Context, announcement *domain.Announcement) error {
	return s.write(func(view *repositoryView) error { return view.CreateAnnouncement(ctx, announcement) })
}

func (s *Store) GetAnnouncement(ctx context.Context, id string) (result *domain.Announcement, err error) {
	err = s.read(func(view *repositoryView) error { result, err = view.GetAnnouncement(ctx, id); return err })
	return
}

func (s *Store) UpdateAnnouncement(ctx context.Context, announcement *domain.Announcement) error {
	return s.write(func(view *repositoryView) error { return view.UpdateAnnouncement(ctx, announcement) })
}

func (s *Store) ListAnnouncements(ctx context.Context, areaID string, publishedOnly bool) (result []*domain.Announcement, err error) {
	err = s.read(func(view *repositoryView) error {
		result, err = view.ListAnnouncements(ctx, areaID, publishedOnly)
		return err
	})
	return
}

func (s *Store) AppendAudit(ctx context.Context, entry domain.AuditEntry) error {
	return s.write(func(view *repositoryView) error { return view.AppendAudit(ctx, entry) })
}

func (s *Store) ListAudit(ctx context.Context, resource, resourceID string) (result []domain.AuditEntry, err error) {
	err = s.read(func(view *repositoryView) error { result, err = view.ListAudit(ctx, resource, resourceID); return err })
	return
}

func (s *Store) SaveToken(ctx context.Context, digest, feedbackID string, expiresAt time.Time) error {
	return s.write(func(view *repositoryView) error { return view.SaveToken(ctx, digest, feedbackID, expiresAt) })
}

func (s *Store) ResolveToken(ctx context.Context, digest string) (result string, err error) {
	err = s.read(func(view *repositoryView) error { result, err = view.ResolveToken(ctx, digest); return err })
	return
}

func (s *Store) GetIdempotency(ctx context.Context, scope, key, requestHash string) (resourceID string, found bool, err error) {
	err = s.read(func(view *repositoryView) error {
		resourceID, found, err = view.GetIdempotency(ctx, scope, key, requestHash)
		return err
	})
	return
}

func (s *Store) SaveIdempotency(ctx context.Context, scope, key, requestHash, resourceID string, expiresAt time.Time) error {
	return s.write(func(view *repositoryView) error {
		return view.SaveIdempotency(ctx, scope, key, requestHash, resourceID, expiresAt)
	})
}

func (s *Store) GetArea(ctx context.Context, id string) (result domain.PublicArea, err error) {
	err = s.read(func(view *repositoryView) error { result, err = view.GetArea(ctx, id); return err })
	return
}

func (s *Store) ListAreas(ctx context.Context) (result []domain.PublicArea, err error) {
	err = s.read(func(view *repositoryView) error { result, err = view.ListAreas(ctx); return err })
	return
}

func (s *Store) GetCategory(ctx context.Context, id string) (result domain.FacilityCategory, err error) {
	err = s.read(func(view *repositoryView) error { result, err = view.GetCategory(ctx, id); return err })
	return
}

func (s *Store) GetSubject(ctx context.Context, code string) (result domain.FeedbackSubject, err error) {
	err = s.read(func(view *repositoryView) error { result, err = view.GetSubject(ctx, code); return err })
	return
}

func (s *Store) Trend(ctx context.Context, areaID string, from, to time.Time, bucket string) (result []domain.TrendPoint, err error) {
	err = s.read(func(view *repositoryView) error { result, err = view.Trend(ctx, areaID, from, to, bucket); return err })
	return
}
