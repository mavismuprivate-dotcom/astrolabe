package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

type runtimeConfig struct {
	Port               string
	DBPath             string
	RateLimitPerMinute int
}

func loadConfigFromEnv() (runtimeConfig, error) {
	cfg := runtimeConfig{
		Port:               strings.TrimSpace(os.Getenv("PORT")),
		DBPath:             strings.TrimSpace(os.Getenv("ASTROLABE_DB_PATH")),
		RateLimitPerMinute: 120,
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if _, err := parsePort(cfg.Port); err != nil {
		return runtimeConfig{}, fmt.Errorf("invalid PORT: %w", err)
	}

	if cfg.DBPath == "" {
		cfg.DBPath = filepath.Join("data", "astrolabe.db")
	}

	if raw := strings.TrimSpace(os.Getenv("ASTROLABE_RATE_LIMIT_RPM")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			return runtimeConfig{}, fmt.Errorf("invalid ASTROLABE_RATE_LIMIT_RPM: must be a positive integer")
		}
		cfg.RateLimitPerMinute = value
	}

	return cfg, nil
}

func parsePort(value string) (int, error) {
	port, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("must be between 1 and 65535")
	}
	return port, nil
}

func withRuntimeMiddleware(logger *log.Logger, rateLimitPerMinute int, next http.Handler) http.Handler {
	handler := rateLimitMiddleware(rateLimitPerMinute, time.Now, next)
	handler = recoverMiddleware(logger, handler)
	handler = requestLoggingMiddleware(logger, handler)
	return handler
}

func requestLoggingMiddleware(logger *log.Logger, next http.Handler) http.Handler {
	if logger == nil {
		logger = log.New(os.Stdout, "", log.LstdFlags)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(recorder, r)
		logger.Printf("request method=%s path=%s status=%d duration=%s client=%s", r.Method, r.URL.Path, recorder.statusCode, time.Since(start).Round(time.Millisecond), clientIdentifier(r))
	})
}

func recoverMiddleware(logger *log.Logger, next http.Handler) http.Handler {
	if logger == nil {
		logger = log.New(os.Stdout, "", log.LstdFlags)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Printf("panic recovered method=%s path=%s err=%v\n%s", r.Method, r.URL.Path, rec, debug.Stack())
				writeRuntimeError(w, r, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func rateLimitMiddleware(limit int, now func() time.Time, next http.Handler) http.Handler {
	limiter := &fixedWindowRateLimiter{
		limit:  limit,
		now:    now,
		counts: map[string]int{},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !shouldRateLimit(r) {
			next.ServeHTTP(w, r)
			return
		}
		if !limiter.allow(clientIdentifier(r)) {
			w.Header().Set("Retry-After", "60")
			writeRuntimeError(w, r, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func shouldRateLimit(r *http.Request) bool {
	if r.URL.Path == "/healthz" {
		return false
	}
	return strings.HasPrefix(r.URL.Path, "/api/")
}

func writeRuntimeError(w http.ResponseWriter, r *http.Request, code int, msg string) {
	if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/healthz" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": msg})
		return
	}
	http.Error(w, msg, code)
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

type fixedWindowRateLimiter struct {
	mu          sync.Mutex
	windowStart time.Time
	limit       int
	now         func() time.Time
	counts      map[string]int
}

func (l *fixedWindowRateLimiter) allow(client string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	current := l.now().UTC().Truncate(time.Minute)
	if l.windowStart.IsZero() || !l.windowStart.Equal(current) {
		l.windowStart = current
		l.counts = map[string]int{}
	}

	l.counts[client]++
	return l.counts[client] <= l.limit
}

func clientIdentifier(r *http.Request) string {
	forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0])
	if forwarded != "" {
		return forwarded
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	if strings.TrimSpace(r.RemoteAddr) != "" {
		return r.RemoteAddr
	}
	return "unknown"
}
