package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"astrolabe/internal/astrology"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *SQLiteStore) migrate() error {
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
	version INTEGER PRIMARY KEY,
	name TEXT NOT NULL,
	applied_at TEXT NOT NULL
);
`); err != nil {
		return err
	}

	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	for _, m := range migrations {
		var exists int
		if err := s.db.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE version = ?`, m.version).Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			continue
		}

		tx, err := s.db.Begin()
		if err != nil {
			return err
		}

		if _, err := tx.Exec(m.sql); err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err := tx.Exec(
			`INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`,
			m.version,
			m.name,
			time.Now().UTC().Format(time.RFC3339Nano),
		); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}

func (s *SQLiteStore) SaveReport(ctx context.Context, report Report) error {
	if report.ID == "" {
		report.ID = NewReportID()
	}
	if report.CreatedAt.IsZero() {
		report.CreatedAt = time.Now().UTC()
	}
	report.Response.ReportID = report.ID

	responseJSON, err := json.Marshal(report.Response)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO reports (id, session_id, created_at, response_json, birth_date, birth_city, birth_country, approximate, confidence)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   session_id=excluded.session_id,
		   created_at=excluded.created_at,
		   response_json=excluded.response_json,
		   birth_date=excluded.birth_date,
		   birth_city=excluded.birth_city,
		   birth_country=excluded.birth_country,
		   approximate=excluded.approximate,
		   confidence=excluded.confidence`,
		report.ID,
		report.SessionID,
		report.CreatedAt.UTC().Format(time.RFC3339Nano),
		string(responseJSON),
		report.Response.Meta.Input.BirthDate,
		report.Response.Meta.Input.BirthCity,
		report.Response.Meta.Input.BirthCountry,
		boolToInt(report.Response.Meta.Approximate),
		report.Response.Meta.Confidence,
	)
	return err
}

func (s *SQLiteStore) GetReport(ctx context.Context, id string, sessionID string) (Report, error) {
	var (
		createdAtRaw string
		responseJSON string
	)

	err := s.db.QueryRowContext(
		ctx,
		`SELECT created_at, response_json
		 FROM reports
		 WHERE id = ? AND session_id = ?`,
		id,
		sessionID,
	).Scan(&createdAtRaw, &responseJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return Report{}, ErrReportNotFound
	}
	if err != nil {
		return Report{}, err
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return Report{}, err
	}

	var response astrology.NatalChartResponse
	if err := json.Unmarshal([]byte(responseJSON), &response); err != nil {
		return Report{}, err
	}
	if response.ReportID == "" {
		response.ReportID = id
	}

	return Report{
		ID:        id,
		SessionID: sessionID,
		CreatedAt: createdAt,
		Response:  response,
	}, nil
}

func (s *SQLiteStore) ListReports(ctx context.Context, sessionID string, limit int) ([]ReportSummary, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, created_at, birth_date, birth_city, birth_country, approximate, confidence
		 FROM reports
		 WHERE session_id = ?
		 ORDER BY datetime(created_at) DESC, id DESC
		 LIMIT ?`,
		sessionID,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ReportSummary, 0, limit)
	for rows.Next() {
		var (
			item         ReportSummary
			createdAtRaw string
			approximate  int
		)
		if err := rows.Scan(
			&item.ID,
			&createdAtRaw,
			&item.BirthDate,
			&item.BirthCity,
			&item.BirthCountry,
			&approximate,
			&item.Confidence,
		); err != nil {
			return nil, err
		}
		item.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAtRaw)
		if err != nil {
			return nil, err
		}
		item.Approximate = approximate == 1
		items = append(items, item)
	}

	return items, rows.Err()
}

func (s *SQLiteStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
