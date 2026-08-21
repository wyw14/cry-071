package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry-071/internal/application"
	"github.com/wyw14/cry-071/internal/domain"
	"github.com/wyw14/cry-071/internal/middleware"
)

type handlers struct{ services *application.Services }

func newHandlers(services *application.Services) *handlers { return &handlers{services: services} }

type submitRequest struct {
	AreaID             string          `json:"area_id" binding:"required"`
	FacilityCategoryID string          `json:"facility_category_id" binding:"required"`
	SubjectCode        string          `json:"subject_code" binding:"required"`
	Priority           domain.Priority `json:"priority"`
	Title              string          `json:"title" binding:"required,min=5,max=120"`
	Description        string          `json:"description" binding:"required,min=10,max=4000"`
	Location           string          `json:"location" binding:"required,max=300"`
	Anonymous          bool            `json:"anonymous"`
	SubmitterName      string          `json:"submitter_name"`
	SubmitterContact   string          `json:"submitter_contact"`
}

func (h *handlers) submitFeedback(c *gin.Context) {
	var request submitRequest
	if !bindJSON(c, &request) {
		return
	}
	actor, _ := middleware.CurrentActor(c)
	result, err := h.services.Submission.Submit(c.Request.Context(), application.SubmitFeedbackCommand{
		AreaID: request.AreaID, FacilityCategoryID: request.FacilityCategoryID, SubjectCode: request.SubjectCode,
		Priority: request.Priority, Title: request.Title, Description: request.Description, Location: request.Location,
		Anonymous: request.Anonymous, SubmitterID: actor.ID, SubmitterName: request.SubmitterName,
		SubmitterContact: request.SubmitterContact, IdempotencyKey: c.GetHeader("Idempotency-Key"),
		RequestID: middleware.GetRequestID(c),
	})
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusCreated, result)
}

func (h *handlers) viewFeedback(c *gin.Context) {
	result, err := h.services.Submission.ViewByToken(c.Request.Context(), c.Param("token"))
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, result)
}

type transitionRequest struct {
	Version int64                 `json:"version" binding:"required,min=1"`
	To      domain.FeedbackStatus `json:"to" binding:"required"`
	Reason  string                `json:"reason"`
}

func (h *handlers) transitionFeedback(c *gin.Context) {
	var request transitionRequest
	if !bindJSON(c, &request) {
		return
	}
	actor, _ := middleware.CurrentActor(c)
	result, err := h.services.Workflow.Transition(c.Request.Context(), application.TransitionCommand{FeedbackID: c.Param("id"), ExpectedVersion: request.Version, To: request.To, Reason: request.Reason, Actor: actor, RequestID: middleware.GetRequestID(c), IdempotencyKey: c.GetHeader("Idempotency-Key")})
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, result)
}

type assignmentRequest struct {
	Version    int64  `json:"version" binding:"required,min=1"`
	AssigneeID string `json:"assignee_id" binding:"required"`
}

func (h *handlers) assignFeedback(c *gin.Context) {
	var request assignmentRequest
	if !bindJSON(c, &request) {
		return
	}
	actor, _ := middleware.CurrentActor(c)
	result, err := h.services.Workflow.Assign(c.Request.Context(), application.AssignmentCommand{FeedbackID: c.Param("id"), ExpectedVersion: request.Version, AssigneeID: request.AssigneeID, Actor: actor, RequestID: middleware.GetRequestID(c)})
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, result)
}

type replyRequest struct {
	Content string `json:"content" binding:"required,min=2,max=2000"`
	Public  bool   `json:"public"`
}

func (h *handlers) addReply(c *gin.Context) {
	var request replyRequest
	if !bindJSON(c, &request) {
		return
	}
	actor, _ := middleware.CurrentActor(c)
	result, err := h.services.Communication.Reply(c.Request.Context(), application.ReplyCommand{FeedbackID: c.Param("id"), Content: request.Content, Public: request.Public, Actor: actor, RequestID: middleware.GetRequestID(c)})
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusCreated, result)
}

type supplementRequest struct {
	QueryToken string `json:"query_token" binding:"required"`
	Content    string `json:"content" binding:"required,min=2"`
}

func (h *handlers) addSupplement(c *gin.Context) {
	var request supplementRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.services.Communication.Supplement(c.Request.Context(), application.SupplementCommand{QueryToken: request.QueryToken, Content: request.Content, RequestID: middleware.GetRequestID(c)})
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusCreated, result)
}

type satisfactionRequest struct {
	QueryToken string `json:"query_token" binding:"required"`
	Score      int    `json:"score" binding:"required,min=1,max=5"`
	Comment    string `json:"comment" binding:"max=500"`
}

func (h *handlers) confirmSatisfaction(c *gin.Context) {
	var request satisfactionRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.services.Communication.ConfirmSatisfaction(c.Request.Context(), application.SatisfactionCommand{QueryToken: request.QueryToken, Score: request.Score, Comment: request.Comment, RequestID: middleware.GetRequestID(c)})
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, result)
}

func (h *handlers) listQueue(c *gin.Context) {
	actor, _ := middleware.CurrentActor(c)
	page := parseInt(c.Query("page"), 1)
	size := parseInt(c.Query("page_size"), 20)
	result, err := h.services.Queue.List(c.Request.Context(), application.QueueQuery{AreaID: c.Query("area_id"), Status: domain.FeedbackStatus(c.Query("status")), Priority: domain.Priority(c.Query("priority")), AssigneeID: c.Query("assignee_id"), OverdueOnly: c.Query("overdue") == "true", Page: page, PageSize: size, Sort: c.Query("sort"), Descending: c.Query("order") == "desc", Actor: actor})
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, result)
}

func (h *handlers) listAreas(c *gin.Context) {
	result, err := h.services.Catalog.Areas(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, result)
}

func (h *handlers) projectMetadata(c *gin.Context) {
	respond(c, http.StatusOK, h.services.Catalog.Project())
}

func parseInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}
