package memory

import (
	"context"
	"sort"
	"time"

	"github.com/wyw14/cry-071/internal/domain"
)

func (r *repositoryView) GetArea(ctx context.Context, id string) (domain.PublicArea, error) {
	if err := ctx.Err(); err != nil {
		return domain.PublicArea{}, err
	}
	area, exists := r.state.areas[id]
	if !exists {
		return domain.PublicArea{}, domain.ErrNotFound
	}
	return area, nil
}

func (r *repositoryView) ListAreas(ctx context.Context) ([]domain.PublicArea, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	areas := make([]domain.PublicArea, 0, len(r.state.areas))
	for _, area := range r.state.areas {
		areas = append(areas, area)
	}
	sort.Slice(areas, func(i, j int) bool { return areas[i].Name < areas[j].Name })
	return areas, nil
}

func (r *repositoryView) GetCategory(ctx context.Context, id string) (domain.FacilityCategory, error) {
	if err := ctx.Err(); err != nil {
		return domain.FacilityCategory{}, err
	}
	category, exists := r.state.categories[id]
	if !exists {
		return domain.FacilityCategory{}, domain.ErrNotFound
	}
	return category, nil
}

func (r *repositoryView) GetSubject(ctx context.Context, code string) (domain.FeedbackSubject, error) {
	if err := ctx.Err(); err != nil {
		return domain.FeedbackSubject{}, err
	}
	subject, exists := r.state.subjects[code]
	if !exists {
		return domain.FeedbackSubject{}, domain.ErrNotFound
	}
	return subject, nil
}

type trendAccumulator struct {
	point        domain.TrendPoint
	closureHours float64
	closedCount  int
	scoreTotal   int
	scoreCount   int
}

func (r *repositoryView) Trend(ctx context.Context, areaID string, from, to time.Time, bucket string) ([]domain.TrendPoint, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	groups := map[string]*trendAccumulator{}
	for _, feedback := range r.state.feedbacks {
		if !includeTrendFeedback(feedback, areaID, from, to) {
			continue
		}
		bucketTime := truncateBucket(feedback.UpdatedAt, bucket)
		key := feedback.AreaID + ":" + bucketTime.Format(time.RFC3339)
		accumulator := groups[key]
		if accumulator == nil {
			accumulator = &trendAccumulator{
				point: domain.TrendPoint{
					Bucket: bucketTime,
					AreaID: feedback.AreaID,
				},
			}
			groups[key] = accumulator
		}
		accumulateTrend(accumulator, feedback, r.state.satisfaction, to)
	}
	result := finishTrendGroups(groups)
	sort.Slice(result, func(i, j int) bool {
		if result[i].Bucket.Equal(result[j].Bucket) {
			return result[i].AreaID < result[j].AreaID
		}
		return result[i].Bucket.Before(result[j].Bucket)
	})
	return result, nil
}

func includeTrendFeedback(feedback *domain.Feedback, areaID string, from, to time.Time) bool {
	if feedback == nil {
		return false
	}
	if areaID != "" && feedback.AreaID != areaID {
		return false
	}
	return !feedback.UpdatedAt.Before(from) && feedback.UpdatedAt.Before(to)
}

func accumulateTrend(
	accumulator *trendAccumulator,
	feedback *domain.Feedback,
	satisfaction map[string]domain.Satisfaction,
	reportEnd time.Time,
) {
	accumulator.point.Submitted++
	if feedback.Status == domain.StatusClosed {
		accumulator.point.Closed++
		if feedback.ClosedAt != nil {
			accumulator.closureHours += feedback.ClosedAt.Sub(feedback.CreatedAt).Hours()
			accumulator.closedCount++
		}
	}
	if feedback.Status == domain.StatusRejected {
		accumulator.point.Rejected++
	}
	if feedback.IsOverdue(reportEnd) {
		accumulator.point.Overdue++
	}
	if score, exists := satisfaction[feedback.ID]; exists {
		accumulator.scoreTotal += score.Score
		accumulator.scoreCount++
	}
}

func finishTrendGroups(groups map[string]*trendAccumulator) []domain.TrendPoint {
	result := make([]domain.TrendPoint, 0, len(groups))
	for _, accumulator := range groups {
		if accumulator.closedCount > 0 {
			accumulator.point.AverageHours = accumulator.closureHours / float64(accumulator.closedCount)
		}
		if accumulator.scoreCount > 0 {
			accumulator.point.SatisfactionAvg = float64(accumulator.scoreTotal) / float64(accumulator.scoreCount)
		}
		result = append(result, accumulator.point)
	}
	return result
}

func truncateBucket(value time.Time, bucket string) time.Time {
	value = value.UTC()
	switch bucket {
	case "month":
		return time.Date(value.Year(), value.Month(), 1, 0, 0, 0, 0, time.UTC)
	case "week":
		days := (int(value.Weekday()) + 6) % 7
		start := value.AddDate(0, 0, -days)
		return time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	default:
		return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
	}
}
