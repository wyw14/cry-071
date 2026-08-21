package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/wyw14/cry-071/internal/application"
	"github.com/wyw14/cry-071/internal/domain"
)

var allowedFeedbackSorts = map[string]string{
	"created_at": "created_at", "updated_at": "updated_at", "due_at": "due_at", "priority": "priority",
}

func (s *Store) ListFeedbacks(ctx context.Context, filter application.FeedbackFilter) (application.FeedbackPage, error) {
	clauses := []string{"TRUE"}
	args := make([]any, 0)
	add := func(clause string, value any) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf(clause, len(args)))
	}
	if filter.AreaID != "" {
		add("area_id=$%d", filter.AreaID)
	}
	if filter.Status.Valid() {
		add("status=$%d", filter.Status)
	}
	if filter.Priority.Valid() {
		add("priority=$%d", filter.Priority)
	}
	if filter.AssigneeID != "" {
		add("assignee_id=$%d", filter.AssigneeID)
	}
	if filter.SubjectCode != "" {
		add("subject_code=$%d", filter.SubjectCode)
	}
	if filter.OverdueOnly {
		add("due_at < $%d", filter.Now)
		clauses = append(clauses, `status NOT IN ('closed','rejected')`)
	}
	if filter.Search != "" {
		add("document::text ILIKE '%%' || $%d || '%%'", filter.Search)
	}
	where := strings.Join(clauses, " AND ")
	var total int
	if err := s.query(ctx).QueryRow(ctx, "SELECT count(*) FROM feedbacks WHERE "+where, args...).Scan(&total); err != nil {
		return application.FeedbackPage{}, mapError(err)
	}
	sortColumn := allowedFeedbackSorts[filter.Sort]
	if sortColumn == "" {
		sortColumn = "created_at"
	}
	direction := "ASC"
	if filter.Descending {
		direction = "DESC"
	}
	page, size := filter.Page, filter.PageSize
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 10000 {
		size = 10000
	}
	queryArgs := append(append([]any(nil), args...), size, (page-1)*size)
	rows, err := s.query(ctx).Query(ctx, fmt.Sprintf("SELECT document FROM feedbacks WHERE %s ORDER BY %s %s, id ASC LIMIT $%d OFFSET $%d",
		where, sortColumn, direction, len(args)+1, len(args)+2), queryArgs...)
	if err != nil {
		return application.FeedbackPage{}, mapError(err)
	}
	defer rows.Close()
	items := make([]*domain.Feedback, 0)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return application.FeedbackPage{}, err
		}
		var feedback domain.Feedback
		if err := json.Unmarshal(payload, &feedback); err != nil {
			return application.FeedbackPage{}, err
		}
		items = append(items, feedback.Clone())
	}
	if err := rows.Err(); err != nil {
		return application.FeedbackPage{}, err
	}
	return application.FeedbackPage{Items: items, Total: total, Page: page, Size: size}, nil
}
