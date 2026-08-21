package application

import (
	"context"
	"fmt"
	"strings"
)

type Repositories interface {
	FeedbackRecords
	FeedbackSearch
	TimelineJournal
	CommunicationRecords
	AttachmentRecords
	MergeRecords
	AnnouncementRecords
	AuditJournal
	TokenRecords
	IdempotencyRecords
	CatalogRecords
	TrendReader
}

type TransactionManager interface {
	WithinTransaction(context.Context, func(context.Context, Repositories) error) error
}

type Services struct {
	Submission    *SubmissionService
	Workflow      *WorkflowService
	Communication *CommunicationService
	Queue         *QueueService
	Merge         *MergeService
	Announcement  *AnnouncementService
	Attachment    *AttachmentService
	Report        *ReportService
	Catalog       *CatalogService
}

type Dependencies struct {
	Transactions  TransactionManager
	Repositories  Repositories
	Clock         Clock
	IDs           IDGenerator
	Tokens        QueryTokenCodec
	Notifications NotificationSink
	Objects       ObjectStore
	Redactor      Redactor
	Duplicates    DuplicateMatcher
}

func NewServices(deps Dependencies) (*Services, error) {
	missing := make([]string, 0)
	if deps.Transactions == nil {
		missing = append(missing, "transactions")
	}
	if deps.Repositories == nil {
		missing = append(missing, "repositories")
	}
	if deps.Clock == nil {
		missing = append(missing, "clock")
	}
	if deps.IDs == nil {
		missing = append(missing, "ids")
	}
	if deps.Tokens == nil {
		missing = append(missing, "tokens")
	}
	if deps.Notifications == nil {
		missing = append(missing, "notifications")
	}
	if deps.Objects == nil {
		missing = append(missing, "objects")
	}
	if deps.Redactor == nil {
		missing = append(missing, "redactor")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing application dependencies: %s", strings.Join(missing, ", "))
	}
	return &Services{
		Submission:    NewSubmissionService(deps),
		Workflow:      NewWorkflowService(deps),
		Communication: NewCommunicationService(deps),
		Queue:         NewQueueService(deps),
		Merge:         NewMergeService(deps),
		Announcement:  NewAnnouncementService(deps),
		Attachment:    NewAttachmentService(deps),
		Report:        NewReportService(deps),
		Catalog:       NewCatalogService(deps),
	}, nil
}
