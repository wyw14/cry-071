package application

import (
	"context"
	"strings"
	"time"

	"github.com/wyw14/cry-071/internal/domain"
)

type Notification struct {
	Recipient string
	Template  string
	Data      map[string]string
	CreatedAt time.Time
}

type NotificationSink interface {
	Send(context.Context, Notification) error
}

type CommunicationRecords interface {
	CreateReply(context.Context, domain.Reply) error
	ListReplies(context.Context, string) ([]domain.Reply, error)
	CreateSupplement(context.Context, domain.Supplement) error
	SaveSatisfaction(context.Context, domain.Satisfaction) error
	GetSatisfaction(context.Context, string) (*domain.Satisfaction, error)
}

type CommunicationService struct{ deps Dependencies }

func NewCommunicationService(deps Dependencies) *CommunicationService {
	return &CommunicationService{deps: deps}
}

type ReplyCommand struct {
	FeedbackID string
	Content    string
	Public     bool
	Actor      domain.Actor
	RequestID  string
}

func (s *CommunicationService) Reply(ctx context.Context, cmd ReplyCommand) (domain.Reply, error) {
	now := s.deps.Clock.Now()
	feedback, err := s.deps.Repositories.Get(ctx, cmd.FeedbackID)
	if err != nil {
		return domain.Reply{}, err
	}
	if !cmd.Actor.CanManage(feedback.AreaID) {
		return domain.Reply{}, domain.ErrForbidden
	}
	reply := domain.Reply{
		ID: s.deps.IDs.New("reply"), FeedbackID: feedback.ID, ActorID: cmd.Actor.ID,
		Content: strings.TrimSpace(cmd.Content), Public: cmd.Public, CreatedAt: now,
	}
	if err := reply.Validate(); err != nil {
		return domain.Reply{}, err
	}
	visibility := domain.VisibilityInternal
	message := "已添加内部备注"
	kind := domain.EventInternalNoteAdded
	if cmd.Public {
		visibility = domain.VisibilityPublic
		message = "处理人员已回复"
		kind = domain.EventReplyAdded
	}
	err = s.deps.Transactions.WithinTransaction(ctx, func(tx context.Context, repos Repositories) error {
		current, err := repos.Get(tx, feedback.ID)
		if err != nil {
			return err
		}
		expectedVersion := current.Version
		event := newTimelineEvent(s.deps.IDs, current, kind, cmd.Actor.ID, visibility, message,
			map[string]string{"reply_id": reply.ID}, now)
		if err := repos.CreateReply(tx, reply); err != nil {
			return err
		}
		if err := repos.Update(tx, current, expectedVersion); err != nil {
			return err
		}
		if err := repos.Append(tx, event); err != nil {
			return err
		}
		return repos.AppendAudit(tx, newAudit(s.deps.IDs, cmd.Actor.ID, "feedback.reply", "feedback",
			feedback.ID, cmd.RequestID, map[string]string{"public": boolText(cmd.Public)}, now))
	})
	if err != nil {
		return domain.Reply{}, err
	}
	return reply, nil
}

type SupplementCommand struct {
	QueryToken string
	Content    string
	RequestID  string
}

func (s *CommunicationService) Supplement(ctx context.Context, cmd SupplementCommand) (domain.Supplement, error) {
	digest := s.deps.Tokens.Digest(strings.TrimSpace(cmd.QueryToken))
	feedbackID, err := s.deps.Repositories.ResolveToken(ctx, digest)
	if err != nil {
		return domain.Supplement{}, err
	}
	now := s.deps.Clock.Now()
	supplement := domain.Supplement{
		ID: s.deps.IDs.New("supplement"), FeedbackID: feedbackID,
		Content: strings.TrimSpace(cmd.Content), CreatedAt: now,
	}
	if err := supplement.Validate(); err != nil {
		return domain.Supplement{}, err
	}
	err = s.deps.Transactions.WithinTransaction(ctx, func(tx context.Context, repos Repositories) error {
		feedback, err := repos.Get(tx, feedbackID)
		if err != nil {
			return err
		}
		if feedback.Status != domain.StatusNeedsInformation {
			return domain.ValidationError{Field: "status", Message: "feedback is not waiting for material"}
		}
		expectedVersion := feedback.Version
		event := newTimelineEvent(s.deps.IDs, feedback, domain.EventMaterialAdded, feedback.SubmitterID,
			domain.VisibilityPublic, "提交者已补充材料", map[string]string{"supplement_id": supplement.ID}, now)
		if err := repos.CreateSupplement(tx, supplement); err != nil {
			return err
		}
		if err := repos.Update(tx, feedback, expectedVersion); err != nil {
			return err
		}
		if err := repos.Append(tx, event); err != nil {
			return err
		}
		return repos.AppendAudit(tx, newAudit(s.deps.IDs, feedback.SubmitterID, "feedback.supplement", "feedback",
			feedback.ID, cmd.RequestID, nil, now))
	})
	if err != nil {
		return domain.Supplement{}, err
	}
	return supplement, nil
}

type SatisfactionCommand struct {
	QueryToken string
	Score      int
	Comment    string
	RequestID  string
}

func (s *CommunicationService) ConfirmSatisfaction(ctx context.Context, cmd SatisfactionCommand) (*domain.Feedback, error) {
	digest := s.deps.Tokens.Digest(strings.TrimSpace(cmd.QueryToken))
	feedbackID, err := s.deps.Repositories.ResolveToken(ctx, digest)
	if err != nil {
		return nil, err
	}
	now := s.deps.Clock.Now()
	var result *domain.Feedback
	err = s.deps.Transactions.WithinTransaction(ctx, func(tx context.Context, repos Repositories) error {
		feedback, err := repos.Get(tx, feedbackID)
		if err != nil {
			return err
		}
		satisfaction := domain.Satisfaction{FeedbackID: feedbackID, Score: cmd.Score, Comment: cmd.Comment, CreatedAt: now}
		if err := satisfaction.Validate(feedback.Status); err != nil {
			return err
		}
		expected := feedback.Version
		if feedback.Status == domain.StatusPendingConfirm {
			actor := domain.Actor{ID: feedback.SubmitterID, Role: domain.RoleManager}
			if err := feedback.Transition(domain.StatusClosed, actor, "提交者确认处理结果", now); err != nil {
				return err
			}
		}
		event := newTimelineEvent(s.deps.IDs, feedback, domain.EventSatisfactionGiven, feedback.SubmitterID,
			domain.VisibilityPublic, "提交者已确认处理结果", map[string]string{"score": scoreText(cmd.Score)}, now)
		if err := repos.SaveSatisfaction(tx, satisfaction); err != nil {
			return err
		}
		if err := repos.Update(tx, feedback, expected); err != nil {
			return err
		}
		if err := repos.Append(tx, event); err != nil {
			return err
		}
		result = feedback.Clone()
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func scoreText(score int) string { return string(rune('0' + score)) }
