package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type DocumentEditHandler struct {
	service interfaces.DocumentEditService
}

func NewDocumentEditHandler(service interfaces.DocumentEditService) *DocumentEditHandler {
	return &DocumentEditHandler{service: service}
}

func documentEditContext(c *gin.Context) (string, uint64, bool) { return favoriteContext(c) }

func documentEditError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrDocumentEditNotFound):
		c.Error(apperrors.NewNotFoundError(err.Error()))
	case errors.Is(err, service.ErrDocumentEditDisabled), errors.Is(err, service.ErrDocumentEditInvalid):
		c.Error(apperrors.NewBadRequestError(err.Error()))
	default:
		c.Error(apperrors.NewInternalServerError(err.Error()))
	}
}

func (h *DocumentEditHandler) Capabilities(c *gin.Context) {
	capabilities, err := h.service.Capabilities(c.Request.Context())
	if err != nil {
		documentEditError(c, err)
		return
	}
	health, err := h.service.Health(c.Request.Context())
	if err != nil {
		documentEditError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"capabilities": capabilities, "health": health}})
}

func (h *DocumentEditHandler) List(c *gin.Context) {
	userID, tenantID, ok := documentEditContext(c)
	if !ok {
		return
	}
	rows, err := h.service.List(c.Request.Context(), tenantID, userID)
	if err != nil {
		documentEditError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rows})
}

func (h *DocumentEditHandler) Get(c *gin.Context) {
	userID, tenantID, ok := documentEditContext(c)
	if !ok {
		return
	}
	job, err := h.service.Get(c.Request.Context(), tenantID, userID, c.Param("id"))
	if err != nil {
		documentEditError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": job})
}

func (h *DocumentEditHandler) Debug(c *gin.Context) {
	userID, tenantID, ok := documentEditContext(c)
	if !ok {
		return
	}
	debug, err := h.service.Debug(c.Request.Context(), tenantID, userID, c.Param("id"))
	if err != nil {
		documentEditError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": debug})
}

func (h *DocumentEditHandler) DebugBlob(c *gin.Context) {
	userID, tenantID, ok := documentEditContext(c)
	if !ok {
		return
	}
	blob, file, err := h.service.OpenDebugBlob(c.Request.Context(), tenantID, userID, c.Param("id"), c.Param("stageId"), c.Param("kind"))
	if err != nil {
		documentEditError(c, err)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		documentEditError(c, err)
		return
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(data)); !strings.EqualFold(got, blob.SHA256) {
		c.Error(apperrors.NewInternalServerError("debug blob integrity check failed"))
		return
	}
	c.Header("Content-Type", blob.ContentType)
	c.Header("Cache-Control", "private, no-store")
	c.Data(http.StatusOK, blob.ContentType, data)
}

func (h *DocumentEditHandler) Compare(c *gin.Context) {
	userID, tenantID, ok := documentEditContext(c)
	if !ok {
		return
	}
	var request types.DocumentEditComparisonRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid comparison request"))
		return
	}
	comparison, err := h.service.Compare(c.Request.Context(), tenantID, userID, c.Param("id"), request)
	if err != nil {
		documentEditError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": comparison})
}

func (h *DocumentEditHandler) Comparison(c *gin.Context) {
	userID, tenantID, ok := documentEditContext(c)
	if !ok {
		return
	}
	comparison, err := h.service.Comparison(c.Request.Context(), tenantID, userID, c.Param("id"))
	if err != nil {
		documentEditError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": comparison})
}

func (h *DocumentEditHandler) Cancel(c *gin.Context) {
	userID, tenantID, ok := documentEditContext(c)
	if !ok {
		return
	}
	if err := h.service.Cancel(c.Request.Context(), tenantID, userID, c.Param("id")); err != nil {
		documentEditError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *DocumentEditHandler) Create(c *gin.Context) {
	userID, tenantID, ok := documentEditContext(c)
	if !ok {
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.Error(apperrors.NewBadRequestError("DOCX file is required"))
		return
	}
	opened, err := file.Open()
	if err != nil {
		documentEditError(c, err)
		return
	}
	defer opened.Close()
	request := types.DocumentEditCreateRequest{
		FileName:    file.Filename,
		MimeType:    file.Header.Get("Content-Type"),
		Mode:        types.DocumentEditMode(strings.ToLower(strings.TrimSpace(c.PostForm("mode")))),
		Instruction: strings.TrimSpace(c.PostForm("instruction")),
		ModelID:     strings.TrimSpace(c.PostForm("model_id")),
		PlanJSON:    strings.TrimSpace(c.PostForm("edit_plan")),
	}
	job, err := h.service.Create(c.Request.Context(), tenantID, userID, request, opened)
	if err != nil {
		documentEditError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": job})
}

func (h *DocumentEditHandler) Artifact(c *gin.Context) {
	userID, tenantID, ok := documentEditContext(c)
	if !ok {
		return
	}
	artifact, file, err := h.service.OpenArtifact(c.Request.Context(), tenantID, userID, c.Param("id"), c.Param("kind"))
	if err != nil {
		documentEditError(c, err)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		documentEditError(c, err)
		return
	}
	if artifact.SHA256 != "" {
		got := fmt.Sprintf("%x", sha256.Sum256(data))
		if !strings.EqualFold(got, artifact.SHA256) {
			c.Error(apperrors.NewInternalServerError("artifact integrity check failed"))
			return
		}
	}
	c.Header("Content-Type", artifact.MimeType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", strings.ReplaceAll(artifact.FileName, "\"", "")))
	http.ServeContent(c.Writer, c.Request, artifact.FileName, artifact.CreatedAt, bytes.NewReader(data))
}

func (h *DocumentEditHandler) Events(c *gin.Context) {
	userID, tenantID, ok := documentEditContext(c)
	if !ok {
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	eventID := 0
	lastHash := ""
	err := h.service.Events(c.Request.Context(), tenantID, userID, c.Param("id"), func(job *types.DocumentEditJob) error {
		data, err := json.Marshal(job)
		if err != nil {
			return err
		}
		hash := fmt.Sprintf("%x", sha256.Sum256(data))
		if hash == lastHash {
			return nil
		}
		lastHash = hash
		eventID++
		_, _ = fmt.Fprintf(c.Writer, "id: %s\nevent: snapshot\ndata: %s\n\n", strconv.Itoa(eventID), data)
		c.Writer.Flush()
		return nil
	})
	if err != nil && !errors.Is(err, context.Canceled) && c.Request.Context().Err() == nil {
		// The request may close while the browser is navigating away; that is
		// not an application error worth emitting after headers are sent.
		return
	}
}
