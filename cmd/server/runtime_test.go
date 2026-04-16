package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadConfigUsesDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("ASTROLABE_DB_PATH", "")
	t.Setenv("ASTROLABE_RATE_LIMIT_RPM", "")

	cfg, err := loadConfigFromEnv()
	if err != nil {
		t.Fatalf("loadConfigFromEnv returned error: %v", err)
	}

	if cfg.Port != "8080" {
		t.Fatalf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.DBPath != "data\\astrolabe.db" && cfg.DBPath != "data/astrolabe.db" {
		t.Fatalf("expected default db path, got %s", cfg.DBPath)
	}
	if cfg.RateLimitPerMinute != 120 {
		t.Fatalf("expected default rate limit 120, got %d", cfg.RateLimitPerMinute)
	}
}

func TestLoadConfigRejectsInvalidValues(t *testing.T) {
	t.Setenv("PORT", "abc")
	t.Setenv("ASTROLABE_DB_PATH", " ")
	t.Setenv("ASTROLABE_RATE_LIMIT_RPM", "0")

	_, err := loadConfigFromEnv()
	if err == nil {
		t.Fatalf("expected loadConfigFromEnv to fail")
	}

	msg := err.Error()
	if !strings.Contains(msg, "PORT") {
		t.Fatalf("expected PORT validation error, got %q", msg)
	}
}

func TestRecoverMiddlewareReturnsJSONForAPIRequests(t *testing.T) {
	var logs bytes.Buffer
	logger := log.New(&logs, "", 0)
	handler := recoverMiddleware(logger, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", resp.Code)
	}
	if got := resp.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("expected application/json content-type, got %q", got)
	}
	if !strings.Contains(resp.Body.String(), `"error":"internal server error"`) {
		t.Fatalf("expected JSON error body, got %q", resp.Body.String())
	}
	if !strings.Contains(logs.String(), "panic recovered") {
		t.Fatalf("expected panic to be logged, got %q", logs.String())
	}
}

func TestRequestLoggingMiddlewareLogsMethodPathAndStatus(t *testing.T) {
	var logs bytes.Buffer
	logger := log.New(&logs, "", 0)
	handler := requestLoggingMiddleware(logger, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/chart/natal", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	line := logs.String()
	if !strings.Contains(line, "POST") || !strings.Contains(line, "/api/v1/chart/natal") || !strings.Contains(line, "201") {
		t.Fatalf("expected log line with method, path, status; got %q", line)
	}
}

func TestRateLimitMiddlewareReturns429AfterLimit(t *testing.T) {
	now := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	handler := rateLimitMiddleware(1, func() time.Time { return now }, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/reports", nil)
	req1.RemoteAddr = "127.0.0.1:1234"
	resp1 := httptest.NewRecorder()
	handler.ServeHTTP(resp1, req1)
	if resp1.Code != http.StatusOK {
		t.Fatalf("expected first request status 200, got %d", resp1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/reports", nil)
	req2.RemoteAddr = "127.0.0.1:1234"
	resp2 := httptest.NewRecorder()
	handler.ServeHTTP(resp2, req2)
	if resp2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request status 429, got %d", resp2.Code)
	}
	if !strings.Contains(resp2.Body.String(), "rate limit exceeded") {
		t.Fatalf("expected rate limit message, got %q", resp2.Body.String())
	}
}

func TestRateLimitMiddlewareSkipsHealthChecks(t *testing.T) {
	handler := rateLimitMiddleware(1, time.Now, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected healthz status 200 on attempt %d, got %d", i+1, resp.Code)
		}
	}
}

func TestMainReturnsConfigErrorForInvalidPort(t *testing.T) {
	t.Setenv("PORT", "70000")
	t.Setenv("ASTROLABE_DB_PATH", "")
	t.Setenv("ASTROLABE_RATE_LIMIT_RPM", "")

	_, err := loadConfigFromEnv()
	if err == nil {
		t.Fatalf("expected invalid port error")
	}
	if !strings.Contains(err.Error(), "PORT") {
		t.Fatalf("expected PORT error, got %q", err)
	}
}

func TestLoadConfigAcceptsCustomValues(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("ASTROLABE_DB_PATH", "data/custom.db")
	t.Setenv("ASTROLABE_RATE_LIMIT_RPM", "240")

	cfg, err := loadConfigFromEnv()
	if err != nil {
		t.Fatalf("loadConfigFromEnv returned error: %v", err)
	}

	if cfg.Port != "9090" {
		t.Fatalf("expected custom port 9090, got %s", cfg.Port)
	}
	if cfg.DBPath != "data/custom.db" {
		t.Fatalf("expected custom db path, got %s", cfg.DBPath)
	}
	if cfg.RateLimitPerMinute != 240 {
		t.Fatalf("expected custom rate limit 240, got %d", cfg.RateLimitPerMinute)
	}
}

func TestLoadConfigIgnoresUnrelatedEnvironment(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("ASTROLABE_DB_PATH", "")
	t.Setenv("ASTROLABE_RATE_LIMIT_RPM", "")
	t.Setenv("HOME", os.Getenv("HOME"))

	if _, err := loadConfigFromEnv(); err != nil {
		t.Fatalf("expected unrelated env vars to be ignored, got %v", err)
	}
}
