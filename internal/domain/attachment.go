package domain

import (
	"path/filepath"
	"strings"
	"time"
)

const MaxAttachmentSize int64 = 10 << 20

var allowedAttachmentTypes = map[string]struct{}{
	"image/jpeg":      {},
	"image/png":       {},
	"application/pdf": {},
}

type Attachment struct {
	ID          string
	FeedbackID  string
	FileName    string
	ContentType string
	Size        int64
	StorageKey  string
	SHA256      string
	UploadedBy  string
	CreatedAt   time.Time
}

func (a Attachment) Validate() error {
	if strings.TrimSpace(a.ID) == "" || strings.TrimSpace(a.FeedbackID) == "" {
		return ValidationError{Field: "attachment", Message: "attachment and feedback ids are required"}
	}
	if a.Size <= 0 || a.Size > MaxAttachmentSize {
		return ValidationError{Field: "size", Message: "attachment size is outside allowed range"}
	}
	if _, ok := allowedAttachmentTypes[strings.ToLower(a.ContentType)]; !ok {
		return ValidationError{Field: "content_type", Message: "attachment type is not allowed"}
	}
	base := filepath.Base(a.FileName)
	if base == "." || base != a.FileName || strings.TrimSpace(base) == "" {
		return ValidationError{Field: "file_name", Message: "attachment file name is unsafe"}
	}
	if len(a.SHA256) != 64 {
		return ValidationError{Field: "sha256", Message: "attachment digest is invalid"}
	}
	return nil
}

func AttachmentTypeAllowed(contentType string) bool {
	_, ok := allowedAttachmentTypes[strings.ToLower(strings.TrimSpace(contentType))]
	return ok
}
