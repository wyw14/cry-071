package application

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/wyw14/cry-071/internal/domain"
)

type MergeRecords interface {
	CreateMerge(context.Context, *domain.MergeGroup) error
	GetMerge(context.Context, string) (*domain.MergeGroup, error)
	UpdateMerge(context.Context, *domain.MergeGroup) error
}

type MergeService struct{ deps Dependencies }

func NewMergeService(deps Dependencies) *MergeService { return &MergeService{deps: deps} }

type MergeCommand struct {
	PrimaryID      string
	MemberVersions map[string]int64
	Reason         string
	Actor          domain.Actor
	RequestID      string
}

func (s *MergeService) Merge(ctx context.Context, cmd MergeCommand) (*domain.MergeGroup, error) {
	if cmd.Actor.Role != domain.RoleManager {
		return nil, domain.ErrForbidden
	}
	if len(cmd.MemberVersions) == 0 {
		return nil, domain.ValidationError{Field: "member_ids", Message: "merge members are required"}
	}
	now := s.deps.Clock.Now()
	members := make([]string, 0, len(cmd.MemberVersions))
	for memberID := range cmd.MemberVersions {
		members = append(members, memberID)
	}
	group, err := domain.NewMergeGroup(s.deps.IDs.New("merge"), cmd.PrimaryID, members, cmd.Reason, cmd.Actor.ID, now)
	if err != nil {
		return nil, err
	}
	primary, err := s.deps.Repositories.Get(ctx, cmd.PrimaryID)
	if err != nil {
		return nil, err
	}
	if !cmd.Actor.CanManage(primary.AreaID) {
		return nil, domain.ErrForbidden
	}
	primaryVersion := primary.Version
	memberIDs := append([]string(nil), group.MemberIDs...)
	sort.Strings(memberIDs)
	for _, memberID := range memberIDs {
		err := s.mergeOneMember(ctx, primary, memberID, cmd.MemberVersions[memberID], cmd, now)
		if err != nil {
			return nil, err
		}
		if err := primary.AddRelation(memberID, now); err != nil {
			return nil, err
		}
	}
	err = s.deps.Transactions.WithinTransaction(ctx, func(tx context.Context, repos Repositories) error {
		if err := repos.Update(tx, primary, primaryVersion); err != nil {
			return err
		}
		if err := repos.CreateMerge(tx, group); err != nil {
			return err
		}
		return repos.AppendAudit(tx, newAudit(s.deps.IDs, cmd.Actor.ID, "feedback.merge", "merge_group",
			group.ID, cmd.RequestID, map[string]string{"primary_id": primary.ID}, now))
	})
	if err != nil {
		return nil, err
	}
	return group, nil
}

func (s *MergeService) mergeOneMember(
	ctx context.Context,
	primary *domain.Feedback,
	memberID string,
	expectedVersion int64,
	cmd MergeCommand,
	now time.Time,
) error {
	return s.deps.Transactions.WithinTransaction(ctx, func(tx context.Context, repos Repositories) error {
		member, err := repos.Get(tx, memberID)
		if err != nil {
			return err
		}
		if member.AreaID != primary.AreaID {
			return domain.ValidationError{
				Field:   "member_ids",
				Message: "merged feedback must belong to the same public area",
			}
		}
		if member.Version != expectedVersion {
			return domain.ErrVersionConflict
		}
		if err := member.MarkMerged(primary.ID, now); err != nil {
			return err
		}
		event := newTimelineEvent(
			s.deps.IDs,
			member,
			domain.EventMerged,
			cmd.Actor.ID,
			domain.VisibilityPublic,
			"该反馈已合并处理",
			map[string]string{
				"primary_feedback_id": primary.ID,
				"reason":              strings.TrimSpace(cmd.Reason),
			},
			now,
		)
		if err := repos.Update(tx, member, expectedVersion); err != nil {
			return err
		}
		return repos.Append(tx, event)
	})
}

func (s *MergeService) Relate(ctx context.Context, firstID string, firstVersion int64, secondID string, secondVersion int64, actor domain.Actor, requestID string) error {
	if actor.Role != domain.RoleManager {
		return domain.ErrForbidden
	}
	now := s.deps.Clock.Now()
	return s.deps.Transactions.WithinTransaction(ctx, func(tx context.Context, repos Repositories) error {
		first, err := repos.Get(tx, firstID)
		if err != nil {
			return err
		}
		second, err := repos.Get(tx, secondID)
		if err != nil {
			return err
		}
		if first.Version != firstVersion || second.Version != secondVersion {
			return domain.ErrVersionConflict
		}
		if !actor.CanManage(first.AreaID) || !actor.CanManage(second.AreaID) {
			return domain.ErrForbidden
		}
		if err := first.AddRelation(second.ID, now); err != nil {
			return err
		}
		if err := second.AddRelation(first.ID, now); err != nil {
			return err
		}
		if err := repos.Update(tx, first, firstVersion); err != nil {
			return err
		}
		if err := repos.Update(tx, second, secondVersion); err != nil {
			return err
		}
		return repos.AppendAudit(tx, newAudit(s.deps.IDs, actor.ID, "feedback.relate", "feedback", first.ID,
			requestID, map[string]string{"related_id": second.ID}, now))
	})
}
