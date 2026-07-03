package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/config"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

// stubRegisterUserService is a UserService whose ONLY useful method is
// Register; every other call panics. Using an interface embedding plus a
// targeted override keeps the test focused on the Register handler's
// branching logic without dragging in the entire user service surface.
type stubRegisterUserService struct {
	interfaces.UserService
	register          func(ctx context.Context, req *types.RegisterRequest) (*types.User, error)
	getUserByEmail    func(ctx context.Context, email string) (*types.User, error)
	getUserByTenantID func(ctx context.Context, tenantID uint64) (*types.User, error)
	generateTokens    func(ctx context.Context, user *types.User) (string, string, error)
}

func (s *stubRegisterUserService) Register(ctx context.Context, req *types.RegisterRequest) (*types.User, error) {
	return s.register(ctx, req)
}

func (s *stubRegisterUserService) GetUserByEmail(ctx context.Context, email string) (*types.User, error) {
	return s.getUserByEmail(ctx, email)
}

func (s *stubRegisterUserService) GetUserByTenantID(ctx context.Context, tenantID uint64) (*types.User, error) {
	return s.getUserByTenantID(ctx, tenantID)
}

func (s *stubRegisterUserService) GenerateTokens(ctx context.Context, user *types.User) (string, string, error) {
	return s.generateTokens(ctx, user)
}

type stubAuthTenantService struct {
	interfaces.TenantService
	getTenantByID func(ctx context.Context, id uint64) (*types.Tenant, error)
	listTenants   func(ctx context.Context) ([]*types.Tenant, error)
}

func (s *stubAuthTenantService) GetTenantByID(ctx context.Context, id uint64) (*types.Tenant, error) {
	return s.getTenantByID(ctx, id)
}

func (s *stubAuthTenantService) ListTenants(ctx context.Context) ([]*types.Tenant, error) {
	return s.listTenants(ctx)
}

// errorCapture mirrors gin's default ErrorHandler behaviour for tests:
// when a handler calls c.Error(), we surface it as an HTTP response so the
// recorder reflects the real client-visible status. The production
// middleware does the same thing in middleware/error_handler.go.
func errorCapture() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 {
			return
		}
		err := c.Errors.Last().Err
		if appErr, ok := err.(*apperrors.AppError); ok {
			c.JSON(appErr.HTTPCode, gin.H{"error": appErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func newRegisterTestRouter(h *AuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(errorCapture())
	r.POST("/auth/register", h.Register)
	return r
}

func newAutoSetupTestRouter(h *AuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(errorCapture())
	r.POST("/auth/auto-setup", h.AutoSetup)
	return r
}

func doRegister(t *testing.T, r *gin.Engine, body any) *httptest.ResponseRecorder {
	t.Helper()
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func doAutoSetup(t *testing.T, r *gin.Engine) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/auth/auto-setup", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// validRegisterBody returns a payload that passes parameter validation, so
// each test is exercising the gate logic and not the body parser.
func validRegisterBody() map[string]string {
	return map[string]string{
		"username": "alice",
		"email":    "alice@example.com",
		"password": "supersecret",
	}
}

func TestRegister_InviteOnlyRejects(t *testing.T) {
	// PR 3 (#1303): when auth.registration_mode=invite_only, Register
	// must respond 403 BEFORE touching the user service. The frontend
	// already hides the sign-up link via /auth/config; this is the
	// server-side enforcement for direct API hits.
	called := false
	us := &stubRegisterUserService{
		register: func(context.Context, *types.RegisterRequest) (*types.User, error) {
			called = true
			return &types.User{ID: "u1"}, nil
		},
	}
	h := NewAuthHandler(&config.Config{
		Auth: &config.AuthConfig{RegistrationMode: config.AuthRegistrationModeInviteOnly},
	}, us, nil, nil, nil)

	w := doRegister(t, newRegisterTestRouter(h), validRegisterBody())
	if w.Code != http.StatusForbidden {
		t.Fatalf("invite_only must return 403, got %d body=%s", w.Code, w.Body.String())
	}
	if called {
		t.Fatalf("UserService.Register must not be called when invite_only blocks the request")
	}
}

func TestRegister_SelfServeAllowsRegistration(t *testing.T) {
	// Default registration_mode keeps PR 1 behaviour intact: the gate
	// is dormant and the request reaches the user service. We don't
	// exercise the real service here — just confirm the gate let it
	// through by observing the stub being invoked.
	called := false
	us := &stubRegisterUserService{
		register: func(_ context.Context, _ *types.RegisterRequest) (*types.User, error) {
			called = true
			return &types.User{ID: "u1", Email: "alice@example.com"}, nil
		},
	}
	h := NewAuthHandler(&config.Config{
		Auth: &config.AuthConfig{RegistrationMode: config.AuthRegistrationModeSelfServe},
	}, us, nil, nil, nil)

	w := doRegister(t, newRegisterTestRouter(h), validRegisterBody())
	if w.Code != http.StatusCreated {
		t.Fatalf("self_serve must allow registration, got %d body=%s", w.Code, w.Body.String())
	}
	if !called {
		t.Fatalf("UserService.Register should have been invoked")
	}
}

func TestRegister_NilAuthConfigDoesNotPanic(t *testing.T) {
	// Defensive: a nil Auth section means the operator hasn't set the
	// registration mode at all, which must not crash and must keep the
	// legacy "registration enabled" behaviour. Mirrors the nil guard in
	// the handler so a config-loading bug doesn't take the server down.
	us := &stubRegisterUserService{
		register: func(_ context.Context, _ *types.RegisterRequest) (*types.User, error) {
			return &types.User{ID: "u1", Email: "alice@example.com"}, nil
		},
	}
	h := NewAuthHandler(&config.Config{}, us, nil, nil, nil)

	w := doRegister(t, newRegisterTestRouter(h), validRegisterBody())
	if w.Code != http.StatusCreated {
		t.Fatalf("nil Auth config must fall back to allow, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestAutoSetup_StandardEditionRequiresSingleUserMode(t *testing.T) {
	oldEdition := Edition
	Edition = "standard"
	defer func() { Edition = oldEdition }()

	h := NewAuthHandler(&config.Config{Auth: &config.AuthConfig{}}, &stubRegisterUserService{}, nil, nil, nil)
	w := doAutoSetup(t, newAutoSetupTestRouter(h))
	if w.Code != http.StatusForbidden {
		t.Fatalf("standard auto-setup without single_user_mode must return 403, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestAutoSetup_SingleUserModeAllowsDefaultUser(t *testing.T) {
	oldEdition := Edition
	Edition = "standard"
	defer func() { Edition = oldEdition }()

	user := &types.User{ID: "u1", Username: "default", Email: "admin@weknora.local", TenantID: 7, IsActive: true}
	tenant := &types.Tenant{ID: 7, Name: "Default", APIKey: "k"}
	us := &stubRegisterUserService{
		getUserByEmail: func(context.Context, string) (*types.User, error) { return user, nil },
		generateTokens: func(context.Context, *types.User) (string, string, error) {
			return "access", "refresh", nil
		},
	}
	ts := &stubAuthTenantService{
		getTenantByID: func(context.Context, uint64) (*types.Tenant, error) { return tenant, nil },
	}
	h := NewAuthHandler(&config.Config{Auth: &config.AuthConfig{SingleUserMode: true}}, us, ts, nil, nil)

	w := doAutoSetup(t, newAutoSetupTestRouter(h))
	if w.Code != http.StatusOK {
		t.Fatalf("single_user_mode auto-setup must return 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"token":"access"`) {
		t.Fatalf("auto-setup response missing token: %s", w.Body.String())
	}
}

func TestAutoSetup_SingleUserModeCreatesDefaultUser(t *testing.T) {
	oldEdition := Edition
	Edition = "standard"
	defer func() { Edition = oldEdition }()

	tenant := &types.Tenant{ID: 7, Name: "Default", APIKey: "k"}
	registered := false
	us := &stubRegisterUserService{
		getUserByEmail: func(context.Context, string) (*types.User, error) { return nil, nil },
		register: func(_ context.Context, req *types.RegisterRequest) (*types.User, error) {
			registered = true
			if req.Username != "lexai" || req.Email != "admin@lexai.local" {
				t.Fatalf("unexpected default user: username=%q email=%q", req.Username, req.Email)
			}
			return &types.User{ID: "u1", Username: req.Username, Email: req.Email, TenantID: 7, IsActive: true}, nil
		},
		generateTokens: func(context.Context, *types.User) (string, string, error) {
			return "access", "refresh", nil
		},
	}
	ts := &stubAuthTenantService{
		getTenantByID: func(context.Context, uint64) (*types.Tenant, error) { return tenant, nil },
	}
	h := NewAuthHandler(&config.Config{Auth: &config.AuthConfig{SingleUserMode: true}}, us, ts, nil, nil)

	w := doAutoSetup(t, newAutoSetupTestRouter(h))
	if w.Code != http.StatusOK {
		t.Fatalf("single_user_mode auto-setup must return 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !registered {
		t.Fatalf("single_user_mode must create the default user when missing")
	}
}
