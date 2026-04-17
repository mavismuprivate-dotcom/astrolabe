package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"astrolabe/internal/astrology"
)

var ErrReportNotFound = errors.New("report not found")
var ErrAuthSessionNotFound = errors.New("auth session not found")
var ErrUserNotFound = errors.New("user not found")
var ErrMembershipNotFound = errors.New("membership not found")

type Report struct {
	ID        string
	SessionID string
	CreatedAt time.Time
	Response  astrology.NatalChartResponse
}

type ReportSummary struct {
	ID          string    `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	BirthDate   string    `json:"birth_date"`
	BirthCity   string    `json:"birth_city"`
	BirthCountry string   `json:"birth_country"`
	Approximate bool      `json:"approximate"`
	Confidence  float64   `json:"confidence"`
}

type User struct {
	ID        string    `json:"id"`
	Phone     string    `json:"phone"`
	CreatedAt time.Time `json:"created_at"`
}

type AuthSession struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type Membership struct {
	UserID    string
	PlanCode  string
	Status    string
	StartedAt time.Time
	ExpiresAt *time.Time
	UpdatedAt time.Time
}

type PaymentOrder struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Provider  string     `json:"provider"`
	PlanCode  string     `json:"plan_code"`
	AmountCNY int64      `json:"amount_cny"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	PaidAt    *time.Time `json:"paid_at"`
}

type ReportStore interface {
	SaveReport(ctx context.Context, report Report) error
	GetReport(ctx context.Context, id string, sessionID string) (Report, error)
	ListReports(ctx context.Context, sessionID string, limit int) ([]ReportSummary, error)
	Close() error
}

type AuthStore interface {
	GetOrCreateUserByPhone(ctx context.Context, phone string) (User, error)
	SaveLoginCode(ctx context.Context, phone string, codeHash string, expiresAt time.Time) error
	ConsumeLoginCode(ctx context.Context, phone string, codeHash string, now time.Time) (bool, error)
	CreateAuthSession(ctx context.Context, userID string, expiresAt time.Time) (AuthSession, error)
	GetUserByAuthSession(ctx context.Context, sessionID string, now time.Time) (User, error)
	DeleteAuthSession(ctx context.Context, sessionID string) error
	GetMembershipByUserID(ctx context.Context, userID string) (Membership, error)
}

type BillingStore interface {
	UpsertMembership(ctx context.Context, membership Membership) error
	GetMembershipByUserID(ctx context.Context, userID string) (Membership, error)
	SavePaymentOrder(ctx context.Context, order PaymentOrder) error
	ListPaymentOrdersByUserID(ctx context.Context, userID string, limit int) ([]PaymentOrder, error)
}

func NewReportID() string {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return "rpt_" + hex.EncodeToString(buf[:])
}

func NewSessionID() string {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "sess_" + time.Now().UTC().Format("20060102150405.000000000")
	}
	return "sess_" + hex.EncodeToString(buf[:])
}

func NewUserID() string {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "usr_" + time.Now().UTC().Format("20060102150405.000000000")
	}
	return "usr_" + hex.EncodeToString(buf[:])
}

func NewAuthSessionID() string {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "auth_" + time.Now().UTC().Format("20060102150405.000000000")
	}
	return "auth_" + hex.EncodeToString(buf[:])
}

func NewPaymentOrderID() string {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "ord_" + time.Now().UTC().Format("20060102150405.000000000")
	}
	return "ord_" + hex.EncodeToString(buf[:])
}
