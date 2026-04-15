package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"astrolabe/internal/astrology"
	reportpdf "astrolabe/internal/pdf"
	"astrolabe/internal/storage"
)

type Handler struct {
	svc    *astrology.Service
	reports storage.ReportStore
	mux    *http.ServeMux
}

const sessionCookieName = "astrolabe_session"

func NewHandler(svc *astrology.Service) *Handler {
	return NewHandlerWithStore(svc, nil)
}

func NewHandlerWithStore(svc *astrology.Service, reports storage.ReportStore) *Handler {
	if svc == nil {
		svc = astrology.NewService(astrology.NewCityResolver())
	}
	h := &Handler{svc: svc, reports: reports, mux: http.NewServeMux()}
	h.routes()
	return h
}

func (h *Handler) routes() {
	h.mux.HandleFunc("/healthz", h.handleHealthz)
	h.mux.HandleFunc("/api/v1/chart/natal", h.handleNatalChart)
	h.mux.HandleFunc("/api/v1/reports", h.handleListReports)
	h.mux.HandleFunc("/api/v1/reports/", h.handleReportRoutes)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleNatalChart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, http.StatusBadRequest, "content-type must be application/json")
		return
	}

	var req astrology.NatalChartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	sessionID := ensureSession(w, r)

	resp, err := h.svc.GenerateNatalChart(r.Context(), req)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errInternal) {
			status = http.StatusInternalServerError
		}
		writeError(w, status, err.Error())
		return
	}

	if h.reports != nil {
		reportID := storage.NewReportID()
		resp.ReportID = reportID
		if err := h.reports.SaveReport(r.Context(), storage.Report{
			ID:        reportID,
			SessionID: sessionID,
			CreatedAt: resp.Meta.GeneratedAt,
			Response:  resp,
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to save report")
			return
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleReportRoutes(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/pdf") {
		h.handleGetReportPDF(w, r)
		return
	}
	h.handleGetReport(w, r)
}

func (h *Handler) handleGetReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.reports == nil {
		writeError(w, http.StatusNotFound, "report storage unavailable")
		return
	}
	sessionID := ensureSession(w, r)

	id := strings.TrimPrefix(r.URL.Path, "/api/v1/reports/")
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusNotFound, "report not found")
		return
	}

	report, err := h.reports.GetReport(r.Context(), id, sessionID)
	if errors.Is(err, storage.ErrReportNotFound) {
		writeError(w, http.StatusNotFound, "report not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load report")
		return
	}

	writeJSON(w, http.StatusOK, report.Response)
}

func (h *Handler) handleGetReportPDF(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.reports == nil {
		writeError(w, http.StatusNotFound, "report storage unavailable")
		return
	}
	sessionID := ensureSession(w, r)

	id := strings.TrimPrefix(r.URL.Path, "/api/v1/reports/")
	id = strings.TrimSuffix(id, "/pdf")
	id = strings.TrimSuffix(id, "/")
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusNotFound, "report not found")
		return
	}

	report, err := h.reports.GetReport(r.Context(), id, sessionID)
	if errors.Is(err, storage.ErrReportNotFound) {
		writeError(w, http.StatusNotFound, "report not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load report")
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="astrolabe-report-`+id+`.pdf"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(reportpdf.BuildReport(report.Response))
}

func (h *Handler) handleListReports(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.reports == nil {
		writeJSON(w, http.StatusOK, map[string]any{"items": []storage.ReportSummary{}})
		return
	}
	sessionID := ensureSession(w, r)

	items, err := h.reports.ListReports(r.Context(), sessionID, 20)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list reports")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

var errInternal = errors.New("internal")

func writeJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]any{
		"error": msg,
	})
}

func ensureSession(w http.ResponseWriter, r *http.Request) string {
	if cookie, err := r.Cookie(sessionCookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
		return cookie.Value
	}

	sessionID := storage.NewSessionID()
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   60 * 60 * 24 * 30,
	})
	return sessionID
}
