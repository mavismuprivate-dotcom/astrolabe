package storage

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type FileStore struct {
	path string
	mu   sync.Mutex
}

func NewFileStore(path string) (*FileStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	store := &FileStore{path: path}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte("[]"), 0o644); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	return store, nil
}

func (s *FileStore) SaveReport(_ context.Context, report Report) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if report.ID == "" {
		report.ID = NewReportID()
	}
	if report.CreatedAt.IsZero() {
		report.CreatedAt = time.Now().UTC()
	}
	report.Response.ReportID = report.ID

	reports, err := s.readAllLocked()
	if err != nil {
		return err
	}

	replaced := false
	for i := range reports {
		if reports[i].ID == report.ID {
			reports[i] = report
			replaced = true
			break
		}
	}
	if !replaced {
		reports = append(reports, report)
	}

	return s.writeAllLocked(reports)
}

func (s *FileStore) GetReport(_ context.Context, id string, sessionID string) (Report, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	reports, err := s.readAllLocked()
	if err != nil {
		return Report{}, err
	}

	for _, report := range reports {
		if report.ID == id && report.SessionID == sessionID {
			if report.Response.ReportID == "" {
				report.Response.ReportID = report.ID
			}
			return report, nil
		}
	}

	return Report{}, ErrReportNotFound
}

func (s *FileStore) ListReports(_ context.Context, sessionID string, limit int) ([]ReportSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	reports, err := s.readAllLocked()
	if err != nil {
		return nil, err
	}

	filtered := make([]Report, 0, len(reports))
	for _, report := range reports {
		if report.SessionID == sessionID {
			filtered = append(filtered, report)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].CreatedAt.Equal(filtered[j].CreatedAt) {
			return filtered[i].ID > filtered[j].ID
		}
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})

	if limit <= 0 || limit > len(filtered) {
		limit = len(filtered)
	}

	items := make([]ReportSummary, 0, limit)
	for _, report := range filtered[:limit] {
		items = append(items, ReportSummary{
			ID:           report.ID,
			CreatedAt:    report.CreatedAt,
			BirthDate:    report.Response.Meta.Input.BirthDate,
			BirthCity:    report.Response.Meta.Input.BirthCity,
			BirthCountry: report.Response.Meta.Input.BirthCountry,
			Approximate:  report.Response.Meta.Approximate,
			Confidence:   report.Response.Meta.Confidence,
		})
	}

	return items, nil
}

func (s *FileStore) Close() error {
	return nil
}

func (s *FileStore) readAllLocked() ([]Report, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return []Report{}, nil
	}

	var reports []Report
	if err := json.Unmarshal(data, &reports); err != nil {
		return nil, err
	}
	return reports, nil
}

func (s *FileStore) writeAllLocked(reports []Report) error {
	data, err := json.Marshal(reports)
	if err != nil {
		return err
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.path)
}
