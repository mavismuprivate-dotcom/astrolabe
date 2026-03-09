package astrology

import "testing"

func TestCityResolver_ResolveNanjingAliases(t *testing.T) {
	r := NewCityResolver()

	cases := []struct {
		city    string
		country string
	}{
		{city: "NanJing", country: "China"},
		{city: "NAN JING", country: "CN"},
		{city: "南京", country: "中国"},
		{city: "南京市", country: "中国"},
	}

	for _, tc := range cases {
		loc, ok := r.Resolve(tc.city, tc.country)
		if !ok {
			t.Fatalf("expected resolver to support %q/%q", tc.city, tc.country)
		}
		if loc.Timezone != "Asia/Shanghai" {
			t.Fatalf("expected Asia/Shanghai, got %s", loc.Timezone)
		}
	}
}

func TestCityResolver_ResolveChineseProvince(t *testing.T) {
	r := NewCityResolver()

	cases := []string{"江苏", "江苏省", "Jiangsu", "广西壮族自治区", "香港特别行政区", "台湾省"}
	for _, province := range cases {
		loc, ok := r.Resolve(province, "中国")
		if !ok {
			t.Fatalf("expected resolver to support province %q", province)
		}
		if loc.Timezone == "" {
			t.Fatalf("expected timezone for province %q", province)
		}
	}
}
