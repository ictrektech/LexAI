package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/application/service"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type contractReviewHandlerStub struct {
	interfaces.ContractReviewService
	startFn  func(context.Context, uint64, string, string) (*types.ContractReview, error)
	getFn    func(context.Context, uint64, string, string) (*types.ContractReview, error)
	uploadFn func(context.Context, uint64, string, string, string, string, int64, io.Reader) (*types.ContractReview, error)
	openFn   func(context.Context, uint64, string, string) (*types.ContractReview, io.ReadCloser, error)
	bulkFn   func(context.Context, uint64, string, []string, types.ContractReviewBulkAction) (*types.ContractReviewBulkResult, error)
}

func (s *contractReviewHandlerStub) Start(ctx context.Context, tenantID uint64, userID, id string) (*types.ContractReview, error) {
	return s.startFn(ctx, tenantID, userID, id)
}

func (s *contractReviewHandlerStub) Get(ctx context.Context, tenantID uint64, userID, id string) (*types.ContractReview, error) {
	return s.getFn(ctx, tenantID, userID, id)
}

func (s *contractReviewHandlerStub) Upload(ctx context.Context, tenantID uint64, userID, id, fileName, mimeType string, fileSize int64, body io.Reader) (*types.ContractReview, error) {
	return s.uploadFn(ctx, tenantID, userID, id, fileName, mimeType, fileSize, body)
}

func (s *contractReviewHandlerStub) OpenDocument(ctx context.Context, tenantID uint64, userID, id string) (*types.ContractReview, io.ReadCloser, error) {
	return s.openFn(ctx, tenantID, userID, id)
}

func (s *contractReviewHandlerStub) BulkAction(ctx context.Context, tenantID uint64, userID string, ids []string, action types.ContractReviewBulkAction) (*types.ContractReviewBulkResult, error) {
	return s.bulkFn(ctx, tenantID, userID, ids, action)
}

type smartArchiveHandlerStub struct {
	interfaces.SmartArchiveService
	deleteFn      func(context.Context, uint64, string) error
	archiveFn     func(context.Context, uint64, string, bool) (*types.ArchiveDocument, error)
	batchActionFn func(context.Context, uint64, []string, types.ArchiveBulkAction) (*types.ArchiveBulkActionResult, error)
}

func (s *smartArchiveHandlerStub) DeleteDocument(ctx context.Context, tenantID uint64, id string) error {
	return s.deleteFn(ctx, tenantID, id)
}

func (s *smartArchiveHandlerStub) ArchiveDocument(ctx context.Context, tenantID uint64, id string, archive bool) (*types.ArchiveDocument, error) {
	return s.archiveFn(ctx, tenantID, id, archive)
}

func (s *smartArchiveHandlerStub) BatchDocumentAction(ctx context.Context, tenantID uint64, ids []string, action types.ArchiveBulkAction) (*types.ArchiveBulkActionResult, error) {
	return s.batchActionFn(ctx, tenantID, ids, action)
}

func legalHandlerTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(func(c *gin.Context) {
		c.Set(types.UserIDContextKey.String(), "user-1")
		c.Set(types.TenantIDContextKey.String(), uint64(7))
		c.Next()
	})
	return r
}

func TestContractReviewHandlerMapsInvalidStateToBadRequest(t *testing.T) {
	h := NewContractReviewHandler(&contractReviewHandlerStub{
		startFn: func(context.Context, uint64, string, string) (*types.ContractReview, error) {
			return nil, service.ErrContractReviewInvalidState
		},
	})
	r := legalHandlerTestRouter()
	r.POST("/contract-reviews/:id/start", h.Start)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/contract-reviews/review-1/start", nil))

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "not in a valid state")
}

func TestContractReviewHandlerMapsNotFoundAndServesPreview(t *testing.T) {
	missing := NewContractReviewHandler(&contractReviewHandlerStub{
		getFn: func(context.Context, uint64, string, string) (*types.ContractReview, error) {
			return nil, service.ErrContractReviewNotFound
		},
	})
	r := legalHandlerTestRouter()
	r.GET("/contract-reviews/:id", missing.Get)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/contract-reviews/missing", nil))
	require.Equal(t, http.StatusNotFound, w.Code)

	review := &types.ContractReview{ID: "review-1", FileName: "agreement.pdf", FileType: ".pdf", MimeType: "application/pdf", UpdatedAt: time.Now()}
	preview := NewContractReviewHandler(&contractReviewHandlerStub{
		openFn: func(context.Context, uint64, string, string) (*types.ContractReview, io.ReadCloser, error) {
			return review, io.NopCloser(bytes.NewReader([]byte("pdf-bytes"))), nil
		},
	})
	r = legalHandlerTestRouter()
	r.GET("/contract-reviews/:id/document/preview", preview.Preview)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/contract-reviews/review-1/document/preview", nil))
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
	require.Equal(t, "pdf-bytes", w.Body.String())
}

func TestContractReviewHandlerRejectsMissingUploadAndPreservesBulkPartialResults(t *testing.T) {
	called := false
	h := NewContractReviewHandler(&contractReviewHandlerStub{
		uploadFn: func(context.Context, uint64, string, string, string, string, int64, io.Reader) (*types.ContractReview, error) {
			called = true
			return &types.ContractReview{ID: "review-1"}, nil
		},
		bulkFn: func(_ context.Context, _ uint64, _ string, ids []string, action types.ContractReviewBulkAction) (*types.ContractReviewBulkResult, error) {
			require.Equal(t, []string{"review-1", "review-2"}, ids)
			require.Equal(t, types.ContractReviewBulkDelete, action)
			return &types.ContractReviewBulkResult{Action: action, Requested: 2, Succeeded: 1, Failed: 1, Items: []types.ContractReviewBulkItem{{ID: "review-1", Success: true}, {ID: "review-2", Error: "missing"}}}, nil
		},
	})
	r := legalHandlerTestRouter()
	r.POST("/contract-reviews/:id/document", h.Upload)
	r.POST("/contract-reviews/bulk/delete", h.BulkAction(types.ContractReviewBulkDelete))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/contract-reviews/review-1/document", nil))
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.False(t, called)

	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/contract-reviews/bulk/delete", bytes.NewBufferString(`{"ids":["review-1","review-2"]}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"failed":1`)
}

func TestSmartArchiveHandlerDeleteReturnsNoContentAfterServiceSuccess(t *testing.T) {
	h := NewSmartArchiveHandler(&smartArchiveHandlerStub{
		deleteFn: func(context.Context, uint64, string) error { return nil },
	})
	r := legalHandlerTestRouter()
	r.DELETE("/archive/documents/:id", h.DeleteDocument)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/archive/documents/doc-1", nil))
	require.Equal(t, http.StatusNoContent, w.Code)
	require.Empty(t, w.Body.String())
}

func TestSmartArchiveHandlerMapsDeleteAndArchiveErrors(t *testing.T) {
	h := NewSmartArchiveHandler(&smartArchiveHandlerStub{
		deleteFn: func(context.Context, uint64, string) error { return errors.New("reminder table unavailable") },
		archiveFn: func(context.Context, uint64, string, bool) (*types.ArchiveDocument, error) {
			return nil, service.ErrArchiveInvalidState
		},
	})
	r := legalHandlerTestRouter()
	r.DELETE("/archive/documents/:id", h.DeleteDocument)
	r.POST("/archive/documents/:id/archive", h.ArchiveDocument)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/archive/documents/doc-1", nil))
	require.Equal(t, http.StatusInternalServerError, w.Code)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/archive/documents/doc-1/archive", nil))
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSmartArchiveHandlerValidatesBulkIDsAndPassesAction(t *testing.T) {
	called := false
	h := NewSmartArchiveHandler(&smartArchiveHandlerStub{
		batchActionFn: func(_ context.Context, _ uint64, ids []string, action types.ArchiveBulkAction) (*types.ArchiveBulkActionResult, error) {
			called = true
			require.Equal(t, []string{"doc-1"}, ids)
			require.Equal(t, types.ArchiveBulkPurge, action)
			return &types.ArchiveBulkActionResult{Action: action, Requested: 1, Succeeded: 1, Items: []types.ArchiveBulkActionItem{{ID: "doc-1", Success: true}}}, nil
		},
	})
	r := legalHandlerTestRouter()
	r.POST("/archive/documents/bulk/purge", h.BatchPurgeDocuments)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/archive/documents/bulk/purge", bytes.NewBufferString(`{"ids":[]}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.False(t, called)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/archive/documents/bulk/purge", bytes.NewBufferString(`{"ids":["doc-1"]}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"succeeded":1`)
}

func TestContractReviewUploadHandlerAcceptsMultipartFile(t *testing.T) {
	called := false
	h := NewContractReviewHandler(&contractReviewHandlerStub{
		uploadFn: func(_ context.Context, tenantID uint64, userID, id, fileName, mimeType string, fileSize int64, body io.Reader) (*types.ContractReview, error) {
			called = true
			require.Equal(t, uint64(7), tenantID)
			require.Equal(t, "user-1", userID)
			require.Equal(t, "review-1", id)
			require.Equal(t, "agreement.pdf", fileName)
			require.Equal(t, "application/pdf", mimeType)
			data, err := io.ReadAll(body)
			require.NoError(t, err)
			require.Equal(t, []byte("pdf"), data)
			return &types.ContractReview{ID: id, FileName: fileName}, nil
		},
	})
	r := legalHandlerTestRouter()
	r.POST("/contract-reviews/:id/document", h.Upload)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": {`form-data; name="file"; filename="agreement.pdf"`},
		"Content-Type":        {"application/pdf"},
	})
	require.NoError(t, err)
	_, err = part.Write([]byte("pdf"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	req := httptest.NewRequest(http.MethodPost, "/contract-reviews/review-1/document", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusAccepted, w.Code)
	require.True(t, called)
}
