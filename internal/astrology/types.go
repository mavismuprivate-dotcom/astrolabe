package astrology

import "time"

// NatalChartRequest captures API input from web form.
type NatalChartRequest struct {
	BirthDate     string `json:"birth_date"`
	BirthTime     string `json:"birth_time,omitempty"`
	BirthProvince string `json:"birth_province,omitempty"`
	BirthCity     string `json:"birth_city"`
	BirthCountry  string `json:"birth_country"`
	Timezone      string `json:"timezone,omitempty"`
	Calendar      string `json:"calendar,omitempty"`
	HouseSystem   string `json:"house_system,omitempty"`
	ZodiacType    string `json:"zodiac_type,omitempty"`
	Language      string `json:"language,omitempty"`
}

type NatalChartResponse struct {
	ReportID string   `json:"report_id,omitempty"`
	Meta    MetaInfo `json:"meta"`
	Chart   Chart    `json:"chart"`
	Reading Reading  `json:"reading"`
}

type MetaInfo struct {
	Input        NormalizedInput `json:"input"`
	Approximate  bool            `json:"approximate"`
	Confidence   float64         `json:"confidence"`
	GeneratedAt  time.Time       `json:"generated_at"`
	Warnings     []string        `json:"warnings,omitempty"`
	ConfidenceCN string          `json:"confidence_note"`
}

type NormalizedInput struct {
	BirthDate     string  `json:"birth_date"`
	BirthTime     string  `json:"birth_time"`
	BirthProvince string  `json:"birth_province,omitempty"`
	BirthCity     string  `json:"birth_city"`
	BirthCountry  string  `json:"birth_country"`
	Timezone      string  `json:"timezone"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	Calendar      string  `json:"calendar"`
	HouseSystem   string  `json:"house_system"`
	ZodiacType    string  `json:"zodiac_type"`
	Language      string  `json:"language"`
}

type Chart struct {
	JD      float64                   `json:"jd"`
	Planets map[string]PlanetPosition `json:"planets"`
	Houses  []House                   `json:"houses"`
	Aspects []Aspect                  `json:"aspects"`
	Angles  map[string]AnglePoint     `json:"angles"`
}

type PlanetPosition struct {
	Longitude float64 `json:"longitude"`
	Sign      string  `json:"sign"`
	House     int     `json:"house"`
}

type House struct {
	Number int     `json:"number"`
	Cusp   float64 `json:"cusp"`
	Sign   string  `json:"sign"`
}

type AnglePoint struct {
	Longitude float64 `json:"longitude"`
	Sign      string  `json:"sign"`
}

type Aspect struct {
	PairKey string  `json:"pair_key"`
	BodyA   string  `json:"body_a"`
	BodyB   string  `json:"body_b"`
	Type    string  `json:"type"`
	Exact   float64 `json:"exact"`
	Orb     float64 `json:"orb"`
}

type Reading struct {
	Personality   string         `json:"personality"`
	Relationship  string         `json:"relationship"`
	Love          string         `json:"love"`
	Career        string         `json:"career"`
	Money         string         `json:"money"`
	Family        string         `json:"family"`
	Summary       string         `json:"summary"`
	Growth        string         `json:"growth"`
	Action        string         `json:"action"`
	Focus         string         `json:"focus"`
	Evidence      []EvidenceItem `json:"evidence,omitempty"`
	Quality       QualityMetrics `json:"quality"`
	Reminder      string         `json:"reminder"`
	Disclaimer    string         `json:"disclaimer"`
	TextReport    string         `json:"text_report"`
	Entertainment string         `json:"entertainment"`
}

type EvidenceItem struct {
	Theme      string   `json:"theme"`
	Claim      string   `json:"claim"`
	Factors    []string `json:"factors"`
	Confidence float64  `json:"confidence"`
}

type QualityMetrics struct {
	CharCount        int     `json:"char_count"`
	DuplicateRatio   float64 `json:"duplicate_ratio"`
	ThemeCoverage    float64 `json:"theme_coverage"`
	ConsistencyScore float64 `json:"consistency_score"`
}

type Location struct {
	City      string
	Country   string
	Latitude  float64
	Longitude float64
	Timezone  string
}

type BirthInstant struct {
	Local       time.Time
	UTC         time.Time
	JD          float64
	Approximate bool
	LocalHour   int
	LocalMinute int
}
