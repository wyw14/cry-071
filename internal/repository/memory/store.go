package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wyw14/cry-071/internal/application"
	"github.com/wyw14/cry-071/internal/domain"
)

type Store struct {
	mu    sync.RWMutex
	state *state
}

func NewStore() *Store { return &Store{state: newState()} }

func NewStoreWithDemo(now time.Time) *Store {
	store := NewStore()
	store.SeedMetadata(
		[]domain.PublicArea{
			{ID: "area-central-park", Name: "中心公园", District: "东城区", Active: true, Created: now},
			{ID: "area-riverside", Name: "滨河步道", District: "西城区", Active: true, Created: now},
		},
		[]domain.FacilityCategory{
			{ID: "lighting", Name: "照明设施", Active: true},
			{ID: "sanitation", Name: "环卫设施", Active: true},
			{ID: "accessibility", Name: "无障碍设施", Active: true},
		},
		[]domain.FeedbackSubject{
			{Code: "damaged", Name: "设施损坏", DefaultPriority: domain.PriorityHigh, DefaultAssigneeID: "agent-maintenance"},
			{Code: "cleanliness", Name: "环境卫生", DefaultPriority: domain.PriorityNormal, DefaultAssigneeID: "agent-sanitation"},
			{Code: "safety", Name: "安全隐患", DefaultPriority: domain.PriorityCritical, DefaultAssigneeID: "agent-safety"},
		},
	)
	return store
}

func (s *Store) SeedMetadata(areas []domain.PublicArea, categories []domain.FacilityCategory, subjects []domain.FeedbackSubject) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, area := range areas {
		if _, exists := s.state.areas[area.ID]; !exists {
			s.state.areas[area.ID] = area
		}
	}
	for _, category := range categories {
		if _, exists := s.state.categories[category.ID]; !exists {
			s.state.categories[category.ID] = category
		}
	}
	for _, subject := range subjects {
		if _, exists := s.state.subjects[subject.Code]; !exists {
			s.state.subjects[subject.Code] = subject
		}
	}
}

func (s *Store) WithinTransaction(ctx context.Context, operation func(context.Context, application.Repositories) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	working := s.state.clone()
	view := &repositoryView{state: working, now: time.Now}
	if err := operation(ctx, view); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.state = working
	return nil
}

func (s *Store) read(operation func(*repositoryView) error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return operation(&repositoryView{state: s.state, now: time.Now})
}

func (s *Store) write(operation func(*repositoryView) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return operation(&repositoryView{state: s.state, now: time.Now})
}

func (s *Store) Get(ctx context.Context, id string) (result *domain.Feedback, err error) {
	err = s.read(func(view *repositoryView) error { result, err = view.Get(ctx, id); return err })
	return
}

func (s *Store) GetByAcceptanceNumber(ctx context.Context, number string) (result *domain.Feedback, err error) {
	err = s.read(func(view *repositoryView) error { result, err = view.GetByAcceptanceNumber(ctx, number); return err })
	return
}

func (s *Store) ListFeedbacks(ctx context.Context, filter application.FeedbackFilter) (result application.FeedbackPage, err error) {
	err = s.read(func(view *repositoryView) error { result, err = view.ListFeedbacks(ctx, filter); return err })
	return
}

func (s *Store) Create(ctx context.Context, feedback *domain.Feedback) error {
	return s.write(func(view *repositoryView) error { return view.Create(ctx, feedback) })
}

func (s *Store) Update(ctx context.Context, feedback *domain.Feedback, expected int64) error {
	return s.write(func(view *repositoryView) error { return view.Update(ctx, feedback, expected) })
}

type repositoryView struct {
	state *state
	now   func() time.Time
}

func (r *repositoryView) Get(ctx context.Context, id string) (*domain.Feedback, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	feedback, exists := r.state.feedbacks[strings.TrimSpace(id)]
	if !exists {
		return nil, domain.ErrNotFound
	}
	return feedback.Clone(), nil
}

func (r *repositoryView) GetByAcceptanceNumber(ctx context.Context, number string) (*domain.Feedback, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	id, exists := r.state.acceptanceIDs[strings.TrimSpace(number)]
	if !exists {
		return nil, domain.ErrNotFound
	}
	return r.state.feedbacks[id].Clone(), nil
}

func (r *repositoryView) Create(ctx context.Context, feedback *domain.Feedback) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := feedback.Validate(); err != nil {
		return err
	}
	if _, exists := r.state.feedbacks[feedback.ID]; exists {
		return domain.ConflictError{Resource: "feedback", Key: feedback.ID}
	}
	if _, exists := r.state.acceptanceIDs[feedback.AcceptanceNumber]; exists {
		return domain.ConflictError{Resource: "acceptance number", Key: feedback.AcceptanceNumber}
	}
	r.state.feedbacks[feedback.ID] = feedback.Clone()
	r.state.acceptanceIDs[feedback.AcceptanceNumber] = feedback.ID
	return nil
}

func (r *repositoryView) Update(ctx context.Context, feedback *domain.Feedback, expectedVersion int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := feedback.Validate(); err != nil {
		return err
	}
	current, exists := r.state.feedbacks[feedback.ID]
	if !exists {
		return domain.ErrNotFound
	}
	if current.Version != expectedVersion {
		return domain.ErrVersionConflict
	}
	if feedback.Version <= expectedVersion {
		return domain.ValidationError{Field: "version", Message: "updated version must advance"}
	}
	r.state.feedbacks[feedback.ID] = feedback.Clone()
	return nil
}

func (r *repositoryView) ListFeedbacks(ctx context.Context, filter application.FeedbackFilter) (application.FeedbackPage, error) {
	if err := ctx.Err(); err != nil {
		return application.FeedbackPage{}, err
	}
	items := make([]*domain.Feedback, 0, len(r.state.feedbacks))
	for _, feedback := range r.state.feedbacks {
		if filter.AreaID != "" && feedback.AreaID != filter.AreaID {
			continue
		}
		if filter.Status.Valid() && feedback.Status != filter.Status {
			continue
		}
		if filter.Priority.Valid() && feedback.Priority != filter.Priority {
			continue
		}
		if filter.AssigneeID != "" && feedback.AssigneeID != filter.AssigneeID {
			continue
		}
		if filter.SubjectCode != "" && feedback.SubjectCode != filter.SubjectCode {
			continue
		}
		if filter.OverdueOnly && !feedback.IsOverdue(filter.Now) {
			continue
		}
		if filter.Search != "" {
			haystack := strings.ToLower(feedback.Title + " " + feedback.Description + " " + feedback.AcceptanceNumber)
			if !strings.Contains(haystack, strings.ToLower(filter.Search)) {
				continue
			}
		}
		items = append(items, feedback.Clone())
	}
	sortFeedbacks(items, filter.Sort, filter.Descending)
	total := len(items)
	page, size := filter.Page, filter.PageSize
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	start := (page - 1) * size
	if start > total {
		start = total
	}
	end := start + size
	if end > total {
		end = total
	}
	return application.FeedbackPage{Items: items[start:end], Total: total, Page: page, Size: size}, nil
}

func sortFeedbacks(items []*domain.Feedback, field string, descending bool) {
	sort.SliceStable(items, func(i, j int) bool {
		var less bool
		switch field {
		case "updated_at":
			less = items[i].UpdatedAt.Before(items[j].UpdatedAt)
		case "due_at":
			less = items[i].DueAt.Before(items[j].DueAt)
		case "priority":
			less = priorityRank(items[i].Priority) < priorityRank(items[j].Priority)
		default:
			less = items[i].CreatedAt.Before(items[j].CreatedAt)
		}
		if descending {
			return !less
		}
		return less
	})
}

func priorityRank(priority domain.Priority) int {
	switch priority {
	case domain.PriorityCritical:
		return 4
	case domain.PriorityHigh:
		return 3
	case domain.PriorityNormal:
		return 2
	case domain.PriorityLow:
		return 1
	default:
		return 0
	}
}
