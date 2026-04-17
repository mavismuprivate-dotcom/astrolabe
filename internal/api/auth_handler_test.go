package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"astrolabe/internal/auth"
	"astrolabe/internal/astrology"
	"astrolabe/internal/storage"
)

type captureCodeSender struct {
	lastPhone string
	lastCode  string
}

func (c *captureCodeSender) SendLoginCode(_ context.Context, phone string, code string) error {
	c.lastPhone = phone
	c.lastCode = code
	return nil
}

func TestAuthRequestVerifyMeAndLogoutFlow(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "reports.db")
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	sender := &captureCodeSender{}
	authSvc := auth.NewService(
		store,
		sender,
		func() time.Time { return time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC) },
		func() (string, error) { return "123456", nil },
	)
	h := NewHandlerWithDependencies(astrology.NewService(astrology.NewCityResolver()), store, authSvc)

	requestBody, _ := json.Marshal(map[string]string{"phone": "13800138000"})
	requestReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/request-code", bytes.NewReader(requestBody))
	requestReq.Header.Set("Content-Type", "application/json")
	requestW := httptest.NewRecorder()
	h.ServeHTTP(requestW, requestReq)

	if requestW.Code != http.StatusOK {
		t.Fatalf("expected request-code status 200, got %d, body=%s", requestW.Code, requestW.Body.String())
	}
	if sender.lastPhone != "13800138000" || sender.lastCode != "123456" {
		t.Fatalf("expected captured login code for 13800138000, got phone=%s code=%s", sender.lastPhone, sender.lastCode)
	}

	verifyBody, _ := json.Marshal(map[string]string{"phone": "13800138000", "code": "123456"})
	verifyReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-code", bytes.NewReader(verifyBody))
	verifyReq.Header.Set("Content-Type", "application/json")
	verifyW := httptest.NewRecorder()
	h.ServeHTTP(verifyW, verifyReq)

	if verifyW.Code != http.StatusOK {
		t.Fatalf("expected verify-code status 200, got %d, body=%s", verifyW.Code, verifyW.Body.String())
	}
	authCookie := sessionCookieFromRecorder(t, verifyW)
	if authCookie.Name != authCookieName {
		t.Fatalf("expected auth cookie %s, got %s", authCookieName, authCookie.Name)
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	meReq.AddCookie(authCookie)
	meW := httptest.NewRecorder()
	h.ServeHTTP(meW, meReq)

	if meW.Code != http.StatusOK {
		t.Fatalf("expected me status 200, got %d, body=%s", meW.Code, meW.Body.String())
	}

	var meResp struct {
		Authenticated bool `json:"authenticated"`
		User          struct {
			Phone string `json:"phone"`
		} `json:"user"`
	}
	if err := json.Unmarshal(meW.Body.Bytes(), &meResp); err != nil {
		t.Fatalf("parse me response: %v", err)
	}
	if !meResp.Authenticated || meResp.User.Phone != "13800138000" {
		t.Fatalf("expected authenticated user 13800138000, got %+v", meResp)
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutReq.AddCookie(authCookie)
	logoutW := httptest.NewRecorder()
	h.ServeHTTP(logoutW, logoutReq)
	if logoutW.Code != http.StatusOK {
		t.Fatalf("expected logout status 200, got %d, body=%s", logoutW.Code, logoutW.Body.String())
	}

	meAfterLogoutReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	meAfterLogoutReq.AddCookie(authCookie)
	meAfterLogoutW := httptest.NewRecorder()
	h.ServeHTTP(meAfterLogoutW, meAfterLogoutReq)
	if meAfterLogoutW.Code != http.StatusOK {
		t.Fatalf("expected me-after-logout status 200, got %d, body=%s", meAfterLogoutW.Code, meAfterLogoutW.Body.String())
	}
	if err := json.Unmarshal(meAfterLogoutW.Body.Bytes(), &meResp); err != nil {
		t.Fatalf("parse me-after-logout response: %v", err)
	}
	if meResp.Authenticated {
		t.Fatalf("expected authenticated=false after logout, got %+v", meResp)
	}
}
