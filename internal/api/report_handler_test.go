package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"astrolabe/internal/astrology"
	"astrolabe/internal/storage"
)

func sessionCookieFromRecorder(t *testing.T, w *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()

	resp := w.Result()
	defer resp.Body.Close()
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected session cookie to be set")
	}
	return cookies[0]
}

func TestNatalChartPersistsAndFetchesSavedReport(t *testing.T) {
	t.Parallel()

	storePath := filepath.Join(t.TempDir(), "reports.json")
	store, err := storage.NewFileStore(storePath)
	if err != nil {
		t.Fatalf("NewFileStore returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	svc := astrology.NewService(astrology.NewCityResolver())
	h := NewHandlerWithStore(svc, store)

	payload := map[string]any{
		"birth_date":    "1990-01-01",
		"birth_time":    "08:15",
		"birth_city":    "Shanghai",
		"birth_country": "China",
		"language":      "zh-CN",
	}
	buf, _ := json.Marshal(payload)
	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/chart/natal", bytes.NewReader(buf))
	postReq.Header.Set("Content-Type", "application/json")
	postW := httptest.NewRecorder()
	h.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", postW.Code, postW.Body.String())
	}
	sessionCookie := sessionCookieFromRecorder(t, postW)

	var created astrology.NatalChartResponse
	if err := json.Unmarshal(postW.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to parse create response: %v", err)
	}
	if created.ReportID == "" {
		t.Fatalf("expected ReportID to be populated")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/reports/"+created.ReportID, nil)
	getReq.AddCookie(sessionCookie)
	getW := httptest.NewRecorder()
	h.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected GET status 200, got %d, body=%s", getW.Code, getW.Body.String())
	}

	var fetched astrology.NatalChartResponse
	if err := json.Unmarshal(getW.Body.Bytes(), &fetched); err != nil {
		t.Fatalf("failed to parse get response: %v", err)
	}
	if fetched.ReportID != created.ReportID {
		t.Fatalf("expected fetched ReportID %s, got %s", created.ReportID, fetched.ReportID)
	}
	if fetched.Meta.Input.BirthCity != created.Meta.Input.BirthCity {
		t.Fatalf("expected fetched birth city %s, got %s", created.Meta.Input.BirthCity, fetched.Meta.Input.BirthCity)
	}
}

func TestReportPDFExportReturnsPDF(t *testing.T) {
	t.Parallel()

	storePath := filepath.Join(t.TempDir(), "reports.json")
	store, err := storage.NewFileStore(storePath)
	if err != nil {
		t.Fatalf("NewFileStore returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	svc := astrology.NewService(astrology.NewCityResolver())
	h := NewHandlerWithStore(svc, store)

	payload := map[string]any{
		"birth_date":    "1990-01-01",
		"birth_time":    "08:15",
		"birth_city":    "Shanghai",
		"birth_country": "China",
		"language":      "zh-CN",
	}
	buf, _ := json.Marshal(payload)
	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/chart/natal", bytes.NewReader(buf))
	postReq.Header.Set("Content-Type", "application/json")
	postW := httptest.NewRecorder()
	h.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", postW.Code, postW.Body.String())
	}
	sessionCookie := sessionCookieFromRecorder(t, postW)

	var created astrology.NatalChartResponse
	if err := json.Unmarshal(postW.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to parse create response: %v", err)
	}
	if created.ReportID == "" {
		t.Fatalf("expected ReportID to be populated")
	}

	pdfReq := httptest.NewRequest(http.MethodGet, "/api/v1/reports/"+created.ReportID+"/pdf", nil)
	pdfReq.AddCookie(sessionCookie)
	pdfW := httptest.NewRecorder()
	h.ServeHTTP(pdfW, pdfReq)

	if pdfW.Code != http.StatusOK {
		t.Fatalf("expected PDF status 200, got %d, body=%s", pdfW.Code, pdfW.Body.String())
	}
	if got := pdfW.Header().Get("Content-Type"); got != "application/pdf" {
		t.Fatalf("expected application/pdf content-type, got %s", got)
	}
	if body := pdfW.Body.Bytes(); len(body) < 8 || string(body[:5]) != "%PDF-" {
		t.Fatalf("expected PDF body header, got %q", string(body))
	}
}

func TestListReportsReturnsSavedItems(t *testing.T) {
	t.Parallel()

	storePath := filepath.Join(t.TempDir(), "reports.json")
	store, err := storage.NewFileStore(storePath)
	if err != nil {
		t.Fatalf("NewFileStore returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	svc := astrology.NewService(astrology.NewCityResolver())
	h := NewHandlerWithStore(svc, store)

	sessionPayloads := []map[string]any{
		{
			"birth_date":    "1990-01-01",
			"birth_time":    "08:15",
			"birth_city":    "Shanghai",
			"birth_country": "China",
			"language":      "zh-CN",
		},
		{
			"birth_date":    "1992-08-16",
			"birth_time":    "09:20",
			"birth_city":    "Nanjing",
			"birth_country": "China",
			"language":      "zh-CN",
		},
	}

	var sessionCookie *http.Cookie
	for _, payload := range sessionPayloads {
		buf, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chart/natal", bytes.NewReader(buf))
		req.Header.Set("Content-Type", "application/json")
		if sessionCookie != nil {
			req.AddCookie(sessionCookie)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
		}
		if sessionCookie == nil {
			sessionCookie = sessionCookieFromRecorder(t, w)
		}
	}

	buf, _ := json.Marshal(map[string]any{
		"birth_date":    "1993-02-12",
		"birth_time":    "10:30",
		"birth_city":    "Beijing",
		"birth_country": "China",
		"language":      "zh-CN",
	})
	otherReq := httptest.NewRequest(http.MethodPost, "/api/v1/chart/natal", bytes.NewReader(buf))
	otherReq.Header.Set("Content-Type", "application/json")
	otherW := httptest.NewRecorder()
	h.ServeHTTP(otherW, otherReq)
	if otherW.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", otherW.Code, otherW.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/reports", nil)
	listReq.AddCookie(sessionCookie)
	listW := httptest.NewRecorder()
	h.ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Fatalf("expected list status 200, got %d, body=%s", listW.Code, listW.Body.String())
	}

	var resp struct {
		Items []storage.ReportSummary `json:"items"`
	}
	if err := json.Unmarshal(listW.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse list response: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}
	if resp.Items[0].BirthCity != "南京" {
		t.Fatalf("expected newest birth city 南京 first, got %s", resp.Items[0].BirthCity)
	}
	if resp.Items[1].BirthCity != "上海市" {
		t.Fatalf("expected oldest birth city 上海市 second, got %s", resp.Items[1].BirthCity)
	}
}

func TestReportAccessIsScopedToSessionCookie(t *testing.T) {
	t.Parallel()

	storePath := filepath.Join(t.TempDir(), "reports.json")
	store, err := storage.NewFileStore(storePath)
	if err != nil {
		t.Fatalf("NewFileStore returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	svc := astrology.NewService(astrology.NewCityResolver())
	h := NewHandlerWithStore(svc, store)

	payload := map[string]any{
		"birth_date":    "1990-01-01",
		"birth_time":    "08:15",
		"birth_city":    "Shanghai",
		"birth_country": "China",
		"language":      "zh-CN",
	}
	buf, _ := json.Marshal(payload)
	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/chart/natal", bytes.NewReader(buf))
	postReq.Header.Set("Content-Type", "application/json")
	postW := httptest.NewRecorder()
	h.ServeHTTP(postW, postReq)
	if postW.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", postW.Code, postW.Body.String())
	}

	var created astrology.NatalChartResponse
	if err := json.Unmarshal(postW.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to parse create response: %v", err)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/reports/"+created.ReportID, nil)
	getReq.AddCookie(&http.Cookie{Name: "astrolabe_session", Value: "sess_other"})
	getW := httptest.NewRecorder()
	h.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusNotFound {
		t.Fatalf("expected GET status 404 for other session, got %d, body=%s", getW.Code, getW.Body.String())
	}
}
