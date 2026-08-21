package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry-071/internal/application"
	"github.com/wyw14/cry-071/internal/domain"
	"github.com/wyw14/cry-071/internal/middleware"
)

type batchAssignRequest struct {
	FeedbackIDs map[string]int64 `json:"feedback_ids" binding:"required"`
	AssigneeID  string           `json:"assignee_id" binding:"required"`
}

func (h *handlers) batchAssign(c *gin.Context) {
	var request batchAssignRequest
	if !bindJSON(c, &request) {
		return
	}
	actor, _ := middleware.CurrentActor(c)
	result, err := h.services.Queue.BatchAssign(c.Request.Context(), application.BatchAssignCommand{FeedbackIDs: request.FeedbackIDs, AssigneeID: request.AssigneeID, Actor: actor, RequestID: middleware.GetRequestID(c)})
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, result)
}

type mergeRequest struct {
	PrimaryID string           `json:"primary_id" binding:"required"`
	Members   map[string]int64 `json:"members" binding:"required"`
	Reason    string           `json:"reason" binding:"required,min=5"`
}

func (h *handlers) mergeFeedback(c *gin.Context) {
	var request mergeRequest
	if !bindJSON(c, &request) {
		return
	}
	actor, _ := middleware.CurrentActor(c)
	result, err := h.services.Merge.Merge(c.Request.Context(), application.MergeCommand{PrimaryID: request.PrimaryID, MemberVersions: request.Members, Reason: request.Reason, Actor: actor, RequestID: middleware.GetRequestID(c)})
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusCreated, result)
}

type relationRequest struct {
	FirstID       string `json:"first_id" binding:"required"`
	FirstVersion  int64  `json:"first_version" binding:"required,min=1"`
	SecondID      string `json:"second_id" binding:"required"`
	SecondVersion int64  `json:"second_version" binding:"required,min=1"`
}

func (h *handlers) relateFeedback(c *gin.Context) {
	var request relationRequest
	if !bindJSON(c, &request) {
		return
	}
	actor, _ := middleware.CurrentActor(c)
	err := h.services.Merge.Relate(c.Request.Context(), request.FirstID, request.FirstVersion, request.SecondID, request.SecondVersion, actor, middleware.GetRequestID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusNoContent, nil)
}

type announcementRequest struct {
	Title       string   `json:"title" binding:"required,min=4"`
	Content     string   `json:"content" binding:"required,min=10"`
	AreaID      string   `json:"area_id" binding:"required"`
	FeedbackIDs []string `json:"feedback_ids" binding:"required,min=1"`
}

func (h *handlers) createAnnouncement(c *gin.Context) {
	var request announcementRequest
	if !bindJSON(c, &request) {
		return
	}
	actor, _ := middleware.CurrentActor(c)
	result, err := h.services.Announcement.Create(c.Request.Context(), application.CreateAnnouncementCommand{Title: request.Title, Content: request.Content, AreaID: request.AreaID, FeedbackIDs: request.FeedbackIDs, Actor: actor, RequestID: middleware.GetRequestID(c)})
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusCreated, result)
}
func (h *handlers) publishAnnouncement(c *gin.Context) {
	actor, _ := middleware.CurrentActor(c)
	result, err := h.services.Announcement.Publish(c.Request.Context(), c.Param("id"), actor, middleware.GetRequestID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, result)
}
func (h *handlers) listAnnouncements(c *gin.Context) {
	result, err := h.services.Announcement.ListPublic(c.Request.Context(), c.Query("area_id"))
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, result)
}

func (h *handlers) trends(c *gin.Context) {
	actor, _ := middleware.CurrentActor(c)
	from, err := time.Parse(time.RFC3339, c.Query("from"))
	if err != nil {
		respondError(c, applicationError("from"))
		return
	}
	to, err := time.Parse(time.RFC3339, c.Query("to"))
	if err != nil {
		respondError(c, applicationError("to"))
		return
	}
	result, err := h.services.Report.Trend(c.Request.Context(), application.TrendQuery{AreaID: c.Query("area_id"), From: from, To: to, Bucket: c.Query("bucket"), Actor: actor})
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, result)
}
func (h *handlers) quality(c *gin.Context) {
	actor, _ := middleware.CurrentActor(c)
	result, err := h.services.Report.Quality(c.Request.Context(), c.Query("area_id"), actor)
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, result)
}

func applicationError(field string) error {
	return domain.ValidationError{Field: field, Message: "时间必须使用 RFC3339 格式"}
}
