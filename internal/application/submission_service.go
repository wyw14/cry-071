package application

import (
	"context"
	"strings"
	"time"

	"github.com/wyw14/cry-071/internal/domain"
)

type QueryTokenCodec interface {
	Issue(string, time.Time) (plain string, digest string, err error)
	Digest(string) string
}

type Redactor interface {
	Text(string) string
	Contact(string) string
}

type AuditJournal interface {
	AppendAudit(context.Context, domain.AuditEntry) error
	ListAudit(context.Context, string, string) ([]domain.AuditEntry, error)
}

type TokenRecords interface {
	SaveToken(context.Context, string, string, time.Time) error
	ResolveToken(context.Context, string) (string, error)
}

type IdempotencyRecords interface {
	GetIdempotency(context.Context, string, string, string) (string, bool, error)
	SaveIdempotency(context.Context, string, string, string, string, time.Time) error
}

type SubmitFeedbackCommand struct {
	AreaID             string `validate:"required"`
	FacilityCategoryID string `validate:"required"`
	SubjectCode        string `validate:"required"`
	Priority           domain.Priority
	Title              string `validate:"required,min=5,max=120"`
	Description        string `validate:"required,min=10,max=4000"`
	Location           string `validate:"required,max=300"`
	Anonymous          bool
	SubmitterID        string
	SubmitterName      string
	SubmitterContact   string
	IdempotencyKey     string
	RequestID          string
}

type SubmissionResult struct {
	Feedback            *domain.Feedback     `json:"feedback"`
	QueryToken          string               `json:"query_token"`
	DuplicateCandidates []DuplicateCandidate `json:"duplicate_candidates"`
}

type SubmissionService struct{ deps Dependencies }

func NewSubmissionService(deps Dependencies) *SubmissionService {
	return &SubmissionService{deps: deps}
}

func (s *SubmissionService) Submit(ctx context.Context, cmd SubmitFeedbackCommand) (SubmissionResult, error) {
	now := s.deps.Clock.Now()
	idempotencyKey := strings.TrimSpace(cmd.IdempotencyKey)
	hash := requestHash(struct {
		AreaID, FacilityCategoryID, SubjectCode      string
		Priority                                     domain.Priority
		Title, Description, Location                 string
		Anonymous                                    bool
		SubmitterID, SubmitterName, SubmitterContact string
	}{cmd.AreaID, cmd.FacilityCategoryID, cmd.SubjectCode, cmd.Priority, cmd.Title, cmd.Description, cmd.Location, cmd.Anonymous, cmd.SubmitterID, cmd.SubmitterName, cmd.SubmitterContact})
	if idempotencyKey != "" {
		resourceID, found, err := s.deps.Repositories.GetIdempotency(ctx, "feedback.submit", idempotencyKey, hash)
		if err != nil {
			return SubmissionResult{}, err
		}
		if found {
			return s.replaySubmission(ctx, resourceID, now)
		}
	}
	feedbackID := s.deps.IDs.New("feedback")
	numberID := strings.TrimPrefix(s.deps.IDs.New("no"), "no-")
	area, err := s.deps.Repositories.GetArea(ctx, strings.TrimSpace(cmd.AreaID))
	if err != nil {
		return SubmissionResult{}, err
	}
	if !area.Active {
		return SubmissionResult{}, domain.ValidationError{Field: "area_id", Message: "area is not accepting feedback"}
	}
	category, err := s.deps.Repositories.GetCategory(ctx, strings.TrimSpace(cmd.FacilityCategoryID))
	if err != nil {
		return SubmissionResult{}, err
	}
	if !category.Active {
		return SubmissionResult{}, domain.ValidationError{Field: "facility_category_id", Message: "category is inactive"}
	}
	subject, err := s.deps.Repositories.GetSubject(ctx, strings.TrimSpace(cmd.SubjectCode))
	if err != nil {
		return SubmissionResult{}, err
	}
	priority := cmd.Priority
	if !priority.Valid() {
		priority = subject.DefaultPriority
	}
	feedback, err := domain.CreateFeedback(domain.NewFeedback{
		ID: feedbackID, AcceptanceNumber: acceptanceNumber(now, numberID),
		AreaID: area.ID, FacilityCategoryID: category.ID, SubjectCode: subject.Code,
		Priority: priority, Title: cmd.Title, Description: cmd.Description, Location: cmd.Location,
		Anonymous: cmd.Anonymous, SubmitterID: cmd.SubmitterID, SubmitterName: cmd.SubmitterName,
		SubmitterContact: cmd.SubmitterContact, CreatedAt: now,
	})
	if err != nil {
		return SubmissionResult{}, err
	}
	plainToken, tokenDigest, err := s.deps.Tokens.Issue(feedback.ID, now.Add(180*24*time.Hour))
	if err != nil {
		return SubmissionResult{}, err
	}
	event := newTimelineEvent(s.deps.IDs, feedback, domain.EventSubmitted, feedback.SubmitterID,
		domain.VisibilityPublic, "反馈已提交，等待受理", map[string]string{
			"acceptance_number": feedback.AcceptanceNumber,
			"priority":          string(feedback.Priority),
		}, now)
	audit := newAudit(s.deps.IDs, feedback.SubmitterID, "feedback.submit", "feedback", feedback.ID,
		cmd.RequestID, map[string]string{"area_id": feedback.AreaID, "anonymous": boolText(feedback.Anonymous)}, now)
	if err := s.deps.Transactions.WithinTransaction(ctx, func(tx context.Context, repos Repositories) error {
		if err := repos.Create(tx, feedback); err != nil {
			return err
		}
		if err := repos.Append(tx, event); err != nil {
			return err
		}
		if err := repos.SaveToken(tx, tokenDigest, feedback.ID, now.Add(180*24*time.Hour)); err != nil {
			return err
		}
		if idempotencyKey != "" {
			if err := repos.SaveIdempotency(tx, "feedback.submit", idempotencyKey, hash, feedback.ID, now.Add(24*time.Hour)); err != nil {
				return err
			}
		}
		return repos.AppendAudit(tx, audit)
	}); err != nil {
		return SubmissionResult{}, err
	}
	if feedback.SubmitterContact != "" {
		_ = s.deps.Notifications.Send(ctx, Notification{
			Recipient: feedback.SubmitterContact, Template: "feedback_received",
			Data: map[string]string{"acceptance_number": feedback.AcceptanceNumber}, CreatedAt: now,
		})
	}
	candidates := []DuplicateCandidate{}
	if s.deps.Duplicates != nil {
		candidates, _ = s.deps.Duplicates.Find(ctx, feedback, 5)
	}
	return SubmissionResult{Feedback: feedback.Clone(), QueryToken: plainToken, DuplicateCandidates: candidates}, nil
}

func (s *SubmissionService) replaySubmission(ctx context.Context, feedbackID string, now time.Time) (SubmissionResult, error) {
	feedback, err := s.deps.Repositories.Get(ctx, feedbackID)
	if err != nil {
		return SubmissionResult{}, err
	}
	plain, digest, err := s.deps.Tokens.Issue(feedback.ID, now.Add(180*24*time.Hour))
	if err != nil {
		return SubmissionResult{}, err
	}
	if err := s.deps.Transactions.WithinTransaction(ctx, func(tx context.Context, repos Repositories) error {
		return repos.SaveToken(tx, digest, feedback.ID, now.Add(180*24*time.Hour))
	}); err != nil {
		return SubmissionResult{}, err
	}
	candidates := []DuplicateCandidate{}
	if s.deps.Duplicates != nil {
		candidates, _ = s.deps.Duplicates.Find(ctx, feedback, 5)
	}
	return SubmissionResult{Feedback: feedback.Clone(), QueryToken: plain, DuplicateCandidates: candidates}, nil
}

func boolText(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

type PublicFeedbackView struct {
	Feedback     *domain.Feedback       `json:"feedback"`
	Timeline     []domain.TimelineEvent `json:"timeline"`
	Replies      []domain.Reply         `json:"replies"`
	Attachments  []domain.Attachment    `json:"attachments"`
	Satisfaction *domain.Satisfaction   `json:"satisfaction,omitempty"`
}

func (s *SubmissionService) ViewByToken(ctx context.Context, plainToken string) (PublicFeedbackView, error) {
	readContext := context.Background()
	feedback, err := s.resolvePublicFeedback(readContext, plainToken)
	if err != nil {
		return PublicFeedbackView{}, err
	}
	materials, err := s.loadPublicMaterials(readContext, feedback.ID)
	if err != nil {
		return PublicFeedbackView{}, err
	}
	return s.buildPublicView(feedback, materials), nil
}

type publicFeedbackMaterials struct {
	timeline     []domain.TimelineEvent
	replies      []domain.Reply
	attachments  []domain.Attachment
	satisfaction *domain.Satisfaction
}

func (s *SubmissionService) resolvePublicFeedback(ctx context.Context, plainToken string) (*domain.Feedback, error) {
	digest := s.deps.Tokens.Digest(strings.TrimSpace(plainToken))
	feedbackID, err := s.deps.Repositories.ResolveToken(ctx, digest)
	if err != nil {
		return nil, err
	}
	feedback, err := s.deps.Repositories.Get(ctx, feedbackID)
	if err != nil {
		return nil, err
	}
	return feedback, nil
}

func (s *SubmissionService) loadPublicMaterials(ctx context.Context, feedbackID string) (publicFeedbackMaterials, error) {
	result := publicFeedbackMaterials{}
	var err error
	result.timeline, err = s.deps.Repositories.ListTimeline(ctx, feedbackID)
	if err != nil {
		return publicFeedbackMaterials{}, err
	}
	result.replies, err = s.deps.Repositories.ListReplies(ctx, feedbackID)
	if err != nil {
		return publicFeedbackMaterials{}, err
	}
	result.attachments, err = s.deps.Repositories.ListAttachments(ctx, feedbackID)
	if err != nil {
		return publicFeedbackMaterials{}, err
	}
	result.satisfaction, err = s.deps.Repositories.GetSatisfaction(ctx, feedbackID)
	if err == domain.ErrNotFound {
		result.satisfaction = nil
	} else if err != nil {
		return publicFeedbackMaterials{}, err
	}
	return result, nil
}

func (s *SubmissionService) buildPublicView(feedback *domain.Feedback, materials publicFeedbackMaterials) PublicFeedbackView {
	redacted := feedback.Clone()
	redacted.Description = s.deps.Redactor.Text(redacted.Description)
	redacted.SubmitterContact = s.deps.Redactor.Contact(redacted.SubmitterContact)
	if redacted.Anonymous {
		redacted.SubmitterID = ""
		redacted.SubmitterName = "匿名提交者"
	}
	publicReplies := make([]domain.Reply, 0, len(materials.replies))
	for _, reply := range materials.replies {
		if reply.Public {
			reply.Content = s.deps.Redactor.Text(reply.Content)
			publicReplies = append(publicReplies, reply)
		}
	}
	return PublicFeedbackView{
		Feedback:     redacted,
		Timeline:     domain.PublicTimeline(materials.timeline),
		Replies:      publicReplies,
		Attachments:  append([]domain.Attachment(nil), materials.attachments...),
		Satisfaction: materials.satisfaction,
	}
}
