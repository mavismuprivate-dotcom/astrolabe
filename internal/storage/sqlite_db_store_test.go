package storage

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"astrolabe/internal/astrology"
)

func TestSQLiteStore_SaveGetAndListReports(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "reports.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	first := Report{
		ID:        "rpt_first",
		SessionID: "sess_a",
		CreatedAt: time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC),
		Response: astrology.NatalChartResponse{
			Meta: astrology.MetaInfo{
				Input: astrology.NormalizedInput{
					BirthDate:    "1990-01-01",
					BirthCity:    "Shanghai",
					BirthCountry: "China",
					Timezone:     "Asia/Shanghai",
				},
			},
			Reading: astrology.Reading{
				Summary: "first summary",
			},
		},
	}
	second := Report{
		ID:        "rpt_second",
		SessionID: "sess_a",
		CreatedAt: time.Date(2026, 4, 14, 11, 0, 0, 0, time.UTC),
		Response: astrology.NatalChartResponse{
			Meta: astrology.MetaInfo{
				Input: astrology.NormalizedInput{
					BirthDate:    "1992-08-16",
					BirthCity:    "Nanjing",
					BirthCountry: "China",
					Timezone:     "Asia/Shanghai",
				},
			},
			Reading: astrology.Reading{
				Summary: "second summary",
			},
		},
	}

	if err := store.SaveReport(context.Background(), first); err != nil {
		t.Fatalf("SaveReport(first) returned error: %v", err)
	}
	if err := store.SaveReport(context.Background(), second); err != nil {
		t.Fatalf("SaveReport(second) returned error: %v", err)
	}

	got, err := store.GetReport(context.Background(), second.ID, "sess_a")
	if err != nil {
		t.Fatalf("GetReport returned error: %v", err)
	}
	if got.ID != second.ID {
		t.Fatalf("expected ID %s, got %s", second.ID, got.ID)
	}
	if got.Response.Meta.Input.BirthCity != "Nanjing" {
		t.Fatalf("expected stored birth city Nanjing, got %s", got.Response.Meta.Input.BirthCity)
	}

	items, err := store.ListReports(context.Background(), "sess_a", 10)
	if err != nil {
		t.Fatalf("ListReports returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 reports, got %d", len(items))
	}
	if items[0].ID != second.ID {
		t.Fatalf("expected newest report first, got %s", items[0].ID)
	}
	if items[1].ID != first.ID {
		t.Fatalf("expected oldest report second, got %s", items[1].ID)
	}
}

func TestSQLiteStore_ScopesReportsBySessionID(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "reports.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	report := Report{
		ID:        "rpt_private",
		SessionID: "sess_owner",
		CreatedAt: time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC),
		Response: astrology.NatalChartResponse{
			Meta: astrology.MetaInfo{
				Input: astrology.NormalizedInput{
					BirthDate:    "1990-01-01",
					BirthCity:    "Shanghai",
					BirthCountry: "China",
				},
			},
		},
	}

	if err := store.SaveReport(context.Background(), report); err != nil {
		t.Fatalf("SaveReport returned error: %v", err)
	}

	if _, err := store.GetReport(context.Background(), report.ID, "sess_other"); !errors.Is(err, ErrReportNotFound) {
		t.Fatalf("expected ErrReportNotFound for other session, got %v", err)
	}

	items, err := store.ListReports(context.Background(), "sess_other", 10)
	if err != nil {
		t.Fatalf("ListReports returned error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected no reports for other session, got %d", len(items))
	}
}

func TestSQLiteStore_RecordsSchemaVersion(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "reports.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	var version int
	err = store.db.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&version)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("query schema_migrations returned error: %v", err)
	}
	if version < 1 {
		t.Fatalf("expected schema version >= 1, got %d", version)
	}
}

func TestSQLiteStore_AuthCodeAndSessionFlow(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "auth.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	ctx := context.Background()
	user, err := store.GetOrCreateUserByPhone(ctx, "13800138000")
	if err != nil {
		t.Fatalf("GetOrCreateUserByPhone returned error: %v", err)
	}
	if user.Phone != "13800138000" {
		t.Fatalf("expected phone 13800138000, got %s", user.Phone)
	}

	expiresAt := time.Date(2026, 4, 17, 9, 10, 0, 0, time.UTC)
	if err := store.SaveLoginCode(ctx, user.Phone, "hash_123456", expiresAt); err != nil {
		t.Fatalf("SaveLoginCode returned error: %v", err)
	}

	consumed, err := store.ConsumeLoginCode(ctx, user.Phone, "hash_123456", time.Date(2026, 4, 17, 9, 5, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ConsumeLoginCode returned error: %v", err)
	}
	if !consumed {
		t.Fatalf("expected login code to be consumed")
	}

	consumedAgain, err := store.ConsumeLoginCode(ctx, user.Phone, "hash_123456", time.Date(2026, 4, 17, 9, 6, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ConsumeLoginCode second call returned error: %v", err)
	}
	if consumedAgain {
		t.Fatalf("expected consumed login code to be invalid on second use")
	}

	session, err := store.CreateAuthSession(ctx, user.ID, time.Date(2026, 5, 17, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("CreateAuthSession returned error: %v", err)
	}
	if session.ID == "" {
		t.Fatalf("expected auth session ID")
	}

	resolvedUser, err := store.GetUserByAuthSession(ctx, session.ID, time.Date(2026, 4, 17, 9, 1, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("GetUserByAuthSession returned error: %v", err)
	}
	if resolvedUser.Phone != user.Phone {
		t.Fatalf("expected resolved phone %s, got %s", user.Phone, resolvedUser.Phone)
	}

	if err := store.DeleteAuthSession(ctx, session.ID); err != nil {
		t.Fatalf("DeleteAuthSession returned error: %v", err)
	}
	if _, err := store.GetUserByAuthSession(ctx, session.ID, time.Date(2026, 4, 17, 9, 2, 0, 0, time.UTC)); !errors.Is(err, ErrAuthSessionNotFound) {
		t.Fatalf("expected ErrAuthSessionNotFound after delete, got %v", err)
	}
}
