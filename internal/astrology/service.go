package astrology

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type Service struct {
	resolver *CityResolver
}

func NewService(resolver *CityResolver) *Service {
	if resolver == nil {
		resolver = NewCityResolver()
	}
	return &Service{resolver: resolver}
}

func (s *Service) GenerateNatalChart(_ context.Context, req NatalChartRequest) (NatalChartResponse, error) {
	applyDefaults(&req)
	if err := validateRequest(req); err != nil {
		return NatalChartResponse{}, err
	}

	locationInput := strings.TrimSpace(req.BirthCity)
	if strings.TrimSpace(req.BirthProvince) != "" {
		locationInput = strings.TrimSpace(req.BirthProvince)
	}

	loc, ok := s.resolver.Resolve(locationInput, req.BirthCountry)
	if !ok {
		return NatalChartResponse{}, fmt.Errorf("unsupported birth_province/birth_city for local resolver (例如: 江苏省/中国, 南京/中国, New York/United States)")
	}
	if strings.TrimSpace(req.Timezone) != "" {
		loc.Timezone = strings.TrimSpace(req.Timezone)
	} else if loc.Timezone == "" {
		loc.Timezone = defaultTimezoneForCountry(req.BirthCountry)
	}

	instant, err := ResolveBirthDateTime(req, loc.Timezone)
	if err != nil {
		return NatalChartResponse{}, err
	}

	planetLon := CalculatePlanets(instant.JD)
	asc, mc, houseCusps := CalculateAnglesAndHouses(instant.JD, loc.Latitude, loc.Longitude)
	planetLon["ASC"] = asc
	planetLon["MC"] = mc

	planets := make(map[string]PlanetPosition, len(planetLon))
	for body, lon := range planetLon {
		if body == "ASC" || body == "MC" {
			continue
		}
		planets[body] = PlanetPosition{
			Longitude: lon,
			Sign:      ZodiacSign(lon),
			House:     HouseOfLongitude(lon, houseCusps),
		}
	}

	houses := make([]House, 0, 12)
	for i, cusp := range houseCusps {
		houses = append(houses, House{
			Number: i + 1,
			Cusp:   cusp,
			Sign:   ZodiacSign(cusp),
		})
	}

	aspects := DetectMajorAspects(planetLon)
	confidence := 0.86
	warnings := []string{}
	confidenceNote := "出生时间精确，宫位与月亮解读可信度较高。"
	if instant.Approximate {
		confidence = 0.62
		warnings = append(warnings, "未提供出生时刻，系统已按当地 12:00 近似计算。")
		confidenceNote = "当前为近似模式，宫位和月亮细节仅供参考。"
	}

	reading := buildReading(planets, aspects, instant.Approximate)
	response := NatalChartResponse{
		Meta: MetaInfo{
			Input: NormalizedInput{
				BirthDate:     req.BirthDate,
				BirthTime:     fmt.Sprintf("%02d:%02d", instant.LocalHour, instant.LocalMinute),
				BirthProvince: strings.TrimSpace(req.BirthProvince),
				BirthCity:     loc.City,
				BirthCountry:  req.BirthCountry,
				Timezone:      loc.Timezone,
				Latitude:      loc.Latitude,
				Longitude:     loc.Longitude,
				Calendar:      req.Calendar,
				HouseSystem:   req.HouseSystem,
				ZodiacType:    req.ZodiacType,
				Language:      req.Language,
			},
			Approximate:  instant.Approximate,
			Confidence:   confidence,
			GeneratedAt:  time.Now().UTC(),
			Warnings:     warnings,
			ConfidenceCN: confidenceNote,
		},
		Chart: Chart{
			JD:      instant.JD,
			Planets: planets,
			Houses:  houses,
			Aspects: aspects,
			Angles: map[string]AnglePoint{
				"ASC": {Longitude: asc, Sign: ZodiacSign(asc)},
				"MC":  {Longitude: mc, Sign: ZodiacSign(mc)},
			},
		},
		Reading: reading,
	}

	return response, nil
}

func validateRequest(req NatalChartRequest) error {
	if strings.TrimSpace(req.BirthDate) == "" {
		return errors.New("birth_date is required")
	}
	if strings.TrimSpace(req.BirthCity) == "" && strings.TrimSpace(req.BirthProvince) == "" {
		return errors.New("birth_province or birth_city is required")
	}
	return nil
}

func applyDefaults(req *NatalChartRequest) {
	if strings.TrimSpace(req.Calendar) == "" {
		req.Calendar = "gregorian"
	}
	if strings.TrimSpace(req.HouseSystem) == "" {
		req.HouseSystem = "placidus"
	}
	if strings.TrimSpace(req.ZodiacType) == "" {
		req.ZodiacType = "tropical"
	}
	if strings.TrimSpace(req.Language) == "" {
		req.Language = "zh-CN"
	}
	if strings.TrimSpace(req.BirthCountry) == "" {
		req.BirthCountry = "中国"
	}
}

func buildReading(planets map[string]PlanetPosition, aspects []Aspect, approximate bool) Reading {
	sun := planets["Sun"]
	moon := planets["Moon"]
	venus := planets["Venus"]
	mars := planets["Mars"]
	saturn := planets["Saturn"]

	personality := fmt.Sprintf("太阳%s、月亮%s。你的外在决策偏向%s，内在情绪需求偏向%s。太阳宫位第%d宫显示你会把主要精力投入到%s。", signCN(sun.Sign), signCN(moon.Sign), signModeCN(sun.Sign), moonNeedCN(moon.Sign), maxInt(1, sun.House), houseThemeCN(maxInt(1, sun.House)))
	relationship := fmt.Sprintf("金星在%s，关系表达更重视%s；月亮%s提示你在亲密关系里需要%s。建议把“需求表达”放在“情绪反应”之前。", signCN(venus.Sign), relationKeyword(venus.Sign), signCN(moon.Sign), moonNeedCN(moon.Sign))
	love := loveOutlook(planets, aspects)
	career := fmt.Sprintf("火星第%d宫强调行动力应投向%s；土星%s意味着你在%s议题上通过长期主义获胜。职业节奏上，先建立稳定流程，再扩大影响力。", maxInt(1, mars.House), houseThemeCN(maxInt(1, mars.House)), signCN(saturn.Sign), signModeCN(saturn.Sign))
	money := moneyOutlook(planets)
	growth := growthFromAspects(aspects)
	action := fmt.Sprintf("本周行动建议：1) 在%s设一个可量化目标；2) 每天固定 20-30 分钟复盘情绪触发点；3) 每周做一次关键关系沟通清单。", houseThemeCN(maxInt(1, sun.House)))
	focus := aspectHighlights(aspects)
	reminder := "解读逻辑参考通行占星框架：太阳/Identity、月亮/Emotions、上升/外在风格、宫位/生活领域、相位/能量互动。"
	if approximate {
		career = "当前为近似模式，宫位相关职业结论已降级为方向性建议，建议补充出生时刻后复算。"
		reminder += " 你当前使用近似出生时刻，宫位与月亮细节请谨慎解读。"
	}

	disclaimer := "本结果基于西方占星规则模板生成，仅供娱乐与自我观察参考。"
	textReport := strings.Join([]string{
		"【人格底色】" + personality,
		"【关系模式】" + relationship,
		"【爱情解析】" + love,
		"【事业路径】" + career,
		"【金钱主题】" + money,
		"【成长课题】" + growth,
		"【行动建议】" + action,
		"【关键相位】" + focus,
		"【提示】" + reminder,
		disclaimer,
	}, "\n\n")

	return Reading{
		Personality:   personality,
		Relationship:  relationship,
		Love:          love,
		Career:        career,
		Money:         money,
		Growth:        growth,
		Action:        action,
		Focus:         focus,
		Reminder:      reminder,
		Disclaimer:    disclaimer,
		TextReport:    textReport,
		Entertainment: "仅供娱乐与参考",
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func signCN(sign string) string {
	m := map[string]string{
		"Aries": "白羊座", "Taurus": "金牛座", "Gemini": "双子座", "Cancer": "巨蟹座",
		"Leo": "狮子座", "Virgo": "处女座", "Libra": "天秤座", "Scorpio": "天蝎座",
		"Sagittarius": "射手座", "Capricorn": "摩羯座", "Aquarius": "水瓶座", "Pisces": "双鱼座",
	}
	if v, ok := m[sign]; ok {
		return v
	}
	return sign
}

func relationKeyword(sign string) string {
	switch sign {
	case "Aries", "Leo", "Sagittarius":
		return "热烈直接的互动"
	case "Taurus", "Virgo", "Capricorn":
		return "稳定、长期投入"
	case "Gemini", "Libra", "Aquarius":
		return "沟通、精神共鸣"
	default:
		return "情绪连接与安全感"
	}
}

func aspectCN(kind string) string {
	switch kind {
	case "conjunction":
		return "合相"
	case "sextile":
		return "六合"
	case "square":
		return "刑相"
	case "trine":
		return "拱相"
	case "opposition":
		return "冲相"
	default:
		return kind
	}
}

func signModeCN(sign string) string {
	switch sign {
	case "Aries", "Leo", "Sagittarius":
		return "主动推进和目标点燃"
	case "Taurus", "Virgo", "Capricorn":
		return "结构化执行和稳定积累"
	case "Gemini", "Libra", "Aquarius":
		return "沟通协调和思维联动"
	default:
		return "情绪感知和关系连结"
	}
}

func moonNeedCN(sign string) string {
	switch sign {
	case "Aries", "Leo", "Sagittarius":
		return "空间感、自由度与即时反馈"
	case "Taurus", "Virgo", "Capricorn":
		return "秩序感、确定性与可控节奏"
	case "Gemini", "Libra", "Aquarius":
		return "对话感、理解感与观点交换"
	default:
		return "被理解、被接住与情绪安全"
	}
}

func houseThemeCN(house int) string {
	themes := map[int]string{
		1:  "自我呈现与个人定位",
		2:  "资源管理与价值建立",
		3:  "学习表达与信息处理",
		4:  "家庭根基与内在安全",
		5:  "创作表达与恋爱体验",
		6:  "日常系统与工作效率",
		7:  "关系合作与边界协商",
		8:  "深层关系与风险共享",
		9:  "进修扩展与信念升级",
		10: "事业目标与社会角色",
		11: "社群网络与长期愿景",
		12: "休整疗愈与潜意识整合",
	}
	if t, ok := themes[house]; ok {
		return t
	}
	return "生活平衡"
}

func growthFromAspects(aspects []Aspect) string {
	for _, asp := range aspects {
		if asp.Type == "square" || asp.Type == "opposition" {
			return fmt.Sprintf("%s 与 %s 的%s（容许度 %.1f°）提示你在“想法”和“行动”间容易拉扯。成长关键是把冲突拆成可执行步骤，而不是一次性求完美。", asp.BodyA, asp.BodyB, aspectCN(asp.Type), asp.Orb)
		}
	}
	if len(aspects) > 0 {
		asp := aspects[0]
		return fmt.Sprintf("%s 与 %s 的%s（容许度 %.1f°）是你的天然优势。成长方向是把已有天赋转化为长期可复用的方法论。", asp.BodyA, asp.BodyB, aspectCN(asp.Type), asp.Orb)
	}
	return "当前相位信息较少，建议重点观察“重复出现的情绪模式”和“长期坚持的行动模式”。"
}

func aspectHighlights(aspects []Aspect) string {
	if len(aspects) == 0 {
		return "暂无主要相位高亮。"
	}

	limit := 3
	if len(aspects) < limit {
		limit = len(aspects)
	}
	lines := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		asp := aspects[i]
		lines = append(lines, fmt.Sprintf("%s-%s：%s（容许度 %.1f°）", asp.BodyA, asp.BodyB, aspectCN(asp.Type), asp.Orb))
	}
	return strings.Join(lines, "；")
}

func moneyOutlook(planets map[string]PlanetPosition) string {
	secondHouseBodies := make([]string, 0)
	for body, p := range planets {
		if p.House == 2 {
			secondHouseBodies = append(secondHouseBodies, body)
		}
	}

	venus := planets["Venus"]
	jupiter := planets["Jupiter"]
	base := fmt.Sprintf("金星在%s、木星在%s，财务风格偏向“%s + %s”。", signCN(venus.Sign), signCN(jupiter.Sign), relationKeyword(venus.Sign), signModeCN(jupiter.Sign))

	if len(secondHouseBodies) > 0 {
		return base + fmt.Sprintf("第2宫有%s，代表你的收入增长更依赖主动经营个人价值。建议优先建立稳定现金流与长期储备。", strings.Join(secondHouseBodies, "、"))
	}

	return base + "第2宫无主要行星时，更需要用预算纪律与技能复利来放大收益。建议采用“固定储蓄比例 + 分层风险配置”的方式管理资金。"
}

func loveOutlook(planets map[string]PlanetPosition, aspects []Aspect) string {
	venus := planets["Venus"]
	mars := planets["Mars"]
	moon := planets["Moon"]

	seventhHouseBodies := make([]string, 0)
	for body, p := range planets {
		if p.House == 7 {
			seventhHouseBodies = append(seventhHouseBodies, body)
		}
	}
	sort.Strings(seventhHouseBodies)

	base := fmt.Sprintf("金星%s、火星%s，说明你在爱情里既追求%s，也需要%s。月亮%s提示你需要被持续回应与稳定陪伴。", signCN(venus.Sign), signCN(mars.Sign), relationKeyword(venus.Sign), signModeCN(mars.Sign), signCN(moon.Sign))

	if len(seventhHouseBodies) > 0 {
		base += fmt.Sprintf("第7宫有%s，你会把亲密关系当作人生主课题，适合在“明确边界+稳定投入”的关系里成长。", strings.Join(seventhHouseBodies, "、"))
	} else {
		base += "第7宫无主要行星时，关系质量更依赖你主动经营沟通和节奏，而不是等待“命定时刻”。"
	}

	for _, asp := range aspects {
		if (asp.BodyA == "Venus" || asp.BodyB == "Venus" || asp.BodyA == "Moon" || asp.BodyB == "Moon") &&
			(asp.Type == "square" || asp.Type == "opposition") {
			base += fmt.Sprintf("%s-%s 存在%s（容许度 %.1f°），恋爱中要特别注意情绪上头时的表达方式。", asp.BodyA, asp.BodyB, aspectCN(asp.Type), asp.Orb)
			return base
		}
	}

	base += "当前主要爱情相位偏顺畅，适合通过共同目标和日常协作来稳步升温。"
	return base
}
