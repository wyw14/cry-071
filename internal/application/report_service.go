package application

import (
	"context"
	"strings"
	"time"

	"github.com/wyw14/cry-071/internal/domain"
)

type TrendReader interface {
	Trend(context.Context, string, time.Time, time.Time, string) ([]domain.TrendPoint, error)
}

type ReportService struct{ deps Dependencies }

func NewReportService(deps Dependencies) *ReportService { return &ReportService{deps: deps} }

type TrendQuery struct {
	AreaID string
	From   time.Time
	To     time.Time
	Bucket string
	Actor  domain.Actor
}

func (s *ReportService) Trend(ctx context.Context, query TrendQuery) ([]domain.TrendPoint, error) {
	if query.Actor.Role != domain.RoleManager && query.Actor.Role != domain.RoleAuditor {
		return nil, domain.ErrForbidden
	}
	if query.AreaID != "" && !query.Actor.CanAudit() && !query.Actor.CanManage(query.AreaID) {
		return nil, domain.ErrForbidden
	}
	if query.From.IsZero() || query.To.IsZero() || !query.From.Before(query.To) {
		return nil, domain.ValidationError{Field: "period", Message: "report period is invalid"}
	}
	if query.To.Sub(query.From) > 366*24*time.Hour {
		return nil, domain.ValidationError{Field: "period", Message: "report period exceeds one year"}
	}
	query.Bucket = strings.ToLower(strings.TrimSpace(query.Bucket))
	if query.Bucket == "" {
		query.Bucket = "day"
	}
	if query.Bucket != "day" && query.Bucket != "week" && query.Bucket != "month" {
		return nil, domain.ValidationError{Field: "bucket", Message: "bucket must be day, week or month"}
	}
	return s.deps.Repositories.Trend(ctx, query.AreaID, query.From.UTC(), query.To.UTC(), query.Bucket)
}

func (s *ReportService) Quality(ctx context.Context, areaID string, actor domain.Actor) (domain.QualitySummary, error) {
	if actor.Role != domain.RoleManager && actor.Role != domain.RoleAuditor {
		return domain.QualitySummary{}, domain.ErrForbidden
	}
	page, err := s.deps.Repositories.ListFeedbacks(ctx, FeedbackFilter{AreaID: areaID, Page: 1, PageSize: 10000, Sort: "created_at", Now: s.deps.Clock.Now()})
	if err != nil {
		return domain.QualitySummary{}, err
	}
	closed, overdue := 0, 0
	var closureHours float64
	for _, feedback := range page.Items {
		if feedback.Status == domain.StatusClosed && feedback.ClosedAt != nil {
			closed++
			closureHours += feedback.ClosedAt.Sub(feedback.CreatedAt).Hours()
		}
		if feedback.IsOverdue(s.deps.Clock.Now()) {
			overdue++
		}
	}
	average := 0.0
	if closed > 0 {
		average = closureHours / float64(closed)
	}
	return domain.QualitySummary{AreaID: areaID, OpenCount: len(page.Items) - closed,
		OverdueCount: overdue, ClosureRate: domain.Percentage(closed, len(page.Items)), AverageClosureHour: average}, nil
}
