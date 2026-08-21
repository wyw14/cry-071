package application_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/wyw14/cry-071/internal/application"
	"github.com/wyw14/cry-071/internal/domain"
	"github.com/wyw14/cry-071/internal/platform/files"
	"github.com/wyw14/cry-071/internal/platform/identifier"
	"github.com/wyw14/cry-071/internal/platform/notification"
	"github.com/wyw14/cry-071/internal/repository/memory"
	"github.com/wyw14/cry-071/internal/service"
)

type testSystem struct {
	services      *application.Services
	store         *memory.Store
	notifications *notification.LocalSink
	now           time.Time
}

func TestSubmissionIdempotencyReturnsExistingFeedbackAndRejectsChangedRequest(t *testing.T) {
	system := newTestSystem(t)
	command := application.SubmitFeedbackCommand{AreaID: "area-central-park", FacilityCategoryID: "lighting", SubjectCode: "safety", Priority: domain.PriorityHigh, Title: "东门照明线路出现故障", Description: "东门台阶附近照明全部熄灭影响夜间通行安全", Location: "中心公园东门", SubmitterID: "resident-2", SubmitterName: "赵女士", IdempotencyKey: "submit-once", RequestID: "first"}
	first, err := system.services.Submission.Submit(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	command.RequestID = "retry"
	second, err := system.services.Submission.Submit(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	if first.Feedback.ID != second.Feedback.ID {
		t.Fatal("idempotent retry created another feedback")
	}
	if second.QueryToken == "" {
		t.Fatal("idempotent retry did not return a usable query token")
	}
	command.Title = "同一键却修改了标题"
	_, err = system.services.Submission.Submit(context.Background(), command)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("changed idempotent request returned %v", err)
	}
	audit, err := system.store.ListAudit(context.Background(), "feedback", first.Feedback.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(audit) != 1 {
		t.Fatalf("idempotent retry duplicated audit records: %d", len(audit))
	}
}

func newTestSystem(t *testing.T) *testSystem {
	t.Helper()
	now := time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC)
	store := memory.NewStoreWithDemo(now)
	ids := &identifier.Sequence{}
	tokens, err := service.NewTokenCodec("test-pepper-is-long-enough")
	if err != nil {
		t.Fatal(err)
	}
	objects, err := files.NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	notices := notification.NewLocalSink()
	services, err := application.NewServices(application.Dependencies{Transactions: store, Repositories: store, Clock: fixedTime{at: now}, IDs: ids, Tokens: tokens, Notifications: notices, Objects: objects, Redactor: service.Redactor{}, Duplicates: service.NewDuplicateMatcher(store)})
	if err != nil {
		t.Fatal(err)
	}
	return &testSystem{services: services, store: store, notifications: notices, now: now}
}

type fixedTime struct{ at time.Time }

func (f fixedTime) Now() time.Time { return f.at }

func submitFixture(t *testing.T, system *testSystem, title, location string) application.SubmissionResult {
	t.Helper()
	result, err := system.services.Submission.Submit(context.Background(), application.SubmitFeedbackCommand{AreaID: "area-central-park", FacilityCategoryID: "lighting", SubjectCode: "safety", Priority: domain.PriorityHigh, Title: title, Description: "现场设施出现异常并影响夜间通行安全，需要尽快检查处理", Location: location, SubmitterID: "resident-1", SubmitterName: "王女士", SubmitterContact: "13800138000", RequestID: "request-test"})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestSubmissionCreatesTokenTimelineAuditAndRedactedPublicView(t *testing.T) {
	system := newTestSystem(t)
	result := submitFixture(t, system, "公园东门路灯反复闪烁", "中心公园东门")
	if result.QueryToken == "" {
		t.Fatal("query token was empty")
	}
	if result.Feedback.AcceptanceNumber == "" {
		t.Fatal("acceptance number was empty")
	}
	view, err := system.services.Submission.ViewByToken(context.Background(), result.QueryToken)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Timeline) != 1 || view.Timeline[0].Kind != domain.EventSubmitted {
		t.Fatalf("unexpected timeline: %#v", view.Timeline)
	}
	if view.Feedback.SubmitterContact != "138****8000" {
		t.Fatalf("contact was not redacted: %s", view.Feedback.SubmitterContact)
	}
	audit, err := system.store.ListAudit(context.Background(), "feedback", result.Feedback.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(audit) != 1 || audit[0].Action != "feedback.submit" {
		t.Fatalf("unexpected audit: %#v", audit)
	}
	if len(system.notifications.List()) != 1 {
		t.Fatal("local notification was not recorded")
	}
}

func TestWorkflowProducesImmutablePublicTimelineAndHidesInternalNote(t *testing.T) {
	system := newTestSystem(t)
	submitted := submitFixture(t, system, "儿童区照明设施不亮", "儿童活动区入口")
	manager := domain.Actor{ID: "manager", Role: domain.RoleManager}
	assigned, err := system.services.Workflow.Assign(context.Background(), application.AssignmentCommand{FeedbackID: submitted.Feedback.ID, ExpectedVersion: submitted.Feedback.Version, AssigneeID: "agent-light", Actor: manager, RequestID: "assign"})
	if err != nil {
		t.Fatal(err)
	}
	processing, err := system.services.Workflow.Transition(context.Background(), application.TransitionCommand{FeedbackID: assigned.ID, ExpectedVersion: assigned.Version, To: domain.StatusInProgress, Reason: "照明班组已到场", Actor: manager, RequestID: "transition"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = system.services.Communication.Reply(context.Background(), application.ReplyCommand{FeedbackID: processing.ID, Content: "电缆接头受潮，准备更换", Public: false, Actor: manager, RequestID: "note"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = system.services.Communication.Reply(context.Background(), application.ReplyCommand{FeedbackID: processing.ID, Content: "维修人员正在更换受潮接头", Public: true, Actor: manager, RequestID: "reply"})
	if err != nil {
		t.Fatal(err)
	}
	view, err := system.services.Submission.ViewByToken(context.Background(), submitted.QueryToken)
	if err != nil {
		t.Fatal(err)
	}
	for _, reply := range view.Replies {
		if !reply.Public {
			t.Fatal("internal note leaked into public view")
		}
	}
	for _, event := range view.Timeline {
		if event.Visibility != domain.VisibilityPublic {
			t.Fatal("internal timeline event leaked")
		}
	}
	all, err := system.store.ListTimeline(context.Background(), processing.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) <= len(view.Timeline) {
		t.Fatal("internal event was not retained for staff")
	}
}

func TestMergeAndAnnouncementLinkRelatedFeedback(t *testing.T) {
	system := newTestSystem(t)
	primary := submitFixture(t, system, "中心广场两盏路灯熄灭", "中心广场北侧")
	member := submitFixture(t, system, "广场北侧夜间照明不足", "中心广场北侧")
	manager := domain.Actor{ID: "manager", Role: domain.RoleManager}
	group, err := system.services.Merge.Merge(context.Background(), application.MergeCommand{PrimaryID: primary.Feedback.ID, MemberVersions: map[string]int64{member.Feedback.ID: member.Feedback.Version}, Reason: "相同位置和设施故障统一处理", Actor: manager, RequestID: "merge"})
	if err != nil {
		t.Fatal(err)
	}
	if group.PrimaryID != primary.Feedback.ID {
		t.Fatal("wrong primary feedback")
	}
	announcement, err := system.services.Announcement.Create(context.Background(), application.CreateAnnouncementCommand{Title: "中心广场照明恢复公告", Content: "中心广场北侧照明线路已经完成检修并恢复运行。", AreaID: "area-central-park", FeedbackIDs: []string{primary.Feedback.ID, member.Feedback.ID}, Actor: manager, RequestID: "announcement"})
	if err != nil {
		t.Fatal(err)
	}
	published, err := system.services.Announcement.Publish(context.Background(), announcement.ID, manager, "publish")
	if err != nil {
		t.Fatal(err)
	}
	if !published.Published {
		t.Fatal("announcement remained unpublished")
	}
	public, err := system.services.Announcement.ListPublic(context.Background(), "area-central-park")
	if err != nil {
		t.Fatal(err)
	}
	if len(public) != 1 {
		t.Fatalf("expected one public announcement, got %d", len(public))
	}
}

func TestControlledAttachmentAccessUsesStaffOrQueryToken(t *testing.T) {
	system := newTestSystem(t)
	submitted := submitFixture(t, system, "无障碍坡道扶手破损", "中心公园南门")
	content := []byte("local attachment evidence")
	attachment, err := system.services.Attachment.Upload(context.Background(), application.UploadAttachmentCommand{FeedbackID: submitted.Feedback.ID, FileName: "evidence.pdf", ContentType: "application/pdf", Size: int64(len(content)), Reader: bytes.NewReader(content), Actor: domain.Actor{ID: "resident-1", Role: domain.RoleSubmitter}, RequestID: "upload"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := system.services.Attachment.Open(context.Background(), attachment.ID, domain.Actor{}, ""); err != domain.ErrForbidden {
		t.Fatalf("anonymous attachment access returned %v", err)
	}
	_, reader, err := system.services.Attachment.Open(context.Background(), attachment.ID, domain.Actor{}, submitted.QueryToken)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	downloaded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(downloaded, content) {
		t.Fatal("downloaded content changed")
	}
}
