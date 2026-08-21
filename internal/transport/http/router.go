package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry-071/internal/application"
	"github.com/wyw14/cry-071/internal/config"
	"github.com/wyw14/cry-071/internal/domain"
	"github.com/wyw14/cry-071/internal/middleware"
	"go.uber.org/zap"
)

type Readiness interface{ Ping(context.Context) error }

type RouterDependencies struct {
	Services *application.Services
	Config   config.Config
	Logger   *zap.Logger
	Ready    func(context.Context) error
}

func NewRouter(deps RouterDependencies) *gin.Engine {
	if deps.Config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.MaxMultipartMemory = 12 << 20
	router.Use(middleware.RequestID(), middleware.SecurityHeaders(), middleware.CORS(deps.Config.AllowedOrigins),
		middleware.Actor(), middleware.Timeout(deps.Config.RequestTimeout), middleware.Logger(deps.Logger), middleware.Recovery(deps.Logger))
	router.GET("/healthz", func(c *gin.Context) { respond(c, http.StatusOK, gin.H{"status": "ok", "time": time.Now().UTC()}) })
	router.GET("/readyz", func(c *gin.Context) {
		if deps.Ready != nil {
			if err := deps.Ready(c.Request.Context()); err != nil {
				respondError(c, err)
				return
			}
		}
		respond(c, http.StatusOK, gin.H{"status": "ready"})
	})
	handlers := newHandlers(deps.Services)
	api := router.Group("/api/v1")
	api.POST("/feedback", handlers.submitFeedback)
	api.GET("/feedback/query/:token", handlers.viewFeedback)
	api.POST("/feedback/supplements", handlers.addSupplement)
	api.POST("/feedback/satisfaction", handlers.confirmSatisfaction)
	api.GET("/announcements", handlers.listAnnouncements)
	api.GET("/metadata/areas", handlers.listAreas)
	api.GET("/metadata/project", handlers.projectMetadata)
	api.GET("/attachments/:id", handlers.downloadAttachment)

	staff := api.Group("")
	staff.Use(middleware.RequireRoles(domain.RoleAgent, domain.RoleManager, domain.RoleAuditor))
	staff.POST("/feedback/:id/transitions", handlers.transitionFeedback)
	staff.POST("/feedback/:id/assignments", handlers.assignFeedback)
	staff.POST("/feedback/:id/replies", handlers.addReply)
	staff.POST("/feedback/:id/attachments", handlers.uploadAttachment)
	staff.GET("/admin/queue", handlers.listQueue)
	staff.GET("/reports/trends", handlers.trends)
	staff.GET("/reports/quality", handlers.quality)

	managers := api.Group("")
	managers.Use(middleware.RequireRoles(domain.RoleManager))
	managers.POST("/admin/batch/assign", handlers.batchAssign)
	managers.POST("/merges", handlers.mergeFeedback)
	managers.POST("/feedback/relations", handlers.relateFeedback)
	managers.POST("/announcements", handlers.createAnnouncement)
	managers.POST("/announcements/:id/publish", handlers.publishAnnouncement)
	return router
}
