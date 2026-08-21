package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/wyw14/cry-071/internal/domain"
)

func (s *Store) CreateMerge(ctx context.Context, group *domain.MergeGroup) error {
	payload, err := json.Marshal(group)
	if err != nil {
		return err
	}
	_, err = s.query(ctx).Exec(ctx, `INSERT INTO merge_groups (id,primary_id,created_at,document) VALUES ($1,$2,$3,$4)`, group.ID, group.PrimaryID, group.CreatedAt, payload)
	return mapError(err)
}

func (s *Store) GetMerge(ctx context.Context, id string) (*domain.MergeGroup, error) {
	var payload []byte
	err := s.query(ctx).QueryRow(ctx, `SELECT document FROM merge_groups WHERE id=$1`, id).Scan(&payload)
	if err != nil {
		return nil, mapError(err)
	}
	var group domain.MergeGroup
	if err := json.Unmarshal(payload, &group); err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *Store) UpdateMerge(ctx context.Context, group *domain.MergeGroup) error {
	payload, err := json.Marshal(group)
	if err != nil {
		return err
	}
	tag, err := s.query(ctx).Exec(ctx, `UPDATE merge_groups SET document=$2 WHERE id=$1`, group.ID, payload)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() != 1 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) CreateAnnouncement(ctx context.Context, item *domain.Announcement) error {
	payload, err := json.Marshal(item)
	if err != nil {
		return err
	}
	_, err = s.query(ctx).Exec(ctx, `INSERT INTO announcements (id,area_id,published,created_at,document) VALUES ($1,$2,$3,$4,$5)`, item.ID, item.AreaID, item.Published, item.CreatedAt, payload)
	return mapError(err)
}

func (s *Store) GetAnnouncement(ctx context.Context, id string) (*domain.Announcement, error) {
	var payload []byte
	err := s.query(ctx).QueryRow(ctx, `SELECT document FROM announcements WHERE id=$1`, id).Scan(&payload)
	if err != nil {
		return nil, mapError(err)
	}
	var item domain.Announcement
	if err := json.Unmarshal(payload, &item); err != nil {
		return nil, err
	}
	return item.Clone(), nil
}

func (s *Store) UpdateAnnouncement(ctx context.Context, item *domain.Announcement) error {
	payload, err := json.Marshal(item)
	if err != nil {
		return err
	}
	tag, err := s.query(ctx).Exec(ctx, `UPDATE announcements SET published=$2,document=$3 WHERE id=$1`, item.ID, item.Published, payload)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() != 1 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) ListAnnouncements(ctx context.Context, areaID string, publishedOnly bool) ([]*domain.Announcement, error) {
	rows, err := s.query(ctx).Query(ctx, `SELECT document FROM announcements WHERE ($1='' OR area_id=$1) AND (NOT $2 OR published) ORDER BY created_at DESC`, areaID, publishedOnly)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	result := []*domain.Announcement{}
	for rows.Next() {
		var payload []byte
		var item domain.Announcement
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(payload, &item); err != nil {
			return nil, err
		}
		result = append(result, item.Clone())
	}
	return result, rows.Err()
}

func (s *Store) GetArea(ctx context.Context, id string) (domain.PublicArea, error) {
	var item domain.PublicArea
	err := s.query(ctx).QueryRow(ctx, `SELECT id,name,district,active,created_at FROM public_areas WHERE id=$1`, id).Scan(&item.ID, &item.Name, &item.District, &item.Active, &item.Created)
	return item, mapError(err)
}

func (s *Store) ListAreas(ctx context.Context) ([]domain.PublicArea, error) {
	rows, err := s.query(ctx).Query(ctx, `SELECT id,name,district,active,created_at FROM public_areas ORDER BY name`)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	result := []domain.PublicArea{}
	for rows.Next() {
		var item domain.PublicArea
		if err := rows.Scan(&item.ID, &item.Name, &item.District, &item.Active, &item.Created); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) GetCategory(ctx context.Context, id string) (domain.FacilityCategory, error) {
	var item domain.FacilityCategory
	err := s.query(ctx).QueryRow(ctx, `SELECT id,name,description,active FROM facility_categories WHERE id=$1`, id).Scan(&item.ID, &item.Name, &item.Description, &item.Active)
	return item, mapError(err)
}

func (s *Store) GetSubject(ctx context.Context, code string) (domain.FeedbackSubject, error) {
	var item domain.FeedbackSubject
	err := s.query(ctx).QueryRow(ctx, `SELECT code,name,default_priority,default_assignee_id FROM feedback_subjects WHERE code=$1`, code).Scan(&item.Code, &item.Name, &item.DefaultPriority, &item.DefaultAssigneeID)
	return item, mapError(err)
}

func (s *Store) Trend(ctx context.Context, areaID string, from, to time.Time, bucket string) ([]domain.TrendPoint, error) {
	format := "day"
	if bucket == "week" {
		format = "week"
	} else if bucket == "month" {
		format = "month"
	}
	rows, err := s.query(ctx).Query(ctx, `
		SELECT date_trunc($1, created_at) AS bucket, area_id,
			count(*)::int,
			count(*) FILTER (WHERE status='closed')::int,
			count(*) FILTER (WHERE status='rejected')::int,
			count(*) FILTER (WHERE due_at < $4 AND status NOT IN ('closed','rejected'))::int,
			coalesce(avg(extract(epoch FROM ((document->>'ClosedAt')::timestamptz-created_at))/3600) FILTER (WHERE status='closed'),0)
		FROM feedbacks WHERE created_at >= $2 AND created_at < $3 AND ($5='' OR area_id=$5)
		GROUP BY bucket, area_id ORDER BY bucket, area_id`, format, from, to, to, areaID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	result := []domain.TrendPoint{}
	for rows.Next() {
		var item domain.TrendPoint
		if err := rows.Scan(&item.Bucket, &item.AreaID, &item.Submitted, &item.Closed, &item.Rejected, &item.Overdue, &item.AverageHours); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
