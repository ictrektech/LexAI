package router

import (
	"github.com/Tencent/WeKnora/internal/handler"
	"github.com/gin-gonic/gin"
)

func RegisterDocumentEditRoutes(r *gin.RouterGroup, h *handler.DocumentEditHandler, g *rbacGuards) {
	r.GET("/document-edits/capabilities", g.Viewer(), h.Capabilities)
	jobs := r.Group("/document-edits")
	{
		jobs.GET("", g.Viewer(), h.List)
		jobs.POST("", g.Viewer(), h.Create)
		jobs.GET("/:id", g.Viewer(), h.Get)
		jobs.GET("/:id/debug", g.Viewer(), h.Debug)
		jobs.GET("/:id/debug/stages/:stageId/blobs/:kind", g.Viewer(), h.DebugBlob)
		jobs.POST("/:id/comparisons", g.Viewer(), h.Compare)
		jobs.GET("/:id/comparison", g.Viewer(), h.Comparison)
		jobs.POST("/:id/cancel", g.Viewer(), h.Cancel)
		jobs.GET("/:id/events", g.Viewer(), h.Events)
		jobs.GET("/:id/artifacts/:kind", g.Viewer(), h.Artifact)
	}
}
