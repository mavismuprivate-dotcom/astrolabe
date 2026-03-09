package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"astrolabe/internal/astrology"
)

func TestHealthz(t *testing.T) {
	svc := astrology.NewService(astrology.NewCityResolver())
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestNatalChartValidationError(t *testing.T) {
	svc := astrology.NewService(astrology.NewCityResolver())
	h := NewHandler(svc)

	payload := map[string]any{
		"birth_date": "1990-01-01",
	}
	buf, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chart/natal", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestNatalChartOK(t *testing.T) {
	svc := astrology.NewService(astrology.NewCityResolver())
	h := NewHandler(svc)

	payload := map[string]any{
		"birth_date":    "1990-01-01",
		"birth_time":    "08:15",
		"birth_city":    "Shanghai",
		"birth_country": "China",
		"language":      "zh-CN",
	}
	buf, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chart/natal", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var resp astrology.NatalChartResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Meta.Confidence <= 0 {
		t.Fatalf("expected confidence > 0")
	}
	if len(resp.Chart.Houses) != 12 {
		t.Fatalf("expected 12 houses, got %d", len(resp.Chart.Houses))
	}
}

func TestNatalChartChineseNanjingOK(t *testing.T) {
	svc := astrology.NewService(astrology.NewCityResolver())
	h := NewHandler(svc)

	payload := map[string]any{
		"birth_date":    "1992-08-16",
		"birth_time":    "09:20",
		"birth_city":    "南京",
		"birth_country": "中国",
		"language":      "zh-CN",
	}
	buf, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chart/natal", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var resp astrology.NatalChartResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Meta.Input.Timezone != "Asia/Shanghai" {
		t.Fatalf("expected Asia/Shanghai timezone, got %s", resp.Meta.Input.Timezone)
	}
}

func TestNatalChartProvincePayloadOK(t *testing.T) {
	svc := astrology.NewService(astrology.NewCityResolver())
	h := NewHandler(svc)

	payload := map[string]any{
		"birth_date":     "1995-06-21",
		"birth_time":     "14:40",
		"birth_province": "江苏省",
		"language":       "zh-CN",
	}
	buf, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chart/natal", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var resp astrology.NatalChartResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Meta.Input.BirthProvince != "江苏省" {
		t.Fatalf("expected province 江苏省, got %s", resp.Meta.Input.BirthProvince)
	}
}
