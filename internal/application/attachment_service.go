package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"

	"github.com/wyw14/cry-071/internal/domain"
)

type ObjectStore interface {
	Put(context.Context, string, io.Reader, int64) (string, string, error)
	Open(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
}

type AttachmentRecords interface {
	CreateAttachment(context.Context, domain.Attachment) error
	GetAttachment(context.Context, string) (*domain.Attachment, error)
	ListAttachments(context.Context, string) ([]domain.Attachment, error)
}

type AttachmentService struct{ deps Dependencies }

func NewAttachmentService(deps Dependencies) *AttachmentService {
	return &AttachmentService{deps: deps}
}

type UploadAttachmentCommand struct {
	FeedbackID  string
	FileName    string
	ContentType string
	Size        int64
	Reader      io.Reader
	Actor       domain.Actor
	RequestID   string
}

func (s *AttachmentService) Upload(ctx context.Context, cmd UploadAttachmentCommand) (*domain.Attachment, error) {
	feedback, err := s.deps.Repositories.Get(ctx, cmd.FeedbackID)
	if err != nil {
		return nil, err
	}
	if cmd.Actor.Role != domain.RoleSubmitter && !cmd.Actor.CanManage(feedback.AreaID) {
		return nil, domain.ErrForbidden
	}
	if cmd.Size <= 0 || cmd.Size > domain.MaxAttachmentSize {
		return nil, domain.ValidationError{Field: "size", Message: "attachment size is outside allowed range"}
	}
	if !domain.AttachmentTypeAllowed(cmd.ContentType) {
		return nil, domain.ValidationError{Field: "content_type", Message: "attachment type is not allowed"}
	}
	attachmentID := s.deps.IDs.New("attachment")
	storageKey := feedback.ID + "/" + attachmentID
	hasher := sha256.New()
	reader := io.TeeReader(io.LimitReader(cmd.Reader, cmd.Size+1), hasher)
	storedKey, digest, err := s.deps.Objects.Put(ctx, storageKey, reader, cmd.Size)
	if err != nil {
		return nil, err
	}
	computed := hex.EncodeToString(hasher.Sum(nil))
	if digest != "" && !strings.EqualFold(digest, computed) {
		_ = s.deps.Objects.Delete(ctx, storedKey)
		return nil, domain.ConflictError{Resource: "attachment digest", Key: attachmentID}
	}
	now := s.deps.Clock.Now()
	attachment := &domain.Attachment{ID: attachmentID, FeedbackID: feedback.ID, FileName: cmd.FileName,
		ContentType: cmd.ContentType, Size: cmd.Size, StorageKey: storedKey, SHA256: computed,
		UploadedBy: cmd.Actor.ID, CreatedAt: now}
	if err := attachment.Validate(); err != nil {
		_ = s.deps.Objects.Delete(ctx, storedKey)
		return nil, err
	}
	err = s.deps.Transactions.WithinTransaction(ctx, func(tx context.Context, repos Repositories) error {
		if err := repos.CreateAttachment(tx, *attachment); err != nil {
			return err
		}
		return repos.AppendAudit(tx, newAudit(s.deps.IDs, cmd.Actor.ID, "attachment.upload", "attachment",
			attachment.ID, cmd.RequestID, map[string]string{"feedback_id": feedback.ID}, now))
	})
	if err != nil {
		_ = s.deps.Objects.Delete(ctx, storedKey)
		return nil, err
	}
	return attachment, nil
}

func (s *AttachmentService) Open(ctx context.Context, attachmentID string, actor domain.Actor, token string) (*domain.Attachment, io.ReadCloser, error) {
	attachment, err := s.deps.Repositories.GetAttachment(ctx, strings.TrimSpace(attachmentID))
	if err != nil {
		return nil, nil, err
	}
	feedback, err := s.deps.Repositories.Get(ctx, attachment.FeedbackID)
	if err != nil {
		return nil, nil, err
	}
	allowed := actor.CanManage(feedback.AreaID) || actor.CanAudit()
	if !allowed && strings.TrimSpace(token) != "" {
		feedbackID, tokenErr := s.deps.Repositories.ResolveToken(ctx, s.deps.Tokens.Digest(token))
		allowed = tokenErr == nil && feedbackID == feedback.ID
	}
	if !allowed {
		return nil, nil, domain.ErrForbidden
	}
	reader, err := s.deps.Objects.Open(ctx, attachment.StorageKey)
	if err != nil {
		return nil, nil, err
	}
	return attachment, reader, nil
}
