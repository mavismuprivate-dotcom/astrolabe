package api

import (
	"bytes"
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

func TestBillingOrdersRequiresAuthentication(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "billing.db")
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	h := NewHandlerWithDependencies(astrology.NewService(astrology.NewCityResolver()), store, nil)

	body, _ := json.Marshal(map[string]string{
		"plan_code": "vip_monthly",
		"provider":  "alipay",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated billing order creation, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestBillingCreateAndListOrders(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "billing.db")
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
		func() time.Time { return time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC) },
		func() (string, error) { return "123456", nil },
	)
	h := NewHandlerWithDependencies(astrology.NewService(astrology.NewCityResolver()), store, authSvc)
	authCookie := authenticateTestUser(t, h, "13800138002")

	createBody, _ := json.Marshal(map[string]string{
		"plan_code": "vip_monthly",
		"provider":  "alipay",
	})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/billing/orders", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(authCookie)
	createW := httptest.NewRecorder()
	h.ServeHTTP(createW, createReq)

	if createW.Code != http.StatusCreated {
		t.Fatalf("expected create order status 201, got %d, body=%s", createW.Code, createW.Body.String())
	}

	var createResp struct {
		Order struct {
			ID        string `json:"id"`
			PlanCode  string `json:"plan_code"`
			Provider  string `json:"provider"`
			AmountCNY int64  `json:"amount_cny"`
			Status    string `json:"status"`
		} `json:"order"`
		Payment struct {
			Provider    string `json:"provider"`
			CheckoutURL string `json:"checkout_url"`
		} `json:"payment"`
	}
	if err := json.Unmarshal(createW.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("parse create response: %v", err)
	}
	if createResp.Order.ID == "" {
		t.Fatalf("expected created order ID")
	}
	if createResp.Order.PlanCode != "vip_monthly" || createResp.Order.Provider != "alipay" {
		t.Fatalf("unexpected create order payload: %+v", createResp.Order)
	}
	if createResp.Order.AmountCNY != 2900 {
		t.Fatalf("expected monthly amount 2900, got %d", createResp.Order.AmountCNY)
	}
	if createResp.Order.Status != "pending" {
		t.Fatalf("expected pending order status, got %s", createResp.Order.Status)
	}
	if createResp.Payment.Provider != "alipay" || createResp.Payment.CheckoutURL == "" {
		t.Fatalf("unexpected payment payload: %+v", createResp.Payment)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/billing/orders", nil)
	listReq.AddCookie(authCookie)
	listW := httptest.NewRecorder()
	h.ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Fatalf("expected list orders status 200, got %d, body=%s", listW.Code, listW.Body.String())
	}

	var listResp struct {
		Items []struct {
			ID        string `json:"id"`
			PlanCode  string `json:"plan_code"`
			Provider  string `json:"provider"`
			AmountCNY int64  `json:"amount_cny"`
			Status    string `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listW.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("parse list response: %v", err)
	}
	if len(listResp.Items) != 1 {
		t.Fatalf("expected 1 billing order, got %d", len(listResp.Items))
	}
	if listResp.Items[0].ID != createResp.Order.ID {
		t.Fatalf("expected listed order ID %s, got %s", createResp.Order.ID, listResp.Items[0].ID)
	}
}

func authenticateTestUser(t *testing.T, h *Handler, phone string) *http.Cookie {
	t.Helper()

	requestBody, _ := json.Marshal(map[string]string{"phone": phone})
	requestReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/request-code", bytes.NewReader(requestBody))
	requestReq.Header.Set("Content-Type", "application/json")
	requestW := httptest.NewRecorder()
	h.ServeHTTP(requestW, requestReq)
	if requestW.Code != http.StatusOK {
		t.Fatalf("request code failed: %d %s", requestW.Code, requestW.Body.String())
	}

	verifyBody, _ := json.Marshal(map[string]string{"phone": phone, "code": "123456"})
	verifyReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-code", bytes.NewReader(verifyBody))
	verifyReq.Header.Set("Content-Type", "application/json")
	verifyW := httptest.NewRecorder()
	h.ServeHTTP(verifyW, verifyReq)
	if verifyW.Code != http.StatusOK {
		t.Fatalf("verify code failed: %d %s", verifyW.Code, verifyW.Body.String())
	}

	return sessionCookieFromRecorder(t, verifyW)
}
