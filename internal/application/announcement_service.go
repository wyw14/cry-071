package application

import (
	"context"
	"strings"

	"github.com/wyw14/cry-071/internal/domain"
)

type AnnouncementRecords interface {
	CreateAnnouncement(context.Context, *domain.Announcement) error
	GetAnnouncement(context.Context, string) (*domain.Announcement, error)
	UpdateAnnouncement(context.Context, *domain.Announcement) error
	ListAnnouncements(context.Context, string, bool) ([]*domain.Announcement, error)
}

type AnnouncementService struct{ deps Dependencies }

func NewAnnouncementService(deps Dependencies) *AnnouncementService {
	return &AnnouncementService{deps: deps}
}

type CreateAnnouncementCommand struct {
	Title       string
	Content     string
	AreaID      string
	FeedbackIDs []string
	Actor       domain.Actor
	RequestID   string
}

func (s *AnnouncementService) Create(ctx context.Context, cmd CreateAnnouncementCommand) (*domain.Announcement, error) {
	if cmd.Actor.Role != domain.RoleManager {
		return nil, domain.ErrForbidden
	}
	now := s.deps.Clock.Now()
	announcement, err := domain.NewAnnouncement(s.deps.IDs.New("announcement"), cmd.Title, cmd.Content,
		cmd.AreaID, cmd.FeedbackIDs, now)
	if err != nil {
		return nil, err
	}
	for _, feedbackID := range announcement.FeedbackIDs {
		feedback, err := s.deps.Repositories.Get(ctx, feedbackID)
		if err != nil {
			return nil, err
		}
		if feedback.AreaID != announcement.AreaID {
			return nil, domain.ValidationError{Field: "feedback_ids", Message: "announcement feedback belongs to another area"}
		}
	}
	err = s.deps.Transactions.WithinTransaction(ctx, func(tx context.Context, repos Repositories) error {
		if err := repos.CreateAnnouncement(tx, announcement); err != nil {
			return err
		}
		return repos.AppendAudit(tx, newAudit(s.deps.IDs, cmd.Actor.ID, "announcement.create", "announcement",
			announcement.ID, cmd.RequestID, map[string]string{"area_id": announcement.AreaID}, now))
	})
	if err != nil {
		return nil, err
	}
	return announcement.Clone(), nil
}

func (s *AnnouncementService) Publish(ctx context.Context, id string, actor domain.Actor, requestID string) (*domain.Announcement, error) {
	now := s.deps.Clock.Now()
	var result *domain.Announcement
	err := s.deps.Transactions.WithinTransaction(ctx, func(tx context.Context, repos Repositories) error {
		announcement, err := repos.GetAnnouncement(tx, strings.TrimSpace(id))
		if err != nil {
			return err
		}
		if err := announcement.Publish(actor, now); err != nil {
			return err
		}
		if err := repos.UpdateAnnouncement(tx, announcement); err != nil {
			return err
		}
		for _, feedbackID := range announcement.FeedbackIDs {
			feedback, err := repos.Get(tx, feedbackID)
			if err != nil {
				return err
			}
			expected := feedback.Version
			event := newTimelineEvent(s.deps.IDs, feedback, domain.EventAnnouncementLinked, actor.ID,
				domain.VisibilityPublic, "相关问题解决公告已发布", map[string]string{"announcement_id": announcement.ID}, now)
			if err := repos.Update(tx, feedback, expected); err != nil {
				return err
			}
			if err := repos.Append(tx, event); err != nil {
				return err
			}
		}
		if err := repos.AppendAudit(tx, newAudit(s.deps.IDs, actor.ID, "announcement.publish", "announcement",
			announcement.ID, requestID, nil, now)); err != nil {
			return err
		}
		result = announcement.Clone()
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *AnnouncementService) ListPublic(ctx context.Context, areaID string) ([]*domain.Announcement, error) {
	return s.deps.Repositories.ListAnnouncements(ctx, strings.TrimSpace(areaID), true)
}
