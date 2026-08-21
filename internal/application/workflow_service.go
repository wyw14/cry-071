package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/wyw14/cry-071/internal/domain"
)

type FeedbackRecords interface {
	Get(context.Context, string) (*domain.Feedback, error)
	GetByAcceptanceNumber(context.Context, string) (*domain.Feedback, error)
	Create(context.Context, *domain.Feedback) error
	Update(context.Context, *domain.Feedback, int64) error
}

type TimelineJournal interface {
	Append(context.Context, domain.TimelineEvent) error
	ListTimeline(context.Context, string) ([]domain.TimelineEvent, error)
}

type TransitionCommand struct {
	FeedbackID      string
	ExpectedVersion int64
	To              domain.FeedbackStatus
	Reason          string
	Actor           domain.Actor
	RequestID       string
	IdempotencyKey  string
}

type WorkflowService struct{ deps Dependencies }

func NewWorkflowService(deps Dependencies) *WorkflowService { return &WorkflowService{deps: deps} }

func (s *WorkflowService) Transition(ctx context.Context, cmd TransitionCommand) (*domain.Feedback, error) {
	now := s.deps.Clock.Now()
	key := strings.TrimSpace(cmd.IdempotencyKey)
	hash := requestHash(struct {
		FeedbackID      string
		ExpectedVersion int64
		To              domain.FeedbackStatus
		Reason, ActorID string
	}{cmd.FeedbackID, cmd.ExpectedVersion, cmd.To, cmd.Reason, cmd.Actor.ID})
	if key != "" {
		resourceID, found, err := s.deps.Repositories.GetIdempotency(ctx, "feedback.transition", key, hash)
		if err != nil {
			return nil, err
		}
		if found {
			return s.deps.Repositories.Get(ctx, resourceID)
		}
	}
	var result *domain.Feedback
	err := s.deps.Transactions.WithinTransaction(ctx, func(tx context.Context, repos Repositories) error {
		feedback, err := repos.Get(tx, strings.TrimSpace(cmd.FeedbackID))
		if err != nil {
			return err
		}
		if feedback.Version != cmd.ExpectedVersion {
			return domain.ErrVersionConflict
		}
		from := feedback.Status
		if err := feedback.Transition(cmd.To, cmd.Actor, cmd.Reason, now); err != nil {
			return err
		}
		eventKind := domain.EventStatusChanged
		if from == domain.StatusClosed && cmd.To == domain.StatusInProgress {
			eventKind = domain.EventReopened
		}
		event := newTimelineEvent(s.deps.IDs, feedback, eventKind, cmd.Actor.ID,
			domain.VisibilityPublic, fmt.Sprintf("状态由%s变更为%s", from.PublicLabel(), cmd.To.PublicLabel()),
			map[string]string{"from": string(from), "to": string(cmd.To), "reason": strings.TrimSpace(cmd.Reason)}, now)
		if err := repos.Update(tx, feedback, cmd.ExpectedVersion); err != nil {
			return err
		}
		if err := repos.Append(tx, event); err != nil {
			return err
		}
		if err := repos.AppendAudit(tx, newAudit(s.deps.IDs, cmd.Actor.ID, "feedback.transition", "feedback",
			feedback.ID, cmd.RequestID, map[string]string{"from": string(from), "to": string(cmd.To)}, now)); err != nil {
			return err
		}
		if key != "" {
			if err := repos.SaveIdempotency(tx, "feedback.transition", key, hash, feedback.ID, now.Add(24*time.Hour)); err != nil {
				return err
			}
		}
		result = feedback.Clone()
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

type AssignmentCommand struct {
	FeedbackID      string
	ExpectedVersion int64
	AssigneeID      string
	Actor           domain.Actor
	RequestID       string
}

func (s *WorkflowService) Assign(ctx context.Context, cmd AssignmentCommand) (*domain.Feedback, error) {
	now := s.deps.Clock.Now()
	feedback, err := s.deps.Repositories.Get(ctx, cmd.FeedbackID)
	if err != nil {
		return nil, err
	}
	if feedback.Version != cmd.ExpectedVersion {
		return nil, domain.ErrVersionConflict
	}

	previous := feedback.AssigneeID
	if err := feedback.Assign(cmd.Actor, cmd.AssigneeID, now); err != nil {
		return nil, err
	}
	if err := s.deps.Repositories.Update(ctx, feedback, cmd.ExpectedVersion); err != nil {
		return nil, err
	}

	_ = s.deps.Notifications.Send(ctx, Notification{
		Recipient: cmd.AssigneeID,
		Template:  "feedback_assigned",
		Data: map[string]string{
			"feedback_id": cmd.FeedbackID,
		},
		CreatedAt: now,
	})

	event := newTimelineEvent(
		s.deps.IDs,
		feedback,
		domain.EventTransferred,
		cmd.Actor.ID,
		domain.VisibilityPublic,
		"处理人员已调整",
		map[string]string{
			"previous_assignee": previous,
			"new_assignee":      feedback.AssigneeID,
		},
		now,
	)
	if err := s.deps.Repositories.Append(ctx, event); err != nil {
		return nil, err
	}
	if err := s.deps.Repositories.Update(ctx, feedback, cmd.ExpectedVersion+1); err != nil {
		return nil, err
	}

	audit := newAudit(
		s.deps.IDs,
		cmd.Actor.ID,
		"feedback.assign",
		"feedback",
		feedback.ID,
		cmd.RequestID,
		map[string]string{"assignee_id": feedback.AssigneeID},
		now,
	)
	if err := s.deps.Repositories.AppendAudit(ctx, audit); err != nil {
		return nil, err
	}
	return feedback.Clone(), nil
}
