package astrology

import (
	"strings"
	"unicode"
)

type CityResolver struct {
	cities map[string]Location
}

func NewCityResolver() *CityResolver {
	m := make(map[string]Location)

	add := func(cityKey, countryKey, cityName, countryName string, lat, lon float64, tz string) {
		m[keyFor(cityKey, countryKey)] = Location{
			City:      cityName,
			Country:   countryName,
			Latitude:  lat,
			Longitude: lon,
			Timezone:  tz,
		}
	}

	// Core cities for existing compatibility.
	add("nanjing", "china", "南京", "中国", 32.0603, 118.7969, "Asia/Shanghai")
	add("shanghai", "china", "上海", "中国", 31.2304, 121.4737, "Asia/Shanghai")
	add("beijing", "china", "北京", "中国", 39.9042, 116.4074, "Asia/Shanghai")
	add("shenzhen", "china", "深圳", "中国", 22.5431, 114.0579, "Asia/Shanghai")
	add("newyork", "unitedstates", "New York", "United States", 40.7128, -74.0060, "America/New_York")
	add("losangeles", "unitedstates", "Los Angeles", "United States", 34.0522, -118.2437, "America/Los_Angeles")
	add("sanfrancisco", "unitedstates", "San Francisco", "United States", 37.7749, -122.4194, "America/Los_Angeles")
	add("london", "unitedkingdom", "London", "United Kingdom", 51.5074, -0.1278, "Europe/London")
	add("paris", "france", "Paris", "France", 48.8566, 2.3522, "Europe/Paris")
	add("tokyo", "japan", "Tokyo", "Japan", 35.6762, 139.6503, "Asia/Tokyo")
	add("singapore", "singapore", "Singapore", "Singapore", 1.3521, 103.8198, "Asia/Singapore")

	// China province-level divisions (capital coordinates as representative points).
	chinaProvinces := []struct {
		key  string
		name string
		lat  float64
		lon  float64
		tz   string
	}{
		{key: "beijing", name: "北京市", lat: 39.9042, lon: 116.4074, tz: "Asia/Shanghai"},
		{key: "tianjin", name: "天津市", lat: 39.3434, lon: 117.3616, tz: "Asia/Shanghai"},
		{key: "hebei", name: "河北省", lat: 38.0428, lon: 114.5149, tz: "Asia/Shanghai"},
		{key: "shanxi", name: "山西省", lat: 37.8706, lon: 112.5489, tz: "Asia/Shanghai"},
		{key: "innermongolia", name: "内蒙古自治区", lat: 40.8426, lon: 111.7492, tz: "Asia/Shanghai"},
		{key: "liaoning", name: "辽宁省", lat: 41.8057, lon: 123.4315, tz: "Asia/Shanghai"},
		{key: "jilin", name: "吉林省", lat: 43.8171, lon: 125.3235, tz: "Asia/Shanghai"},
		{key: "heilongjiang", name: "黑龙江省", lat: 45.8038, lon: 126.5349, tz: "Asia/Shanghai"},
		{key: "shanghai", name: "上海市", lat: 31.2304, lon: 121.4737, tz: "Asia/Shanghai"},
		{key: "jiangsu", name: "江苏省", lat: 32.0603, lon: 118.7969, tz: "Asia/Shanghai"},
		{key: "zhejiang", name: "浙江省", lat: 30.2741, lon: 120.1551, tz: "Asia/Shanghai"},
		{key: "anhui", name: "安徽省", lat: 31.8206, lon: 117.2272, tz: "Asia/Shanghai"},
		{key: "fujian", name: "福建省", lat: 26.0745, lon: 119.2965, tz: "Asia/Shanghai"},
		{key: "jiangxi", name: "江西省", lat: 28.6829, lon: 115.8582, tz: "Asia/Shanghai"},
		{key: "shandong", name: "山东省", lat: 36.6512, lon: 117.1201, tz: "Asia/Shanghai"},
		{key: "henan", name: "河南省", lat: 34.7466, lon: 113.6254, tz: "Asia/Shanghai"},
		{key: "hubei", name: "湖北省", lat: 30.5928, lon: 114.3055, tz: "Asia/Shanghai"},
		{key: "hunan", name: "湖南省", lat: 28.2282, lon: 112.9388, tz: "Asia/Shanghai"},
		{key: "guangdong", name: "广东省", lat: 23.1291, lon: 113.2644, tz: "Asia/Shanghai"},
		{key: "guangxi", name: "广西壮族自治区", lat: 22.8170, lon: 108.3669, tz: "Asia/Shanghai"},
		{key: "hainan", name: "海南省", lat: 20.0440, lon: 110.1983, tz: "Asia/Shanghai"},
		{key: "chongqing", name: "重庆市", lat: 29.5630, lon: 106.5516, tz: "Asia/Shanghai"},
		{key: "sichuan", name: "四川省", lat: 30.5728, lon: 104.0668, tz: "Asia/Shanghai"},
		{key: "guizhou", name: "贵州省", lat: 26.6470, lon: 106.6302, tz: "Asia/Shanghai"},
		{key: "yunnan", name: "云南省", lat: 24.8801, lon: 102.8329, tz: "Asia/Shanghai"},
		{key: "xizang", name: "西藏自治区", lat: 29.6520, lon: 91.1721, tz: "Asia/Shanghai"},
		{key: "shaanxi", name: "陕西省", lat: 34.3416, lon: 108.9398, tz: "Asia/Shanghai"},
		{key: "gansu", name: "甘肃省", lat: 36.0611, lon: 103.8343, tz: "Asia/Shanghai"},
		{key: "qinghai", name: "青海省", lat: 36.6171, lon: 101.7782, tz: "Asia/Shanghai"},
		{key: "ningxia", name: "宁夏回族自治区", lat: 38.4872, lon: 106.2309, tz: "Asia/Shanghai"},
		{key: "xinjiang", name: "新疆维吾尔自治区", lat: 43.8256, lon: 87.6168, tz: "Asia/Shanghai"},
		{key: "hongkong", name: "香港特别行政区", lat: 22.3193, lon: 114.1694, tz: "Asia/Hong_Kong"},
		{key: "macau", name: "澳门特别行政区", lat: 22.1987, lon: 113.5439, tz: "Asia/Macau"},
		{key: "taiwan", name: "台湾省", lat: 25.0330, lon: 121.5654, tz: "Asia/Taipei"},
	}

	for _, p := range chinaProvinces {
		add(p.key, "china", p.name, "中国", p.lat, p.lon, p.tz)
	}

	return &CityResolver{cities: m}
}

func (r *CityResolver) Resolve(city, country string) (Location, bool) {
	loc, ok := r.cities[keyFor(city, country)]
	if ok && loc.Timezone == "" {
		loc.Timezone = defaultTimezoneForCountry(country)
	}
	return loc, ok
}

func keyFor(city, country string) string {
	return canonicalCity(city) + "|" + canonicalCountry(country)
}

func defaultTimezoneForCountry(country string) string {
	if canonicalCountry(country) == "china" {
		return "Asia/Shanghai"
	}
	return ""
}

func canonicalCountry(raw string) string {
	token := normalizeToken(raw)
	switch token {
	case "", "china", "cn", "zhongguo", "中国", "中华人民共和国":
		return "china"
	case "unitedstates", "usa", "us", "美国":
		return "unitedstates"
	case "unitedkingdom", "uk", "gb", "英国":
		return "unitedkingdom"
	case "france", "法国":
		return "france"
	case "japan", "jp", "日本":
		return "japan"
	case "singapore", "新加坡":
		return "singapore"
	default:
		return token
	}
}

func canonicalCity(raw string) string {
	token := normalizeToken(raw)
	switch token {
	case "shanghai", "上海":
		return "shanghai"
	case "beijing", "北京":
		return "beijing"
	case "nanjing", "南京":
		return "nanjing"
	case "shenzhen", "深圳":
		return "shenzhen"
	case "newyork", "纽约":
		return "newyork"
	case "losangeles", "洛杉矶":
		return "losangeles"
	case "sanfrancisco", "旧金山":
		return "sanfrancisco"
	case "london", "伦敦":
		return "london"
	case "paris", "巴黎":
		return "paris"
	case "tokyo", "东京":
		return "tokyo"
	case "singapore", "新加坡":
		return "singapore"
	case "tianjin", "天津":
		return "tianjin"
	case "hebei", "河北":
		return "hebei"
	case "shanxi", "山西":
		return "shanxi"
	case "innermongolia", "内蒙古":
		return "innermongolia"
	case "liaoning", "辽宁":
		return "liaoning"
	case "jilin", "吉林":
		return "jilin"
	case "heilongjiang", "黑龙江":
		return "heilongjiang"
	case "jiangsu", "江苏":
		return "jiangsu"
	case "zhejiang", "浙江":
		return "zhejiang"
	case "anhui", "安徽":
		return "anhui"
	case "fujian", "福建":
		return "fujian"
	case "jiangxi", "江西":
		return "jiangxi"
	case "shandong", "山东":
		return "shandong"
	case "henan", "河南":
		return "henan"
	case "hubei", "湖北":
		return "hubei"
	case "hunan", "湖南":
		return "hunan"
	case "guangdong", "广东":
		return "guangdong"
	case "guangxi", "广西":
		return "guangxi"
	case "hainan", "海南":
		return "hainan"
	case "chongqing", "重庆":
		return "chongqing"
	case "sichuan", "四川":
		return "sichuan"
	case "guizhou", "贵州":
		return "guizhou"
	case "yunnan", "云南":
		return "yunnan"
	case "xizang", "tibet", "西藏":
		return "xizang"
	case "shaanxi", "陕西":
		return "shaanxi"
	case "gansu", "甘肃":
		return "gansu"
	case "qinghai", "青海":
		return "qinghai"
	case "ningxia", "宁夏":
		return "ningxia"
	case "xinjiang", "新疆":
		return "xinjiang"
	case "hongkong", "香港":
		return "hongkong"
	case "macau", "macao", "澳门":
		return "macau"
	case "taiwan", "臺灣", "台湾":
		return "taiwan"
	default:
		return token
	}
}

func normalizeToken(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return raw
	}

	var b strings.Builder
	for _, r := range raw {
		if unicode.IsSpace(r) {
			continue
		}
		switch r {
		case '-', '_', '\'', '"', '.', '·', '•', '/', '\\', '（', '）', '(', ')':
			continue
		default:
			b.WriteRune(r)
		}
	}

	token := b.String()

	// Chinese suffixes
	replacements := []string{"特别行政区", "特別行政區", "行政区", "行政區", "壮族自治区", "回族自治区", "维吾尔自治区", "自治區", "自治区", "省", "市", "區", "区"}
	for _, s := range replacements {
		token = strings.ReplaceAll(token, s, "")
	}

	// English suffixes
	englishDrops := []string{"specialadministrativeregion", "autonomousregion", "municipality", "province", "region", "city", "sar"}
	for _, s := range englishDrops {
		token = strings.ReplaceAll(token, s, "")
	}

	return token
}
