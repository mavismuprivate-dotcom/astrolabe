package astrology

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	errMissingDate   = errors.New("birth_date is required")
	birthDatePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

type planetDef struct {
	Name  string
	Base  float64
	Speed float64
}

var planetDefs = []planetDef{
	{Name: "Sun", Base: 280.466, Speed: 0.98564736},
	{Name: "Moon", Base: 218.316, Speed: 13.176396},
	{Name: "Mercury", Base: 252.251, Speed: 4.09233445},
	{Name: "Venus", Base: 181.980, Speed: 1.60213034},
	{Name: "Mars", Base: 355.433, Speed: 0.52402068},
	{Name: "Jupiter", Base: 34.351, Speed: 0.08308529},
	{Name: "Saturn", Base: 50.077, Speed: 0.03344414},
	{Name: "Uranus", Base: 314.055, Speed: 0.01172834},
	{Name: "Neptune", Base: 304.348, Speed: 0.00598103},
	{Name: "Pluto", Base: 238.929, Speed: 0.00396400},
}

var canonicalBodyOrder = []string{
	"Sun", "Moon", "Mercury", "Venus", "Mars", "Jupiter", "Saturn", "Uranus", "Neptune", "Pluto", "ASC", "MC",
}

type aspectRule struct {
	Name  string
	Exact float64
	Orb   float64
}

var majorAspectRules = []aspectRule{
	{Name: "conjunction", Exact: 0, Orb: 8},
	{Name: "sextile", Exact: 60, Orb: 4},
	{Name: "square", Exact: 90, Orb: 6},
	{Name: "trine", Exact: 120, Orb: 6},
	{Name: "opposition", Exact: 180, Orb: 8},
}

func ResolveBirthDateTime(req NatalChartRequest, timezone string) (BirthInstant, error) {
	if strings.TrimSpace(req.BirthDate) == "" {
		return BirthInstant{}, errMissingDate
	}
	if !birthDatePattern.MatchString(strings.TrimSpace(req.BirthDate)) {
		return BirthInstant{}, fmt.Errorf("invalid birth_date format, expected YYYY-MM-DD with 4-digit year")
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return BirthInstant{}, fmt.Errorf("invalid timezone: %w", err)
	}

	date, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(req.BirthDate), loc)
	if err != nil {
		return BirthInstant{}, fmt.Errorf("invalid birth_date format, expected YYYY-MM-DD")
	}

	approximate := strings.TrimSpace(req.BirthTime) == ""
	hour := 12
	minute := 0
	if !approximate {
		tm, err := time.Parse("15:04", strings.TrimSpace(req.BirthTime))
		if err != nil {
			return BirthInstant{}, fmt.Errorf("invalid birth_time format, expected HH:MM")
		}
		hour = tm.Hour()
		minute = tm.Minute()
	}

	local := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, loc)
	if local.Year() != date.Year() || local.Month() != date.Month() || local.Day() != date.Day() || local.Hour() != hour || local.Minute() != minute {
		return BirthInstant{}, fmt.Errorf("invalid local time for timezone/daylight-saving transition")
	}

	utc := local.UTC()
	jd := JulianDate(utc)
	return BirthInstant{
		Local:       local,
		UTC:         utc,
		JD:          round3(jd),
		Approximate: approximate,
		LocalHour:   local.Hour(),
		LocalMinute: local.Minute(),
	}, nil
}

func JulianDate(t time.Time) float64 {
	return float64(t.Unix())/86400.0 + 2440587.5
}

func CalculatePlanets(jd float64) map[string]float64 {
	days := jd - 2451545.0
	out := make(map[string]float64, len(planetDefs))
	for _, p := range planetDefs {
		lon := normalize360(p.Base + p.Speed*days)
		out[p.Name] = round3(lon)
	}
	return out
}

func CalculateAnglesAndHouses(jd, latitude, longitude float64) (asc float64, mc float64, cusps []float64) {
	lst := localSiderealDegrees(jd, longitude)
	eps := 23.439291
	theta := deg2rad(lst)
	phi := deg2rad(latitude)
	e := deg2rad(eps)

	mc = normalize360(rad2deg(math.Atan2(math.Sin(theta)*math.Cos(e), math.Cos(theta))))
	asc = normalize360(rad2deg(math.Atan2(-math.Cos(theta), math.Sin(theta)*math.Cos(e)+math.Tan(phi)*math.Sin(e))))

	cusps = make([]float64, 12)
	for i := 0; i < 12; i++ {
		cusps[i] = round3(normalize360(asc + float64(i)*30.0))
	}

	return round3(asc), round3(mc), cusps
}

func localSiderealDegrees(jd, longitude float64) float64 {
	gmst := 280.46061837 + 360.98564736629*(jd-2451545.0)
	return normalize360(gmst + longitude)
}

func ZodiacSign(longitude float64) string {
	signs := []string{"Aries", "Taurus", "Gemini", "Cancer", "Leo", "Virgo", "Libra", "Scorpio", "Sagittarius", "Capricorn", "Aquarius", "Pisces"}
	idx := int(math.Floor(normalize360(longitude) / 30.0))
	if idx < 0 {
		idx = 0
	}
	if idx > 11 {
		idx = 11
	}
	return signs[idx]
}

func HouseOfLongitude(longitude float64, cusps []float64) int {
	if len(cusps) != 12 {
		return 0
	}
	lon := normalize360(longitude)
	for i := 0; i < 12; i++ {
		start := normalize360(cusps[i])
		end := normalize360(cusps[(i+1)%12])
		if inArc(lon, start, end) {
			return i + 1
		}
	}
	return 12
}

func inArc(value, start, end float64) bool {
	if start <= end {
		return value >= start && value < end
	}
	return value >= start || value < end
}

func DetectMajorAspects(points map[string]float64) []Aspect {
	names := orderedBodies(points)
	result := make([]Aspect, 0)
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			a := names[i]
			b := names[j]
			delta := shortestAngleDistance(points[a], points[b])
			for _, r := range majorAspectRules {
				orb := math.Abs(delta - r.Exact)
				if orb <= r.Orb {
					result = append(result, Aspect{
						PairKey: a + "-" + b,
						BodyA:   a,
						BodyB:   b,
						Type:    r.Name,
						Exact:   r.Exact,
						Orb:     round3(orb),
					})
					break
				}
			}
		}
	}
	return result
}

func orderedBodies(points map[string]float64) []string {
	present := make(map[string]bool, len(points))
	for k := range points {
		present[k] = true
	}
	ordered := make([]string, 0, len(points))
	for _, name := range canonicalBodyOrder {
		if present[name] {
			ordered = append(ordered, name)
			delete(present, name)
		}
	}
	if len(present) > 0 {
		extra := make([]string, 0, len(present))
		for name := range present {
			extra = append(extra, name)
		}
		sort.Strings(extra)
		ordered = append(ordered, extra...)
	}
	return ordered
}

func shortestAngleDistance(a, b float64) float64 {
	d := math.Abs(normalize360(a) - normalize360(b))
	if d > 180 {
		return 360 - d
	}
	return d
}

func normalize360(v float64) float64 {
	v = math.Mod(v, 360)
	if v < 0 {
		v += 360
	}
	return v
}

func round3(v float64) float64 {
	return math.Round(v*1000) / 1000
}

func deg2rad(d float64) float64 {
	return d * math.Pi / 180.0
}

func rad2deg(r float64) float64 {
	return r * 180.0 / math.Pi
}
