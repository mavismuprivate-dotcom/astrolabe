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

type themeSignal struct {
	Theme   string
	Score   float64
	Factors []string
}

func buildReading(planets map[string]PlanetPosition, aspects []Aspect, approximate bool) Reading {
	sun := planets["Sun"]
	moon := planets["Moon"]
	venus := planets["Venus"]
	mars := planets["Mars"]

	signals := extractThemeSignals(planets, aspects)

	personality := fmt.Sprintf("太阳%s、月亮%s。你的外在决策偏向%s，内在情绪需求偏向%s。太阳位于第%d宫，说明你会持续把精力投入在%s。你对自我成长的驱动并不短促，而是倾向通过目标拆解和行动复盘来稳步提升。", signCN(sun.Sign), signCN(moon.Sign), signModeCN(sun.Sign), moonNeedCN(moon.Sign), maxInt(1, sun.House), houseThemeCN(maxInt(1, sun.House)))
	relationship := fmt.Sprintf("金星在%s、月亮在%s，关系风格强调%s与%s。你在亲密关系里既重视情绪反馈，也希望保留个人节奏，适合通过“提前约定边界+稳定沟通仪式”来降低误解成本。", signCN(venus.Sign), signCN(moon.Sign), relationKeyword(venus.Sign), moonNeedCN(moon.Sign))

	love := buildThemeNarrative("love", signals["love"], "你在爱情关系中会把“情感安全”和“长期成长”并重考虑。", "风险点在于当情绪波动出现时，容易出现需求表达和实际行动不同步。", "建议固定每周一次关系复盘，用事实和感受分层表达，先确认共识再推进承诺。")
	career := buildThemeNarrative("career", signals["career"], fmt.Sprintf("职业发展上，你更适合通过%s建立稳定竞争力。", houseThemeCN(maxInt(1, mars.House))), "风险点是阶段性目标过多时容易拉高内耗，执行路径可能被频繁切换。", "建议采用“季度主目标 + 周度交付件”的节奏，并保留复盘窗口来做策略校准。")
	money := buildThemeNarrative("money", signals["money"], moneyOutlook(planets), "风险点在于收入扩张速度与风险承受能力不匹配时，可能放大现金流波动。", "建议持续执行分层资金管理：基础储备、稳健配置、进取预算三层分离。")
	family := buildThemeNarrative("family", signals["family"], "家庭与内在安全议题是你的长期底层驱动力。你会在情感连接、责任承担和边界维护之间寻找平衡。", "风险点是过度承担他人情绪劳动，导致个人恢复周期被压缩。", "建议建立“可持续支持”原则：明确可提供帮助范围，避免长期透支。")

	growth := growthFromAspects(aspects)
	action := fmt.Sprintf("行动建议：1) 在%s设一个可量化目标；2) 每天固定20-30分钟复盘触发情绪；3) 每周一次主题复盘（爱情/事业/金钱/家庭）；4) 每月做一次长期目标与现实资源对表。", houseThemeCN(maxInt(1, sun.House)))
	focus := aspectHighlights(aspects)
	summary := buildSummary(love, career, money, family)
	reminder := "解读依据来自“行星×星座×宫位×相位”规则系统。每条主题结论均附结构化证据，便于复核。"
	if approximate {
		reminder += " 当前为近似出生时刻模式，宫位细节和月亮相关结论请保守解读。"
	}

	evidence := generateThemeEvidence(signals, love, career, money, family)
	disclaimer := "本结果基于西方占星规则模板生成，仅供娱乐与自我观察参考。"

	textReport := strings.Join([]string{
		"【人格底色】" + personality,
		"【关系模式】" + relationship,
		"【爱情解析】" + love,
		"【事业路径】" + career,
		"【金钱主题】" + money,
		"【家庭主题】" + family,
		"【成长课题】" + growth,
		"【行动建议】" + action,
		"【关键相位】" + focus,
		"【综合摘要】" + summary,
		"【提示】" + reminder,
		disclaimer,
	}, "\n\n")

	reading := Reading{
		Personality:   personality,
		Relationship:  relationship,
		Love:          love,
		Career:        career,
		Money:         money,
		Family:        family,
		Summary:       summary,
		Growth:        growth,
		Action:        action,
		Focus:         focus,
		Evidence:      evidence,
		Reminder:      reminder,
		Disclaimer:    disclaimer,
		TextReport:    textReport,
		Entertainment: "仅供娱乐与参考",
	}
	reading.Quality = assessReadingQuality(reading)

	if ok, reasons := passesQualityGate(reading); !ok {
		reading.Summary = "内容质量门禁未完全通过，已降级为保守版结论。请补全出生时刻并稍后重试。"
		reading.Reminder += " 质量门禁提示：" + strings.Join(reasons, "；")
	}

	return reading
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

func extractThemeSignals(planets map[string]PlanetPosition, aspects []Aspect) map[string]themeSignal {
	signals := map[string]themeSignal{
		"love":   {Theme: "love", Score: 1.0, Factors: []string{}},
		"career": {Theme: "career", Score: 1.0, Factors: []string{}},
		"money":  {Theme: "money", Score: 1.0, Factors: []string{}},
		"family": {Theme: "family", Score: 1.0, Factors: []string{}},
	}

	venus := planets["Venus"]
	mars := planets["Mars"]
	moon := planets["Moon"]
	sun := planets["Sun"]
	jupiter := planets["Jupiter"]
	saturn := planets["Saturn"]

	addSignalFactor(signals, "love", 1.2, fmt.Sprintf("金星%s第%d宫", signCN(venus.Sign), maxInt(1, venus.House)))
	addSignalFactor(signals, "love", 1.0, fmt.Sprintf("月亮%s第%d宫", signCN(moon.Sign), maxInt(1, moon.House)))
	addSignalFactor(signals, "love", 0.8, fmt.Sprintf("火星%s第%d宫", signCN(mars.Sign), maxInt(1, mars.House)))

	addSignalFactor(signals, "career", 1.3, fmt.Sprintf("太阳第%d宫", maxInt(1, sun.House)))
	addSignalFactor(signals, "career", 1.2, fmt.Sprintf("火星第%d宫", maxInt(1, mars.House)))
	addSignalFactor(signals, "career", 1.0, fmt.Sprintf("土星%s", signCN(saturn.Sign)))

	addSignalFactor(signals, "money", 1.3, fmt.Sprintf("木星%s第%d宫", signCN(jupiter.Sign), maxInt(1, jupiter.House)))
	addSignalFactor(signals, "money", 1.1, fmt.Sprintf("金星第%d宫", maxInt(1, venus.House)))
	addSignalFactor(signals, "money", 1.0, fmt.Sprintf("第2宫主题=%s", houseThemeCN(2)))

	addSignalFactor(signals, "family", 1.4, fmt.Sprintf("月亮第%d宫", maxInt(1, moon.House)))
	addSignalFactor(signals, "family", 1.1, fmt.Sprintf("月亮%s", signCN(moon.Sign)))
	addSignalFactor(signals, "family", 1.0, fmt.Sprintf("第4宫主题=%s", houseThemeCN(4)))

	for _, asp := range aspects {
		ab := asp.BodyA + "-" + asp.BodyB + " " + aspectCN(asp.Type)
		switch {
		case containsAny(asp.BodyA, asp.BodyB, "Venus", "Moon", "Mars"):
			addSignalFactor(signals, "love", 0.9, "相位:"+ab)
		case containsAny(asp.BodyA, asp.BodyB, "Sun", "Mars", "Saturn", "MC"):
			addSignalFactor(signals, "career", 0.9, "相位:"+ab)
		case containsAny(asp.BodyA, asp.BodyB, "Jupiter", "Venus", "Saturn"):
			addSignalFactor(signals, "money", 0.9, "相位:"+ab)
		}
		if containsAny(asp.BodyA, asp.BodyB, "Moon") {
			addSignalFactor(signals, "family", 0.8, "相位:"+ab)
		}
	}

	for k, s := range signals {
		s.Factors = dedupStrings(s.Factors)
		signals[k] = s
	}
	return signals
}

func addSignalFactor(signals map[string]themeSignal, theme string, score float64, factor string) {
	s := signals[theme]
	s.Score += score
	s.Factors = append(s.Factors, factor)
	signals[theme] = s
}

func containsAny(a, b string, names ...string) bool {
	for _, n := range names {
		if a == n || b == n {
			return true
		}
	}
	return false
}

func dedupStrings(in []string) []string {
	if len(in) == 0 {
		return in
	}
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func buildThemeNarrative(theme string, signal themeSignal, mainClaim string, risk string, action string) string {
	factors := signal.Factors
	if len(factors) > 4 {
		factors = factors[:4]
	}
	anchors := strings.Join(factors, "、")
	return fmt.Sprintf("%s 该主题的核心信号来自：%s。风险提示：%s 建议动作：%s 在执行层面，建议使用“目标-证据-复盘”闭环：先定义目标，再记录事实证据，最后以周为单位复盘并做微调。", mainClaim, anchors, risk, action)
}

func buildSummary(love, career, money, family string) string {
	return fmt.Sprintf("综合来看，你的本命结构呈现“关系与责任并重、目标与稳定并行”的特征。爱情维度强调长期协同，事业维度强调节奏管理，金钱维度强调结构化分配，家庭维度强调边界与支持的平衡。建议把四个主题放在同一成长路径中：以关系稳定支撑事业执行，以财务秩序降低情绪波动，再通过家庭边界维持长期续航。%s%s", shortSnippet(love), shortSnippet(career))
}

func shortSnippet(v string) string {
	if len([]rune(v)) <= 46 {
		return v
	}
	r := []rune(v)
	return string(r[:46]) + "…"
}

func generateThemeEvidence(signals map[string]themeSignal, love, career, money, family string) []EvidenceItem {
	themeClaims := map[string]string{
		"love":   love,
		"career": career,
		"money":  money,
		"family": family,
	}

	items := make([]EvidenceItem, 0, 12)
	for _, theme := range []string{"love", "career", "money", "family"} {
		sig := signals[theme]
		claim := themeClaims[theme]
		items = append(items, buildEvidenceFromSignal(theme, claim, sig, 0))
		items = append(items, buildEvidenceFromSignal(theme, claim, sig, 1))
	}
	return items
}

func buildEvidenceFromSignal(theme, claim string, sig themeSignal, offset int) EvidenceItem {
	factors := sig.Factors
	if len(factors) == 0 {
		factors = []string{"默认规则模板"}
	}
	start := offset * 2
	if start >= len(factors) {
		start = 0
	}
	end := start + 3
	if end > len(factors) {
		end = len(factors)
	}
	selected := factors[start:end]
	if len(selected) < 2 && len(factors) >= 2 {
		selected = factors[:2]
	}
	return EvidenceItem{
		Theme:      theme,
		Claim:      strings.TrimSpace(claim),
		Factors:    selected,
		Confidence: confidenceFromScore(sig.Score),
	}
}

func confidenceFromScore(score float64) float64 {
	c := 0.42 + score*0.045
	if c > 0.95 {
		c = 0.95
	}
	if c < 0.35 {
		c = 0.35
	}
	return round2(c)
}

func assessReadingQuality(reading Reading) QualityMetrics {
	charCount := len([]rune(reading.TextReport))
	duplicate := calcDuplicateRatio(reading.TextReport)
	themeCoverage := calcThemeCoverage(reading)
	consistency := calcConsistencyScore(reading.TextReport, duplicate)

	return QualityMetrics{
		CharCount:        charCount,
		DuplicateRatio:   round2(duplicate),
		ThemeCoverage:    round2(themeCoverage),
		ConsistencyScore: round2(consistency),
	}
}

func calcDuplicateRatio(text string) float64 {
	sentences := splitSentences(text)
	if len(sentences) <= 1 {
		return 0
	}
	unique := make(map[string]bool, len(sentences))
	for _, s := range sentences {
		n := normalizeSentence(s)
		if n == "" {
			continue
		}
		unique[n] = true
	}
	if len(unique) == 0 {
		return 1
	}
	return 1 - float64(len(unique))/float64(len(sentences))
}

func splitSentences(text string) []string {
	s := strings.NewReplacer("！", "。", "？", "。", "；", "。", "!", ".", "?", ".", ";", ".", "\n", "。").Replace(text)
	parts := strings.Split(s, "。")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func normalizeSentence(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.NewReplacer(" ", "", "，", "", ",", "", "。", "", ".", "", "：", "", ":", "", "“", "", "”", "").Replace(s)
	return s
}

func calcThemeCoverage(reading Reading) float64 {
	total := 4.0
	covered := 0.0
	if strings.TrimSpace(reading.Love) != "" {
		covered++
	}
	if strings.TrimSpace(reading.Career) != "" {
		covered++
	}
	if strings.TrimSpace(reading.Money) != "" {
		covered++
	}
	if strings.TrimSpace(reading.Family) != "" {
		covered++
	}
	return covered / total
}

func calcConsistencyScore(text string, duplicateRatio float64) float64 {
	pairs := [][2]string{
		{"高风险", "低风险"},
		{"激进", "保守"},
		{"稳定", "失控"},
		{"低波动", "高波动"},
	}
	hits := 0
	for _, p := range pairs {
		if strings.Contains(text, p[0]) && strings.Contains(text, p[1]) {
			hits++
		}
	}
	score := 1.0 - float64(hits)*0.12 - duplicateRatio*0.35
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	return score
}

func passesQualityGate(reading Reading) (bool, []string) {
	reasons := make([]string, 0)
	if len([]rune(reading.Love)) < 180 {
		reasons = append(reasons, "爱情主题长度不足")
	}
	if len([]rune(reading.Career)) < 180 {
		reasons = append(reasons, "事业主题长度不足")
	}
	if len([]rune(reading.Money)) < 180 {
		reasons = append(reasons, "金钱主题长度不足")
	}
	if len([]rune(reading.Family)) < 180 {
		reasons = append(reasons, "家庭主题长度不足")
	}
	eviCount := map[string]int{}
	for _, e := range reading.Evidence {
		eviCount[e.Theme]++
		if len(e.Factors) < 2 {
			reasons = append(reasons, "存在依据因素少于2条")
			break
		}
	}
	for _, theme := range []string{"love", "career", "money", "family"} {
		if eviCount[theme] < 2 {
			reasons = append(reasons, fmt.Sprintf("%s主题依据不足2条", theme))
		}
	}
	if reading.Quality.CharCount < 1200 {
		reasons = append(reasons, "总长度不足")
	}
	if reading.Quality.DuplicateRatio > 0.28 {
		reasons = append(reasons, "重复率过高")
	}
	if reading.Quality.ThemeCoverage < 1.0 {
		reasons = append(reasons, "主题覆盖不足")
	}
	if reading.Quality.ConsistencyScore < 0.55 {
		reasons = append(reasons, "一致性分数偏低")
	}
	return len(reasons) == 0, dedupStrings(reasons)
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
