package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"astrolabe/internal/astrology"
)

type Handler struct {
	svc *astrology.Service
	mux *http.ServeMux
}

func NewHandler(svc *astrology.Service) *Handler {
	if svc == nil {
		svc = astrology.NewService(astrology.NewCityResolver())
	}
	h := &Handler{svc: svc, mux: http.NewServeMux()}
	h.routes()
	return h
}

func (h *Handler) routes() {
	h.mux.HandleFunc("/healthz", h.handleHealthz)
	h.mux.HandleFunc("/api/v1/chart/natal", h.handleNatalChart)
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

	resp, err := h.svc.GenerateNatalChart(r.Context(), req)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errInternal) {
			status = http.StatusInternalServerError
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
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
