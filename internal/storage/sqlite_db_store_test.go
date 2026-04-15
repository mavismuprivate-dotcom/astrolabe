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
