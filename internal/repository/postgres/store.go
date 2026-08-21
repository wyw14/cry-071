package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wyw14/cry-071/internal/application"
	"github.com/wyw14/cry-071/internal/domain"
)

type txContextKey struct{}

type querier interface {
	Exec(context.Context, string, ...any) (pgconnCommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type pgconnCommandTag interface {
	RowsAffected() int64
}

type Store struct{ pool *pgxpool.Pool }

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	config.MaxConns = 10
	config.MinConns = 1
	config.MaxConnLifetime = 30 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("open postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

func (s *Store) query(ctx context.Context) querier {
	if tx, ok := ctx.Value(txContextKey{}).(pgx.Tx); ok {
		return txAdapter{tx}
	}
	return poolAdapter{s.pool}
}

type poolAdapter struct{ pool *pgxpool.Pool }

func (p poolAdapter) Exec(ctx context.Context, sql string, args ...any) (pgconnCommandTag, error) {
	return p.pool.Exec(ctx, sql, args...)
}
func (p poolAdapter) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return p.pool.Query(ctx, sql, args...)
}
func (p poolAdapter) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return p.pool.QueryRow(ctx, sql, args...)
}

type txAdapter struct{ tx pgx.Tx }

func (t txAdapter) Exec(ctx context.Context, sql string, args ...any) (pgconnCommandTag, error) {
	return t.tx.Exec(ctx, sql, args...)
}
func (t txAdapter) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return t.tx.Query(ctx, sql, args...)
}
func (t txAdapter) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return t.tx.QueryRow(ctx, sql, args...)
}

func (s *Store) WithinTransaction(ctx context.Context, operation func(context.Context, application.Repositories) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(context.Background())
		}
	}()
	txContext := context.WithValue(ctx, txContextKey{}, tx)
	if err := operation(txContext, s); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return mapError(err)
	}
	committed = true
	return nil
}

func (s *Store) Create(ctx context.Context, feedback *domain.Feedback) error {
	payload, err := json.Marshal(feedback)
	if err != nil {
		return err
	}
	_, err = s.query(ctx).Exec(ctx, `
		INSERT INTO feedbacks (id, acceptance_number, area_id, subject_code, priority, status,
			assignee_id, version, created_at, updated_at, due_at, document)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, feedback.ID, feedback.AcceptanceNumber,
		feedback.AreaID, feedback.SubjectCode, feedback.Priority, feedback.Status, feedback.AssigneeID,
		feedback.Version, feedback.CreatedAt, feedback.UpdatedAt, feedback.DueAt, payload)
	return mapError(err)
}

func (s *Store) Get(ctx context.Context, id string) (*domain.Feedback, error) {
	var payload []byte
	err := s.query(ctx).QueryRow(ctx, `SELECT document FROM feedbacks WHERE id=$1`, id).Scan(&payload)
	if err != nil {
		return nil, mapError(err)
	}
	var feedback domain.Feedback
	if err := json.Unmarshal(payload, &feedback); err != nil {
		return nil, fmt.Errorf("decode feedback: %w", err)
	}
	return feedback.Clone(), nil
}

func (s *Store) GetByAcceptanceNumber(ctx context.Context, number string) (*domain.Feedback, error) {
	var payload []byte
	err := s.query(ctx).QueryRow(ctx, `SELECT document FROM feedbacks WHERE acceptance_number=$1`, number).Scan(&payload)
	if err != nil {
		return nil, mapError(err)
	}
	var feedback domain.Feedback
	if err := json.Unmarshal(payload, &feedback); err != nil {
		return nil, fmt.Errorf("decode feedback: %w", err)
	}
	return feedback.Clone(), nil
}

func (s *Store) Update(ctx context.Context, feedback *domain.Feedback, expectedVersion int64) error {
	payload, err := json.Marshal(feedback)
	if err != nil {
		return err
	}
	tag, err := s.query(ctx).Exec(ctx, `
		UPDATE feedbacks SET area_id=$2, subject_code=$3, priority=$4, status=$5, assignee_id=$6,
			version=$7, updated_at=$8, due_at=$9, document=$10
		WHERE id=$1 AND version=$11`, feedback.ID, feedback.AreaID, feedback.SubjectCode,
		feedback.Priority, feedback.Status, feedback.AssigneeID, feedback.Version, feedback.UpdatedAt,
		feedback.DueAt, payload, expectedVersion)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() != 1 {
		return domain.ErrVersionConflict
	}
	return nil
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		switch pgErr.SQLState() {
		case "23505":
			return domain.ErrConflict
		case "40001", "40P01":
			return domain.ErrVersionConflict
		}
	}
	return err
}
