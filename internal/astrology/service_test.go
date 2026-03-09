package astrology

import (
	"context"
	"encoding/json"
	"testing"
)

func TestResolveBirthDateTime_ApproximateNoonWhenTimeMissing(t *testing.T) {
	req := NatalChartRequest{
		BirthDate:    "1990-01-01",
		BirthCity:    "Shanghai",
		BirthCountry: "China",
	}

	instant, err := ResolveBirthDateTime(req, "Asia/Shanghai")
	if err != nil {
		t.Fatalf("ResolveBirthDateTime returned error: %v", err)
	}

	if !instant.Approximate {
		t.Fatalf("expected Approximate=true when birth_time missing")
	}
	if instant.LocalHour != 12 || instant.LocalMinute != 0 {
		t.Fatalf("expected fallback local time 12:00, got %02d:%02d", instant.LocalHour, instant.LocalMinute)
	}
}

func TestResolveBirthDateTime_InvalidDSTTimeReturnsError(t *testing.T) {
	req := NatalChartRequest{
		BirthDate:    "2024-03-10",
		BirthTime:    "02:30",
		BirthCity:    "New York",
		BirthCountry: "United States",
	}

	_, err := ResolveBirthDateTime(req, "America/New_York")
	if err == nil {
		t.Fatalf("expected error for nonexistent local DST time")
	}
}

func TestResolveBirthDateTime_RejectsFiveDigitYear(t *testing.T) {
	req := NatalChartRequest{
		BirthDate:    "10000-01-01",
		BirthTime:    "08:00",
		BirthCity:    "南京",
		BirthCountry: "中国",
	}

	_, err := ResolveBirthDateTime(req, "Asia/Shanghai")
	if err == nil {
		t.Fatalf("expected error for 5-digit year")
	}
}

func TestDetectMajorAspects(t *testing.T) {
	points := map[string]float64{
		"Sun":     0,
		"Moon":    120,
		"Mars":    180,
		"Venus":   60,
		"Mercury": 90,
	}

	aspects := DetectMajorAspects(points)

	expects := map[string]string{
		"Sun-Moon":    "trine",
		"Sun-Mars":    "opposition",
		"Sun-Venus":   "sextile",
		"Sun-Mercury": "square",
	}

	for pair, kind := range expects {
		found := false
		for _, asp := range aspects {
			if asp.PairKey == pair && asp.Type == kind {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected aspect %s=%s not found", pair, kind)
		}
	}
}

func TestSignAndHouseMapping(t *testing.T) {
	if got := ZodiacSign(0); got != "Aries" {
		t.Fatalf("expected Aries at 0 deg, got %s", got)
	}
	if got := ZodiacSign(359.9); got != "Pisces" {
		t.Fatalf("expected Pisces at 359.9 deg, got %s", got)
	}

	cusps := []float64{0, 30, 60, 90, 120, 150, 180, 210, 240, 270, 300, 330}
	if got := HouseOfLongitude(45, cusps); got != 2 {
		t.Fatalf("expected house 2 for 45 deg, got %d", got)
	}
	if got := HouseOfLongitude(350, cusps); got != 12 {
		t.Fatalf("expected house 12 for 350 deg, got %d", got)
	}
}

func TestServiceSnapshotStableForJ2000Input(t *testing.T) {
	svc := NewService(NewCityResolver())
	req := NatalChartRequest{
		BirthDate:    "2000-01-01",
		BirthTime:    "12:00",
		BirthCity:    "London",
		BirthCountry: "United Kingdom",
		Timezone:     "UTC",
		Language:     "zh-CN",
	}

	got, err := svc.GenerateNatalChart(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateNatalChart returned error: %v", err)
	}

	if got.Chart.Planets["Sun"].Longitude != 280.466 {
		t.Fatalf("expected Sun longitude 280.466 for J2000 snapshot, got %.3f", got.Chart.Planets["Sun"].Longitude)
	}

	buf, err := json.Marshal(got.Chart)
	if err != nil {
		t.Fatalf("json marshal failed: %v", err)
	}
	if len(buf) < 500 {
		t.Fatalf("expected rich chart snapshot payload, got %d bytes", len(buf))
	}
}

func TestServiceSupportsChineseNanjingWithDefaultChinaTimezone(t *testing.T) {
	svc := NewService(NewCityResolver())
	req := NatalChartRequest{
		BirthDate:    "1992-08-16",
		BirthTime:    "09:20",
		BirthCity:    "NanJing",
		BirthCountry: "中国",
		Language:     "zh-CN",
	}

	resp, err := svc.GenerateNatalChart(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateNatalChart returned error: %v", err)
	}

	if resp.Meta.Input.Timezone != "Asia/Shanghai" {
		t.Fatalf("expected timezone Asia/Shanghai, got %s", resp.Meta.Input.Timezone)
	}
	if len(resp.Chart.Houses) != 12 {
		t.Fatalf("expected 12 houses, got %d", len(resp.Chart.Houses))
	}
}

func TestServiceSupportsProvinceInputAndDefaultCountry(t *testing.T) {
	svc := NewService(NewCityResolver())
	req := NatalChartRequest{
		BirthDate:     "1995-06-21",
		BirthTime:     "14:40",
		BirthProvince: "江苏省",
		Language:      "zh-CN",
	}

	resp, err := svc.GenerateNatalChart(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateNatalChart returned error: %v", err)
	}

	if resp.Meta.Input.BirthProvince != "江苏省" {
		t.Fatalf("expected province 江苏省, got %s", resp.Meta.Input.BirthProvince)
	}
	if resp.Meta.Input.BirthCountry != "中国" {
		t.Fatalf("expected default country 中国, got %s", resp.Meta.Input.BirthCountry)
	}
	if resp.Meta.Input.Timezone != "Asia/Shanghai" {
		t.Fatalf("expected timezone Asia/Shanghai, got %s", resp.Meta.Input.Timezone)
	}
}

func TestServiceReadingIsExpanded(t *testing.T) {
	svc := NewService(NewCityResolver())
	req := NatalChartRequest{
		BirthDate:    "1990-01-01",
		BirthTime:    "08:15",
		BirthCity:    "南京",
		BirthCountry: "中国",
		Language:     "zh-CN",
	}

	resp, err := svc.GenerateNatalChart(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateNatalChart returned error: %v", err)
	}

	if len([]rune(resp.Reading.TextReport)) < 280 {
		t.Fatalf("expected expanded text report, got %d runes", len([]rune(resp.Reading.TextReport)))
	}
	if resp.Reading.Growth == "" || resp.Reading.Action == "" {
		t.Fatalf("expected growth/action sections to be populated")
	}
	if resp.Reading.Money == "" {
		t.Fatalf("expected money section to be populated")
	}
	if resp.Reading.Love == "" {
		t.Fatalf("expected love section to be populated")
	}
}
