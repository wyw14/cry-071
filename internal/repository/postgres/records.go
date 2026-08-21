package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/wyw14/cry-071/internal/domain"
)

func marshal(value any) ([]byte, error) { return json.Marshal(value) }

func (s *Store) Append(ctx context.Context, event domain.TimelineEvent) error {
	payload, err := marshal(event)
	if err != nil {
		return err
	}
	_, err = s.query(ctx).Exec(ctx, `INSERT INTO timeline_events (id,feedback_id,sequence,visibility,occurred_at,document) VALUES ($1,$2,$3,$4,$5,$6)`,
		event.ID, event.FeedbackID, event.Sequence, event.Visibility, event.OccurredAt, payload)
	return mapError(err)
}

func (s *Store) ListTimeline(ctx context.Context, feedbackID string) ([]domain.TimelineEvent, error) {
	rows, err := s.query(ctx).Query(ctx, `SELECT document FROM timeline_events WHERE feedback_id=$1 ORDER BY sequence`, feedbackID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	result := []domain.TimelineEvent{}
	for rows.Next() {
		var payload []byte
		var item domain.TimelineEvent
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(payload, &item); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) CreateReply(ctx context.Context, reply domain.Reply) error {
	payload, err := marshal(reply)
	if err != nil {
		return err
	}
	_, err = s.query(ctx).Exec(ctx, `INSERT INTO replies (id,feedback_id,public,created_at,document) VALUES ($1,$2,$3,$4,$5)`, reply.ID, reply.FeedbackID, reply.Public, reply.CreatedAt, payload)
	return mapError(err)
}

func (s *Store) ListReplies(ctx context.Context, feedbackID string) ([]domain.Reply, error) {
	rows, err := s.query(ctx).Query(ctx, `SELECT document FROM replies WHERE feedback_id=$1 ORDER BY created_at`, feedbackID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	result := []domain.Reply{}
	for rows.Next() {
		var payload []byte
		var item domain.Reply
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(payload, &item); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) CreateSupplement(ctx context.Context, item domain.Supplement) error {
	payload, err := marshal(item)
	if err != nil {
		return err
	}
	_, err = s.query(ctx).Exec(ctx, `INSERT INTO supplements (id,feedback_id,created_at,document) VALUES ($1,$2,$3,$4)`, item.ID, item.FeedbackID, item.CreatedAt, payload)
	return mapError(err)
}

func (s *Store) SaveSatisfaction(ctx context.Context, item domain.Satisfaction) error {
	payload, err := marshal(item)
	if err != nil {
		return err
	}
	_, err = s.query(ctx).Exec(ctx, `INSERT INTO satisfaction (feedback_id,score,created_at,document) VALUES ($1,$2,$3,$4)`, item.FeedbackID, item.Score, item.CreatedAt, payload)
	return mapError(err)
}

func (s *Store) GetSatisfaction(ctx context.Context, feedbackID string) (*domain.Satisfaction, error) {
	var payload []byte
	err := s.query(ctx).QueryRow(ctx, `SELECT document FROM satisfaction WHERE feedback_id=$1`, feedbackID).Scan(&payload)
	if err != nil {
		return nil, mapError(err)
	}
	var item domain.Satisfaction
	if err := json.Unmarshal(payload, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) CreateAttachment(ctx context.Context, item domain.Attachment) error {
	payload, err := marshal(item)
	if err != nil {
		return err
	}
	_, err = s.query(ctx).Exec(ctx, `INSERT INTO attachments (id,feedback_id,created_at,document) VALUES ($1,$2,$3,$4)`, item.ID, item.FeedbackID, item.CreatedAt, payload)
	return mapError(err)
}

func (s *Store) GetAttachment(ctx context.Context, id string) (*domain.Attachment, error) {
	var payload []byte
	err := s.query(ctx).QueryRow(ctx, `SELECT document FROM attachments WHERE id=$1`, id).Scan(&payload)
	if err != nil {
		return nil, mapError(err)
	}
	var item domain.Attachment
	if err := json.Unmarshal(payload, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) ListAttachments(ctx context.Context, feedbackID string) ([]domain.Attachment, error) {
	rows, err := s.query(ctx).Query(ctx, `SELECT document FROM attachments WHERE feedback_id=$1 ORDER BY created_at`, feedbackID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	result := []domain.Attachment{}
	for rows.Next() {
		var payload []byte
		var item domain.Attachment
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(payload, &item); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) SaveToken(ctx context.Context, digest, feedbackID string, expiresAt time.Time) error {
	_, err := s.query(ctx).Exec(ctx, `INSERT INTO query_tokens (digest,feedback_id,expires_at) VALUES ($1,$2,$3)`, digest, feedbackID, expiresAt)
	return mapError(err)
}

func (s *Store) ResolveToken(ctx context.Context, digest string) (string, error) {
	var feedbackID string
	err := s.query(ctx).QueryRow(ctx, `SELECT feedback_id FROM query_tokens WHERE digest=$1 AND expires_at>now()`, digest).Scan(&feedbackID)
	return feedbackID, mapError(err)
}

func (s *Store) GetIdempotency(ctx context.Context, scope, key, requestHash string) (string, bool, error) {
	var resourceID, storedHash string
	err := s.query(ctx).QueryRow(ctx, `SELECT request_hash,resource_id FROM idempotency_keys WHERE scope=$1 AND key=$2 AND expires_at>now()`, scope, key).Scan(&storedHash, &resourceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, mapError(err)
	}
	if storedHash != requestHash {
		return "", false, domain.ConflictError{Resource: "idempotency key", Key: key}
	}
	return resourceID, true, nil
}

func (s *Store) SaveIdempotency(ctx context.Context, scope, key, requestHash, resourceID string, expiresAt time.Time) error {
	_, err := s.query(ctx).Exec(ctx, `INSERT INTO idempotency_keys(scope,key,request_hash,resource_id,expires_at) VALUES($1,$2,$3,$4,$5)`, scope, key, requestHash, resourceID, expiresAt)
	return mapError(err)
}

func (s *Store) AppendAudit(ctx context.Context, item domain.AuditEntry) error {
	payload, err := marshal(item)
	if err != nil {
		return err
	}
	_, err = s.query(ctx).Exec(ctx, `INSERT INTO audit_entries (id,resource,resource_id,occurred_at,document) VALUES ($1,$2,$3,$4,$5)`, item.ID, item.Resource, item.ResourceID, item.OccurredAt, payload)
	return mapError(err)
}

func (s *Store) ListAudit(ctx context.Context, resource, resourceID string) ([]domain.AuditEntry, error) {
	rows, err := s.query(ctx).Query(ctx, `SELECT document FROM audit_entries WHERE ($1='' OR resource=$1) AND ($2='' OR resource_id=$2) ORDER BY occurred_at`, resource, resourceID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	result := []domain.AuditEntry{}
	for rows.Next() {
		var payload []byte
		var item domain.AuditEntry
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(payload, &item); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
