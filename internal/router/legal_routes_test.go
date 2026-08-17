package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/handler"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type legalRouteArchiveServiceStub struct {
	interfaces.SmartArchiveService
}

func (s *legalRouteArchiveServiceStub) GetSettings(context.Context, uint64) (*types.ArchiveSettings, error) {
	return &types.ArchiveSettings{ID: "settings-1"}, nil
}

func (s *legalRouteArchiveServiceStub) UpdateSettings(context.Context, uint64, *types.ArchiveSettings) (*types.ArchiveSettings, error) {
	return &types.ArchiveSettings{ID: "settings-1"}, nil
}

func (s *legalRouteArchiveServiceStub) ListDocuments(context.Context, uint64, string, bool) ([]*types.ArchiveDocument, error) {
	return []*types.ArchiveDocument{}, nil
}

func (s *legalRouteArchiveServiceStub) ArchiveDocument(context.Context, uint64, string, bool) (*types.ArchiveDocument, error) {
	return &types.ArchiveDocument{ID: "doc-1"}, nil
}

func (s *legalRouteArchiveServiceStub) DeleteDocument(context.Context, uint64, string) error {
	return nil
}

func TestSmartArchiveRoutesEnforceLegalRoleMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enforce := true
	cfg := &config.Config{Tenant: &config.TenantConfig{EnableRBAC: &enforce}}
	service := &legalRouteArchiveServiceStub{}
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set(types.UserIDContextKey.String(), "user-1")
		c.Set(types.TenantIDContextKey.String(), uint64(7))
		c.Next()
	})
	v1 := engine.Group("/api/v1")
	RegisterSmartArchiveRoutes(v1, handler.NewSmartArchiveHandler(service), &rbacGuards{cfg: cfg})

	tests := []struct {
		name   string
		role   types.TenantRole
		method string
		path   string
		want   int
	}{
		{name: "viewer can list documents", role: types.TenantRoleViewer, method: http.MethodGet, path: "/api/v1/archive/documents", want: http.StatusOK},
		{name: "viewer cannot archive", role: types.TenantRoleViewer, method: http.MethodPost, path: "/api/v1/archive/documents/doc-1/archive", want: http.StatusForbidden},
		{name: "contributor can update settings", role: types.TenantRoleContributor, method: http.MethodPatch, path: "/api/v1/archive/settings", want: http.StatusOK},
		{name: "contributor can archive", role: types.TenantRoleContributor, method: http.MethodPost, path: "/api/v1/archive/documents/doc-1/archive", want: http.StatusOK},
		{name: "contributor cannot delete", role: types.TenantRoleContributor, method: http.MethodDelete, path: "/api/v1/archive/documents/doc-1", want: http.StatusForbidden},
		{name: "admin can delete", role: types.TenantRoleAdmin, method: http.MethodDelete, path: "/api/v1/archive/documents/doc-1", want: http.StatusNoContent},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(test.method, test.path, nil)
			ctx := context.WithValue(req.Context(), types.TenantRoleContextKey, test.role)
			req = req.WithContext(ctx)
			if test.method == http.MethodPatch {
				req = httptest.NewRequest(test.method, test.path, strings.NewReader("{}"))
				req.Header.Set("Content-Type", "application/json")
				ctx := context.WithValue(req.Context(), types.TenantRoleContextKey, test.role)
				req = req.WithContext(ctx)
			}
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			require.Equal(t, test.want, w.Code)
		})
	}
}
