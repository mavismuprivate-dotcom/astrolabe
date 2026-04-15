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

type ReportStore interface {
	SaveReport(ctx context.Context, report Report) error
	GetReport(ctx context.Context, id string, sessionID string) (Report, error)
	ListReports(ctx context.Context, sessionID string, limit int) ([]ReportSummary, error)
	Close() error
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
