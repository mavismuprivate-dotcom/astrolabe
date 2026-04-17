package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"astrolabe/internal/auth"
	"astrolabe/internal/astrology"
	reportpdf "astrolabe/internal/pdf"
	"astrolabe/internal/storage"
)

type Handler struct {
	svc     *astrology.Service
	reports storage.ReportStore
	billing storage.BillingStore
	auth    *auth.Service
	mux     *http.ServeMux
	now     func() time.Time
}

type membershipReader interface {
	GetMembershipByUserID(ctx context.Context, userID string) (storage.Membership, error)
}

const sessionCookieName = "astrolabe_session"
const authCookieName = "astrolabe_auth"

type billingPlan struct {
	Code      string
	AmountCNY int64
}

var billingPlans = map[string]billingPlan{
	"vip_monthly":   {Code: "vip_monthly", AmountCNY: 2900},
	"vip_quarterly": {Code: "vip_quarterly", AmountCNY: 6800},
	"vip_yearly":    {Code: "vip_yearly", AmountCNY: 19800},
}

func NewHandler(svc *astrology.Service) *Handler {
	return NewHandlerWithDependencies(svc, nil, nil)
}

func NewHandlerWithStore(svc *astrology.Service, reports storage.ReportStore) *Handler {
	return NewHandlerWithDependencies(svc, reports, nil)
}

func NewHandlerWithDependencies(svc *astrology.Service, reports storage.ReportStore, authSvc *auth.Service) *Handler {
	if svc == nil {
		svc = astrology.NewService(astrology.NewCityResolver())
	}
	var billingStore storage.BillingStore
	if store, ok := reports.(storage.BillingStore); ok {
		billingStore = store
	}
	h := &Handler{svc: svc, reports: reports, billing: billingStore, auth: authSvc, mux: http.NewServeMux(), now: time.Now}
	h.routes()
	return h
}

func (h *Handler) routes() {
	h.mux.HandleFunc("/healthz", h.handleHealthz)
	h.mux.HandleFunc("/api/v1/chart/natal", h.handleNatalChart)
	h.mux.HandleFunc("/api/v1/auth/request-code", h.handleAuthRequestCode)
	h.mux.HandleFunc("/api/v1/auth/verify-code", h.handleAuthVerifyCode)
	h.mux.HandleFunc("/api/v1/auth/logout", h.handleAuthLogout)
	h.mux.HandleFunc("/api/v1/me", h.handleCurrentUser)
	h.mux.HandleFunc("/api/v1/billing/orders", h.handleBillingOrders)
	h.mux.HandleFunc("/api/v1/billing/orders/", h.handleBillingOrderRoutes)
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
	if strings.HasSuffix(r.URL.Path, "/json") {
		h.handleGetReportJSON(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/text") {
		h.handleGetReportText(w, r)
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
	if !h.ensureVIPExportAccess(w, r) {
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="astrolabe-report-`+id+`.pdf"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(reportpdf.BuildReport(report.Response))
}

func (h *Handler) handleGetReportJSON(w http.ResponseWriter, r *http.Request) {
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
	id = strings.TrimSuffix(id, "/json")
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
	if !h.ensureVIPExportAccess(w, r) {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="astrolabe-report-`+id+`.json"`)
	writeJSON(w, http.StatusOK, report.Response)
}

func (h *Handler) handleGetReportText(w http.ResponseWriter, r *http.Request) {
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
	id = strings.TrimSuffix(id, "/text")
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
	if !h.ensureVIPExportAccess(w, r) {
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="astrolabe-report-`+id+`.txt"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(report.Response.Reading.TextReport))
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

func (h *Handler) handleAuthRequestCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.auth == nil {
		writeError(w, http.StatusNotImplemented, "auth unavailable")
		return
	}
	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, http.StatusBadRequest, "content-type must be application/json")
		return
	}

	var req struct {
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.auth.RequestCode(r.Context(), req.Phone); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errInternal) {
			status = http.StatusInternalServerError
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) handleAuthVerifyCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.auth == nil {
		writeError(w, http.StatusNotImplemented, "auth unavailable")
		return
	}
	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, http.StatusBadRequest, "content-type must be application/json")
		return
	}

	var req struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	user, session, err := h.auth.VerifyCode(r.Context(), req.Phone, req.Code)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errInternal) {
			status = http.StatusInternalServerError
		}
		writeError(w, status, err.Error())
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   60 * 60 * 24 * 30,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"user": map[string]any{
			"id":    user.ID,
			"phone": user.Phone,
		},
	})
}

func (h *Handler) handleCurrentUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.auth == nil {
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
		return
	}
	cookie, err := r.Cookie(authCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
		return
	}
	user, err := h.auth.CurrentUser(r.Context(), cookie.Value)
	if errors.Is(err, storage.ErrAuthSessionNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load current user")
		return
	}
	membershipPayload := map[string]any{
		"status":     "none",
		"plan_code":  "",
		"is_vip":     false,
		"expires_at": nil,
	}
	if reader, ok := h.reports.(membershipReader); ok {
		membership, err := reader.GetMembershipByUserID(r.Context(), user.ID)
		if err == nil {
			membershipPayload["status"] = membership.Status
			membershipPayload["plan_code"] = membership.PlanCode
			membershipPayload["is_vip"] = membership.Status == "active"
			if membership.ExpiresAt != nil {
				membershipPayload["expires_at"] = membership.ExpiresAt.UTC().Format(time.RFC3339Nano)
			}
		} else if !errors.Is(err, storage.ErrMembershipNotFound) {
			writeError(w, http.StatusInternalServerError, "failed to load membership status")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"user": map[string]any{
			"id":    user.ID,
			"phone": user.Phone,
		},
		"membership": membershipPayload,
	})
}

func (h *Handler) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.auth != nil {
		if cookie, err := r.Cookie(authCookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
			_ = h.auth.Logout(r.Context(), cookie.Value)
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) handleBillingOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.handleCreateBillingOrder(w, r)
	case http.MethodGet:
		h.handleListBillingOrders(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleBillingOrderRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/mock-pay") {
		h.handleMockPayBillingOrder(w, r)
		return
	}
	writeError(w, http.StatusNotFound, "billing order not found")
}

func (h *Handler) handleCreateBillingOrder(w http.ResponseWriter, r *http.Request) {
	if h.billing == nil {
		writeError(w, http.StatusNotImplemented, "billing unavailable")
		return
	}
	user, ok := h.requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, http.StatusBadRequest, "content-type must be application/json")
		return
	}

	var req struct {
		PlanCode string `json:"plan_code"`
		Provider string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	plan, exists := billingPlans[strings.TrimSpace(req.PlanCode)]
	if !exists {
		writeError(w, http.StatusBadRequest, "unsupported plan_code")
		return
	}
	provider := strings.TrimSpace(strings.ToLower(req.Provider))
	if provider != "alipay" && provider != "wechat" {
		writeError(w, http.StatusBadRequest, "unsupported provider")
		return
	}

	order := storage.PaymentOrder{
		ID:        storage.NewPaymentOrderID(),
		UserID:    user.ID,
		Provider:  provider,
		PlanCode:  plan.Code,
		AmountCNY: plan.AmountCNY,
		Status:    "pending",
		CreatedAt: h.now().UTC(),
	}
	if err := h.billing.SavePaymentOrder(r.Context(), order); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create payment order")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"order": map[string]any{
			"id":         order.ID,
			"plan_code":  order.PlanCode,
			"provider":   order.Provider,
			"amount_cny": order.AmountCNY,
			"status":     order.Status,
			"created_at": order.CreatedAt.UTC().Format(time.RFC3339Nano),
		},
		"payment": map[string]any{
			"provider":     provider,
			"mode":         "mock",
			"checkout_url": "mockpay://checkout/" + order.ID + "?provider=" + provider,
		},
	})
}

func (h *Handler) handleListBillingOrders(w http.ResponseWriter, r *http.Request) {
	if h.billing == nil {
		writeError(w, http.StatusNotImplemented, "billing unavailable")
		return
	}
	user, ok := h.requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	items, err := h.billing.ListPaymentOrdersByUserID(r.Context(), user.ID, 20)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list payment orders")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) handleMockPayBillingOrder(w http.ResponseWriter, r *http.Request) {
	if h.billing == nil {
		writeError(w, http.StatusNotImplemented, "billing unavailable")
		return
	}
	user, ok := h.requireAuthenticatedUser(w, r)
	if !ok {
		return
	}

	orderID := strings.TrimPrefix(r.URL.Path, "/api/v1/billing/orders/")
	orderID = strings.TrimSuffix(orderID, "/mock-pay")
	orderID = strings.TrimSuffix(orderID, "/")
	if orderID == "" || strings.Contains(orderID, "/") {
		writeError(w, http.StatusNotFound, "payment order not found")
		return
	}

	order, err := h.billing.GetPaymentOrderByID(r.Context(), orderID)
	if errors.Is(err, storage.ErrPaymentOrderNotFound) {
		writeError(w, http.StatusNotFound, "payment order not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load payment order")
		return
	}
	if order.UserID != user.ID {
		writeError(w, http.StatusNotFound, "payment order not found")
		return
	}

	now := h.now().UTC()
	if order.Status != "paid" {
		order.Status = "paid"
		order.PaidAt = &now
		if err := h.billing.SavePaymentOrder(r.Context(), order); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update payment order")
			return
		}
		membership, err := h.activateMembershipForOrder(r.Context(), user.ID, order.PlanCode, now)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to activate membership")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"order":      order,
			"membership": membershipResponse(membership),
		})
		return
	}

	membership, err := h.billing.GetMembershipByUserID(r.Context(), user.ID)
	if err != nil && !errors.Is(err, storage.ErrMembershipNotFound) {
		writeError(w, http.StatusInternalServerError, "failed to load membership status")
		return
	}
	resp := map[string]any{"order": order}
	if err == nil {
		resp["membership"] = membershipResponse(membership)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) activateMembershipForOrder(ctx context.Context, userID string, planCode string, now time.Time) (storage.Membership, error) {
	expiresAt, ok := membershipExpiry(now, planCode)
	if !ok {
		return storage.Membership{}, errors.New("unsupported plan_code")
	}
	startAt := now
	if current, err := h.billing.GetMembershipByUserID(ctx, userID); err == nil {
		if current.Status == "active" && current.ExpiresAt != nil && current.ExpiresAt.After(now) {
			startAt = *current.ExpiresAt
			nextExpiry, ok := membershipExpiry(startAt, planCode)
			if !ok {
				return storage.Membership{}, errors.New("unsupported plan_code")
			}
			expiresAt = nextExpiry
		}
	} else if !errors.Is(err, storage.ErrMembershipNotFound) {
		return storage.Membership{}, err
	}
	membership := storage.Membership{
		UserID:    userID,
		PlanCode:  planCode,
		Status:    "active",
		StartedAt: now,
		ExpiresAt: &expiresAt,
		UpdatedAt: now,
	}
	if err := h.billing.UpsertMembership(ctx, membership); err != nil {
		return storage.Membership{}, err
	}
	return membership, nil
}

func membershipExpiry(start time.Time, planCode string) (time.Time, bool) {
	switch planCode {
	case "vip_monthly":
		return start.Add(30 * 24 * time.Hour), true
	case "vip_quarterly":
		return start.Add(90 * 24 * time.Hour), true
	case "vip_yearly":
		return start.Add(365 * 24 * time.Hour), true
	default:
		return time.Time{}, false
	}
}

func membershipResponse(membership storage.Membership) map[string]any {
	resp := map[string]any{
		"status":     membership.Status,
		"plan_code":  membership.PlanCode,
		"is_vip":     membership.Status == "active",
		"expires_at": nil,
	}
	if membership.ExpiresAt != nil {
		resp["expires_at"] = membership.ExpiresAt.UTC().Format(time.RFC3339Nano)
	}
	return resp
}

func (h *Handler) ensureVIPExportAccess(w http.ResponseWriter, r *http.Request) bool {
	if h.auth == nil || h.billing == nil {
		return true
	}
	user, ok := h.requireAuthenticatedUser(w, r)
	if !ok {
		return false
	}
	membership, err := h.billing.GetMembershipByUserID(r.Context(), user.ID)
	if errors.Is(err, storage.ErrMembershipNotFound) {
		writeError(w, http.StatusForbidden, "vip membership required")
		return false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load membership status")
		return false
	}
	if membership.Status != "active" {
		writeError(w, http.StatusForbidden, "vip membership required")
		return false
	}
	if membership.ExpiresAt != nil && !membership.ExpiresAt.After(h.now().UTC()) {
		writeError(w, http.StatusForbidden, "vip membership required")
		return false
	}
	return true
}

func (h *Handler) requireAuthenticatedUser(w http.ResponseWriter, r *http.Request) (storage.User, bool) {
	if h.auth == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return storage.User{}, false
	}
	cookie, err := r.Cookie(authCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return storage.User{}, false
	}
	user, err := h.auth.CurrentUser(r.Context(), cookie.Value)
	if errors.Is(err, storage.ErrAuthSessionNotFound) {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return storage.User{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load current user")
		return storage.User{}, false
	}
	return user, true
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
