package astrology

import (
	"context"
	"encoding/json"
	"strings"
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

	if len([]rune(resp.Reading.TextReport)) < 1200 {
		t.Fatalf("expected long text report, got %d runes", len([]rune(resp.Reading.TextReport)))
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
	if resp.Reading.Family == "" || resp.Reading.Summary == "" {
		t.Fatalf("expected family/summary sections to be populated")
	}
	if len(resp.Reading.Evidence) < 8 {
		t.Fatalf("expected evidence items >= 8, got %d", len(resp.Reading.Evidence))
	}
	if resp.Reading.Quality.CharCount < 1200 {
		t.Fatalf("expected quality char_count >=1200, got %d", resp.Reading.Quality.CharCount)
	}
	if resp.Reading.Quality.ThemeCoverage < 1.0 {
		t.Fatalf("expected full theme coverage, got %.2f", resp.Reading.Quality.ThemeCoverage)
	}
	if resp.Reading.Quality.ConsistencyScore <= 0.5 {
		t.Fatalf("expected consistency score >0.5, got %.2f", resp.Reading.Quality.ConsistencyScore)
	}
}

func TestExtractThemeSignals_CoversPrimaryThemes(t *testing.T) {
	planets := map[string]PlanetPosition{
		"Sun":     {Sign: "Aries", House: 10},
		"Moon":    {Sign: "Cancer", House: 4},
		"Venus":   {Sign: "Libra", House: 7},
		"Mars":    {Sign: "Capricorn", House: 2},
		"Jupiter": {Sign: "Taurus", House: 2},
		"Saturn":  {Sign: "Virgo", House: 6},
	}
	aspects := []Aspect{
		{BodyA: "Venus", BodyB: "Moon", Type: "square", Orb: 2.1},
		{BodyA: "Mars", BodyB: "Jupiter", Type: "trine", Orb: 1.6},
	}

	signals := extractThemeSignals(planets, aspects)
	themes := []string{"love", "career", "money", "family"}
	for _, theme := range themes {
		sig, ok := signals[theme]
		if !ok {
			t.Fatalf("expected signal for theme %s", theme)
		}
		if sig.Score <= 0 {
			t.Fatalf("expected positive score for theme %s", theme)
		}
		if len(sig.Factors) < 2 {
			t.Fatalf("expected >=2 factors for theme %s, got %d", theme, len(sig.Factors))
		}
	}
}

func TestAssessReadingQuality_DetectsHighDuplicateRatio(t *testing.T) {
	repeated := "相同句子。相同句子。相同句子。相同句子。"
	reading := Reading{
		Love:        repeated,
		Career:      repeated,
		Money:       repeated,
		Family:      repeated,
		TextReport:  repeated + repeated + repeated,
		Summary:     "摘要",
		Evidence:    []EvidenceItem{{Theme: "love", Claim: "test", Factors: []string{"金星7宫", "月亮4宫"}, Confidence: 0.8}},
		Quality:     QualityMetrics{},
		Personality: "人格",
	}

	quality := assessReadingQuality(reading)
	if quality.DuplicateRatio < 0.3 {
		t.Fatalf("expected duplicate ratio >= 0.3, got %.2f", quality.DuplicateRatio)
	}
}

func TestBuildEvidenceFromSignal_ClaimNotTruncated(t *testing.T) {
	claim := "你在爱情关系中会把情感安全与长期成长并重，且会在关系节奏里持续进行协商和复盘，以建立稳定信任。"
	sig := themeSignal{
		Theme:   "love",
		Score:   3.8,
		Factors: []string{"金星双鱼座第9宫", "月亮巨蟹座第4宫", "火星天秤座第7宫"},
	}

	item := buildEvidenceFromSignal("love", claim, sig, 0)
	if item.Claim != claim {
		t.Fatalf("expected full claim output, got %q", item.Claim)
	}
	if strings.Contains(item.Claim, "…") {
		t.Fatalf("expected claim without truncation ellipsis, got %q", item.Claim)
	}
}
