package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/wyw14/cry-071/internal/domain"
)

func TestFeedbackStateMachineRequiresAssignmentAndReopenReason(t *testing.T) {
	now := time.Date(2026, 8, 22, 8, 0, 0, 0, time.UTC)
	feedback, err := domain.CreateFeedback(domain.NewFeedback{ID: "f-1", AcceptanceNumber: "PSF-1", AreaID: "area-1", FacilityCategoryID: "lighting", SubjectCode: "safety", Priority: domain.PriorityHigh, Title: "公园路灯持续闪烁", Description: "东门附近路灯持续闪烁并发出异响", Location: "中心公园东门", SubmitterName: "张先生", CreatedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	manager := domain.Actor{ID: "manager", Role: domain.RoleManager}
	if err := feedback.Transition(domain.StatusInProgress, manager, "开始处理", now.Add(time.Hour)); err == nil {
		t.Fatal("unassigned feedback entered processing")
	}
	if err := feedback.Assign(manager, "agent-1", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := feedback.Transition(domain.StatusInProgress, manager, "开始现场处理", now.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := feedback.Transition(domain.StatusPendingConfirm, manager, "现场修复完成", now.Add(3*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := feedback.Transition(domain.StatusClosed, manager, "提交者确认", now.Add(4*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := feedback.Transition(domain.StatusInProgress, manager, "短", now.Add(5*time.Hour)); err == nil {
		t.Fatal("closed feedback reopened without specific reason")
	}
	if err := feedback.Transition(domain.StatusInProgress, manager, "夜间复查发现故障复发", now.Add(5*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if feedback.ClosedAt != nil {
		t.Fatal("reopened feedback retained closure time")
	}
}

func TestFeedbackRejectsUnsupportedTransition(t *testing.T) {
	now := time.Now().UTC()
	feedback, err := domain.CreateFeedback(domain.NewFeedback{ID: "f-2", AcceptanceNumber: "PSF-2", AreaID: "area-1", FacilityCategoryID: "lighting", SubjectCode: "damaged", Priority: domain.PriorityNormal, Title: "长椅扶手已经松动", Description: "儿童活动区旁边长椅扶手已经明显松动", Location: "儿童活动区", SubmitterName: "李女士", CreatedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	err = feedback.Transition(domain.StatusClosed, domain.Actor{ID: "m", Role: domain.RoleManager}, "直接关闭", now.Add(time.Minute))
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("expected invalid transition, got %v", err)
	}
}

func TestPublicTimelineReturnsCopiesAndFiltersInternalEvents(t *testing.T) {
	events := []domain.TimelineEvent{{ID: "e1", FeedbackID: "f", Sequence: 1, Kind: domain.EventSubmitted, Visibility: domain.VisibilityPublic, Summary: "已提交", Details: map[string]string{"a": "b"}, OccurredAt: time.Now()}, {ID: "e2", FeedbackID: "f", Sequence: 2, Kind: domain.EventInternalNoteAdded, Visibility: domain.VisibilityInternal, Summary: "内部备注", OccurredAt: time.Now()}}
	public := domain.PublicTimeline(events)
	if len(public) != 1 {
		t.Fatalf("expected one public event, got %d", len(public))
	}
	public[0].Details["a"] = "changed"
	if events[0].Details["a"] != "b" {
		t.Fatal("public timeline exposed mutable event details")
	}
}
