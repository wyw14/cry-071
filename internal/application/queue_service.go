package application

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/wyw14/cry-071/internal/domain"
)

type FeedbackFilter struct {
	AreaID      string
	Status      domain.FeedbackStatus
	Priority    domain.Priority
	AssigneeID  string
	SubjectCode string
	OverdueOnly bool
	Search      string
	Page        int
	PageSize    int
	Sort        string
	Descending  bool
	Now         time.Time
}

type FeedbackPage struct {
	Items []*domain.Feedback
	Total int
	Page  int
	Size  int
}

type FeedbackSearch interface {
	ListFeedbacks(context.Context, FeedbackFilter) (FeedbackPage, error)
}

type DuplicateMatcher interface {
	Find(context.Context, *domain.Feedback, int) ([]DuplicateCandidate, error)
}

type DuplicateCandidate struct {
	FeedbackID      string  `json:"feedback_id"`
	AcceptanceNo    string  `json:"acceptance_number"`
	Title           string  `json:"title"`
	SimilarityScore float64 `json:"similarity_score"`
}

type QueueService struct{ deps Dependencies }

func NewQueueService(deps Dependencies) *QueueService { return &QueueService{deps: deps} }

type QueueQuery struct {
	AreaID      string
	Status      domain.FeedbackStatus
	Priority    domain.Priority
	AssigneeID  string
	OverdueOnly bool
	Page        int
	PageSize    int
	Sort        string
	Descending  bool
	Actor       domain.Actor
}

func (s *QueueService) List(ctx context.Context, query QueueQuery) (FeedbackPage, error) {
	if query.Actor.Role != domain.RoleManager && query.Actor.Role != domain.RoleAgent {
		return FeedbackPage{}, domain.ErrForbidden
	}
	if query.AreaID != "" && !query.Actor.CanManage(query.AreaID) {
		return FeedbackPage{}, domain.ErrForbidden
	}
	allowedSorts := map[string]bool{"created_at": true, "updated_at": true, "due_at": true, "priority": true}
	if query.Sort == "" {
		query.Sort = "created_at"
	}
	if !allowedSorts[query.Sort] {
		return FeedbackPage{}, domain.ValidationError{Field: "sort", Message: "sort field is not allowed"}
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}
	return s.deps.Repositories.ListFeedbacks(ctx, FeedbackFilter{
		AreaID: strings.TrimSpace(query.AreaID), Status: query.Status, Priority: query.Priority,
		AssigneeID: strings.TrimSpace(query.AssigneeID), OverdueOnly: query.OverdueOnly,
		Page: query.Page, PageSize: query.PageSize, Sort: query.Sort,
		Descending: query.Descending, Now: s.deps.Clock.Now(),
	})
}

type BatchAssignCommand struct {
	FeedbackIDs map[string]int64
	AssigneeID  string
	Actor       domain.Actor
	RequestID   string
}

type BatchResult struct {
	Succeeded []string          `json:"succeeded"`
	Failed    map[string]string `json:"failed"`
}

func (s *QueueService) BatchAssign(ctx context.Context, cmd BatchAssignCommand) (BatchResult, error) {
	if cmd.Actor.Role != domain.RoleManager {
		return BatchResult{}, domain.ErrForbidden
	}
	if len(cmd.FeedbackIDs) == 0 || len(cmd.FeedbackIDs) > 100 {
		return BatchResult{}, domain.ValidationError{Field: "feedback_ids", Message: "batch size must be between 1 and 100"}
	}
	ids := make([]string, 0, len(cmd.FeedbackIDs))
	for id := range cmd.FeedbackIDs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := BatchResult{Succeeded: []string{}, Failed: map[string]string{}}
	for _, id := range ids {
		_, err := s.deps.Workflow().Assign(ctx, AssignmentCommand{
			FeedbackID: id, ExpectedVersion: cmd.FeedbackIDs[id], AssigneeID: cmd.AssigneeID,
			Actor: cmd.Actor, RequestID: cmd.RequestID,
		})
		if err != nil {
			result.Failed[id] = err.Error()
			continue
		}
		result.Succeeded = append(result.Succeeded, id)
	}
	return result, nil
}

func (d Dependencies) Workflow() *WorkflowService { return NewWorkflowService(d) }

func (s *QueueService) OverdueSweep(ctx context.Context, actor domain.Actor) (int, error) {
	if actor.Role != domain.RoleManager {
		return 0, domain.ErrForbidden
	}
	page, err := s.deps.Repositories.ListFeedbacks(ctx, FeedbackFilter{
		OverdueOnly: true, Page: 1, PageSize: 500, Sort: "due_at", Now: s.deps.Clock.Now(),
	})
	if err != nil {
		return 0, err
	}
	notified := 0
	for _, feedback := range page.Items {
		if feedback.AssigneeID == "" {
			continue
		}
		err := s.deps.Notifications.Send(ctx, Notification{
			Recipient: feedback.AssigneeID, Template: "feedback_overdue",
			Data: map[string]string{"acceptance_number": feedback.AcceptanceNumber}, CreatedAt: s.deps.Clock.Now(),
		})
		if err == nil {
			notified++
		}
	}
	return notified, nil
}
