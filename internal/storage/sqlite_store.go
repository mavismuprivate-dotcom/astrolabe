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

func (s *SQLiteStore) GetOrCreateUserByPhone(ctx context.Context, phone string) (User, error) {
	now := time.Now().UTC()
	user := User{
		ID:        NewUserID(),
		Phone:     phone,
		CreatedAt: now,
	}

	if _, err := s.db.ExecContext(
		ctx,
		`INSERT INTO users (id, phone, created_at) VALUES (?, ?, ?)
		 ON CONFLICT(phone) DO NOTHING`,
		user.ID,
		user.Phone,
		user.CreatedAt.Format(time.RFC3339Nano),
	); err != nil {
		return User{}, err
	}

	var createdAtRaw string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, phone, created_at FROM users WHERE phone = ?`,
		phone,
	).Scan(&user.ID, &user.Phone, &createdAtRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, err
	}
	user.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *SQLiteStore) SaveLoginCode(ctx context.Context, phone string, codeHash string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO login_codes (phone, code_hash, expires_at, consumed_at, created_at)
		 VALUES (?, ?, ?, NULL, ?)
		 ON CONFLICT(phone) DO UPDATE SET
		   code_hash=excluded.code_hash,
		   expires_at=excluded.expires_at,
		   consumed_at=NULL,
		   created_at=excluded.created_at`,
		phone,
		codeHash,
		expiresAt.UTC().Format(time.RFC3339Nano),
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (s *SQLiteStore) ConsumeLoginCode(ctx context.Context, phone string, codeHash string, now time.Time) (bool, error) {
	result, err := s.db.ExecContext(
		ctx,
		`UPDATE login_codes
		 SET consumed_at = ?
		 WHERE phone = ?
		   AND code_hash = ?
		   AND consumed_at IS NULL
		   AND datetime(expires_at) >= datetime(?)`,
		now.UTC().Format(time.RFC3339Nano),
		phone,
		codeHash,
		now.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (s *SQLiteStore) CreateAuthSession(ctx context.Context, userID string, expiresAt time.Time) (AuthSession, error) {
	session := AuthSession{
		ID:        NewAuthSessionID(),
		UserID:    userID,
		ExpiresAt: expiresAt.UTC(),
		CreatedAt: time.Now().UTC(),
	}
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO auth_sessions (id, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)`,
		session.ID,
		session.UserID,
		session.ExpiresAt.Format(time.RFC3339Nano),
		session.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return AuthSession{}, err
	}
	return session, nil
}

func (s *SQLiteStore) GetUserByAuthSession(ctx context.Context, sessionID string, now time.Time) (User, error) {
	var (
		user         User
		createdAtRaw string
	)
	err := s.db.QueryRowContext(
		ctx,
		`SELECT users.id, users.phone, users.created_at
		 FROM auth_sessions
		 INNER JOIN users ON users.id = auth_sessions.user_id
		 WHERE auth_sessions.id = ?
		   AND datetime(auth_sessions.expires_at) >= datetime(?)`,
		sessionID,
		now.UTC().Format(time.RFC3339Nano),
	).Scan(&user.ID, &user.Phone, &createdAtRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrAuthSessionNotFound
	}
	if err != nil {
		return User{}, err
	}
	user.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *SQLiteStore) DeleteAuthSession(ctx context.Context, sessionID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM auth_sessions WHERE id = ?`, sessionID)
	return err
}

func (s *SQLiteStore) UpsertMembership(ctx context.Context, membership Membership) error {
	var expiresAt any
	if membership.ExpiresAt != nil {
		expiresAt = membership.ExpiresAt.UTC().Format(time.RFC3339Nano)
	}
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO memberships (user_id, plan_code, status, started_at, expires_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET
		   plan_code=excluded.plan_code,
		   status=excluded.status,
		   started_at=excluded.started_at,
		   expires_at=excluded.expires_at,
		   updated_at=excluded.updated_at`,
		membership.UserID,
		membership.PlanCode,
		membership.Status,
		membership.StartedAt.UTC().Format(time.RFC3339Nano),
		expiresAt,
		membership.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (s *SQLiteStore) GetMembershipByUserID(ctx context.Context, userID string) (Membership, error) {
	var (
		membership Membership
		startedAtRaw string
		expiresAtRaw sql.NullString
		updatedAtRaw string
	)
	err := s.db.QueryRowContext(
		ctx,
		`SELECT user_id, plan_code, status, started_at, expires_at, updated_at
		 FROM memberships
		 WHERE user_id = ?`,
		userID,
	).Scan(
		&membership.UserID,
		&membership.PlanCode,
		&membership.Status,
		&startedAtRaw,
		&expiresAtRaw,
		&updatedAtRaw,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Membership{}, ErrMembershipNotFound
	}
	if err != nil {
		return Membership{}, err
	}
	var errParse error
	membership.StartedAt, errParse = time.Parse(time.RFC3339Nano, startedAtRaw)
	if errParse != nil {
		return Membership{}, errParse
	}
	membership.UpdatedAt, errParse = time.Parse(time.RFC3339Nano, updatedAtRaw)
	if errParse != nil {
		return Membership{}, errParse
	}
	if expiresAtRaw.Valid && expiresAtRaw.String != "" {
		expiresAt, errParse := time.Parse(time.RFC3339Nano, expiresAtRaw.String)
		if errParse != nil {
			return Membership{}, errParse
		}
		membership.ExpiresAt = &expiresAt
	}
	return membership, nil
}

func (s *SQLiteStore) SavePaymentOrder(ctx context.Context, order PaymentOrder) error {
	var paidAt any
	if order.PaidAt != nil {
		paidAt = order.PaidAt.UTC().Format(time.RFC3339Nano)
	}
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO payment_orders (id, user_id, provider, plan_code, amount_cny, status, created_at, paid_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   provider=excluded.provider,
		   plan_code=excluded.plan_code,
		   amount_cny=excluded.amount_cny,
		   status=excluded.status,
		   created_at=excluded.created_at,
		   paid_at=excluded.paid_at`,
		order.ID,
		order.UserID,
		order.Provider,
		order.PlanCode,
		order.AmountCNY,
		order.Status,
		order.CreatedAt.UTC().Format(time.RFC3339Nano),
		paidAt,
	)
	return err
}

func (s *SQLiteStore) ListPaymentOrdersByUserID(ctx context.Context, userID string, limit int) ([]PaymentOrder, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, user_id, provider, plan_code, amount_cny, status, created_at, paid_at
		 FROM payment_orders
		 WHERE user_id = ?
		 ORDER BY datetime(created_at) DESC, id DESC
		 LIMIT ?`,
		userID,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]PaymentOrder, 0, limit)
	for rows.Next() {
		var (
			item PaymentOrder
			createdAtRaw string
			paidAtRaw sql.NullString
		)
		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.Provider,
			&item.PlanCode,
			&item.AmountCNY,
			&item.Status,
			&createdAtRaw,
			&paidAtRaw,
		); err != nil {
			return nil, err
		}
		item.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAtRaw)
		if err != nil {
			return nil, err
		}
		if paidAtRaw.Valid && paidAtRaw.String != "" {
			paidAt, err := time.Parse(time.RFC3339Nano, paidAtRaw.String)
			if err != nil {
				return nil, err
			}
			item.PaidAt = &paidAt
		}
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
