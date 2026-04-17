package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustReadPage(t *testing.T, name string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join("..", "..", "web", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}

	return string(content)
}

func requireMarkers(t *testing.T, html string, markers ...string) {
	t.Helper()

	for _, marker := range markers {
		if !strings.Contains(html, marker) {
			t.Fatalf("expected marker %q", marker)
		}
	}
}

func TestIndexPageIncludesRedesignStructure(t *testing.T) {
	html := mustReadPage(t, "index.html")

	requireMarkers(t, html,
		`id="bg-video"`,
		`id="hero"`,
		`id="generator"`,
		`id="chart-section"`,
		`id="reading-section"`,
		`id="todo-section"`,
		`id="recent-reports"`,
		`id="refresh-reports"`,
		`id="download-pdf"`,
		`id="copy-report-link"`,
		`function syncReportURL(reportID)`,
		`function loadReportFromQuery()`,
	)
}

func TestIndexPagesUseSeelioBrandAndSeekingStatus(t *testing.T) {
	for _, name := range []string{"index.html", "index.preview-c.html"} {
		html := mustReadPage(t, name)

		requireMarkers(t, html,
			`<title>观我 Seelio - 西方本命盘</title>`,
			`>观我 Seelio<`,
			`<h1 class="hero-title">观我 Seelio</h1>`,
			`向内求索，与星辰对话。`,
			`输入出生信息，绘制你的专属星图与本命说明书。`,
			`<h2 class="section-title">输入出生信息，绘制星图</h2>`,
			`时间像一条河流，你的第一声啼哭是涟漪的起点。`,
			`<span class="pill-core">绘制专属星图</span>`,
			`data-idle-text="绘制专属星图"`,
			`data-loading-text="正在计算黄道夹角..."`,
			`const submitIdleStatus = 'Seeking...';`,
			`setSubmitStatus(submitIdleStatus);`,
			`正在计算黄道夹角...`,
			`追溯那一刻的星尘记忆...`,
			`绘制你的心灵等高线...`,
			`function startSubmitLoadingStatus()`,
			`function stopSubmitLoadingStatus()`,
		)

		blockedMarkers := []string{
			`Astrolabe - 西方本命盘`,
			`ASTROLABE`,
			`输入出生信息，生成星盘可视化与结构化解读。`,
			`开始生成`,
			`输入出生信息，生成星盘`,
			`填写你的出生日期、时间、省份与基础内容，获取你的本命盘说明书。`,
		}

		for _, marker := range blockedMarkers {
			if strings.Contains(html, marker) {
				t.Fatalf("expected marker to be removed %q in %s", marker, name)
			}
		}
	}
}

func TestIndexPagesDeferLoadingUntilBirthDateValidationPasses(t *testing.T) {
	for _, name := range []string{"index.html", "index.preview-c.html"} {
		html := mustReadPage(t, name)

		validationIdx := strings.Index(html, `if (!isValidBirthDate(payload.birth_date)) {`)
		loadingIdx := strings.Index(html, `startSubmitLoadingStatus();`)
		disableIdx := strings.Index(html, `submitButton.disabled = true;`)

		if validationIdx < 0 || loadingIdx < 0 || disableIdx < 0 {
			t.Fatalf("missing submit flow markers in %s", name)
		}
		if !(validationIdx < loadingIdx && validationIdx < disableIdx) {
			t.Fatalf("expected validation to run before loading state in %s", name)
		}
	}
}

func TestIndexPageUsesReadableChineseForReportActions(t *testing.T) {
	html := mustReadPage(t, "index.html")

	requireMarkers(t, html,
		`复制详情链接`,
		`最近报告`,
		`刷新`,
		`加载报告失败`,
		`详情链接已复制`,
	)
}

func TestIndexPageUsesUpdatedNavigationLabels(t *testing.T) {
	html := mustReadPage(t, "index.html")

	requireMarkers(t, html,
		`>我的信息<`,
		`>我的星盘<`,
		`>本命解读<`,
		`>更多内容<`,
	)
}

func TestChartSectionUsesCompactMyChartCopy(t *testing.T) {
	html := mustReadPage(t, "index.html")

	requireMarkers(t, html,
		`<h2 class="section-title">我的星盘</h2>`,
		`class="chart-copy chart-copy-compact"`,
		`<h3>我的星盘</h3>`,
		`宫位、黄道、ASC / MC 与相位会在这里集中显示。`,
	)
}

func TestReadingSectionUsesCompactNatalReadingCopy(t *testing.T) {
	html := mustReadPage(t, "index.html")

	requireMarkers(t, html,
		`<h2 class="section-title">本命解读</h2>`,
		`<h3>本命解读</h3>`,
		`summary-card-compact`,
		`四大主题、依据链和质量指标会在这里集中展开。`,
	)
}

func TestIndexPageUsesReadableUnifiedSelectStyles(t *testing.T) {
	html := mustReadPage(t, "index.html")

	requireMarkers(t, html,
		`select {`,
		`background-color: rgba(8, 8, 10, 0.96);`,
		`color: #f5f7fb;`,
		`border: 1px solid rgba(255, 255, 255, 0.16);`,
		`box-shadow: 0 18px 40px rgba(0, 0, 0, 0.32);`,
		`select option {`,
		`background: #101217;`,
		`color: #f5f7fb;`,
	)
}

func TestChartSectionIncludesPlanetPlacementList(t *testing.T) {
	html := mustReadPage(t, "index.html")

	requireMarkers(t, html,
		`id="planet-placement-list"`,
		`class="planet-placement-list"`,
		`class="planet-placement-empty"`,
		`const planetPlacementLabels = {`,
		`const signLabels = {`,
		`function renderPlanetPlacements(chart)`,
		`planet-placement-line`,
		`planet.house`,
	)
}

func TestIndexPageUsesMonochromeProductTokens(t *testing.T) {
	html := mustReadPage(t, "index.html")

	requireMarkers(t, html,
		`--surface: rgba(248, 248, 248, 0.04);`,
		`--surface-strong: rgba(255, 255, 255, 0.08);`,
		`--surface-soft: rgba(214, 214, 214, 0.03);`,
		`--shadow: 0 22px 64px rgba(0, 0, 0, 0.5);`,
		`--section-pad-x: clamp(20px, 6vw, 96px);`,
		`background: rgba(0, 0, 0, 0.82);`,
		`letter-spacing: 0.24em;`,
		`background: linear-gradient(135deg, #d7dbe2, #ffffff);`,
	)
}

func TestIndexPreviewCPreservesStructureWithBlackTitaniumTokens(t *testing.T) {
	html := mustReadPage(t, "index.preview-c.html")

	requireMarkers(t, html,
		`id="bg-video"`,
		`id="generator"`,
		`id="chart-section"`,
		`id="reading-section"`,
		`id="recent-reports"`,
		`function loadReportFromQuery()`,
		`--accent: #a68e64;`,
		`--accent-soft: rgba(166, 142, 100, 0.1);`,
		`--navy-soft: rgba(28, 35, 46, 0.36);`,
		`radial-gradient(circle at top, rgba(28, 35, 46, 0.36), transparent 34%)`,
		`background: linear-gradient(180deg, rgba(36, 42, 50, 0.96), rgba(14, 16, 20, 0.98));`,
	)
}

func TestIndexPageUsesDefaultConfigSummaryAndVipLocks(t *testing.T) {
	html := mustReadPage(t, "index.html")

	requireMarkers(t, html,
		`class="field field-hidden"`,
		`class="form-defaults"`,
		`默认宫位系统：Placidus`,
		`默认黄道类型：Tropical`,
		`class="actions-group actions-group-free"`,
		`class="actions-group actions-group-vip"`,
		`class="actions actions-free"`,
		`class="actions actions-vip"`,
		`class="actions-heading actions-heading-vip"`,
		`class="vip-badge"`,
		`VIP会员专享`,
		`id="download-json"`,
		`id="download-pdf"`,
		`id="copy-report"`,
		`id="copy-report-link"`,
		`disabled data-vip-locked="true"`,
		`下载 JSON`,
		`下载 PDF`,
		`复制文本报告`,
		`id="fill-sample"`,
		`action-hidden`,
	)
}

func TestIndexPreviewCUsesDefaultConfigSummaryAndVipLocks(t *testing.T) {
	html := mustReadPage(t, "index.preview-c.html")

	requireMarkers(t, html,
		`class="field field-hidden"`,
		`class="form-defaults"`,
		`默认宫位系统：Placidus`,
		`默认黄道类型：Tropical`,
		`class="actions-group actions-group-free"`,
		`class="actions-group actions-group-vip"`,
		`class="actions actions-free"`,
		`class="actions actions-vip"`,
		`class="actions-heading actions-heading-vip"`,
		`class="vip-badge"`,
		`VIP会员专享`,
		`id="download-json"`,
		`id="download-pdf"`,
		`id="copy-report"`,
		`id="copy-report-link"`,
		`disabled data-vip-locked="true"`,
		`下载 JSON`,
		`下载 PDF`,
		`复制文本报告`,
		`id="fill-sample"`,
		`action-hidden`,
	)
}

func TestIndexPageRefinesGeneratorReadingAndRoadmapCopy(t *testing.T) {
	html := mustReadPage(t, "index.html")

	requireMarkers(t, html,
		`输入出生信息，绘制你的专属星图与本命说明书。`,
		`向内求索，与星辰对话。`,
		`时间像一条河流，你的第一声啼哭是涟漪的起点。`,
		`四大主题、依据链和质量指标会在这里集中展开。`,
		`class="todo-header-meta"`,
		`class="vip-badge vip-badge-roadmap"`,
		`解锁持续更新的个人周运与 VIP 专享月运解读。`,
		`class="surface todo-card todo-card-vip"`,
	)

	blockedMarkers := []string{
		`<aside class="surface intro-surface">`,
		`<h3>快速开始</h3>`,
		`填写出生日期、时间、省份与基础参数，开始生成你的本命盘结果。`,
		`后续还会持续扩展更多占星相关功能。`,
	}

	for _, marker := range blockedMarkers {
		if strings.Contains(html, marker) {
			t.Fatalf("expected marker to be removed %q", marker)
		}
	}
}

func TestIndexPreviewCRefinesGeneratorReadingAndRoadmapCopy(t *testing.T) {
	html := mustReadPage(t, "index.preview-c.html")

	requireMarkers(t, html,
		`输入出生信息，绘制你的专属星图与本命说明书。`,
		`向内求索，与星辰对话。`,
		`时间像一条河流，你的第一声啼哭是涟漪的起点。`,
		`四大主题、依据链和质量指标会在这里集中展开。`,
		`class="todo-header-meta"`,
		`class="vip-badge vip-badge-roadmap"`,
		`解锁持续更新的个人周运与 VIP 专享月运解读。`,
		`class="surface todo-card todo-card-vip"`,
	)

	blockedMarkers := []string{
		`<aside class="surface intro-surface">`,
		`<h3>快速开始</h3>`,
		`填写出生日期、时间、省份与基础参数，开始生成你的本命盘结果。`,
		`后续还会持续扩展更多占星相关功能。`,
	}

	for _, marker := range blockedMarkers {
		if strings.Contains(html, marker) {
			t.Fatalf("expected preview marker to be removed %q", marker)
		}
	}
}

func TestIndexPageIncludesAuthEntryAndClientFlow(t *testing.T) {
	html := mustReadPage(t, "index.html")

	requireMarkers(t, html,
		`id="auth-entry-button"`,
		`id="auth-modal"`,
		`id="auth-phone"`,
		`id="auth-code"`,
		`id="request-code-button"`,
		`id="verify-code-button"`,
		`id="logout-button"`,
		`async function loadCurrentUser()`,
		`fetch('/api/v1/me')`,
		`fetch('/api/v1/auth/request-code'`,
		`fetch('/api/v1/auth/verify-code'`,
		`fetch('/api/v1/auth/logout'`,
	)
}

func TestMemberCenterStructureExistsInBothPages(t *testing.T) {
	for _, name := range []string{"index.html", "index.preview-c.html"} {
		html := mustReadPage(t, name)

		requireMarkers(t, html,
			`id="member-section"`,
			`会员中心`,
			`id="member-summary-card"`,
			`id="member-orders-list"`,
			`id="member-phone"`,
			`id="member-status"`,
			`id="member-plan"`,
			`id="member-expiry"`,
			`const memberSummaryCard = document.querySelector('#member-summary-card');`,
			`const memberOrdersList = document.querySelector('#member-orders-list');`,
			`async function loadBillingOrders() {`,
			`function renderMemberCenter() {`,
			`function syncVipActions() {`,
			`/api/v1/billing/orders`,
			`/api/v1/reports/${encodeURIComponent(latest.report_id)}/json`,
			`/api/v1/reports/${encodeURIComponent(latest.report_id)}/text`,
		)
	}
}

func TestMembershipCopyReflectsWeeklyAndMonthlyBoundary(t *testing.T) {
	for _, name := range []string{"index.html", "index.preview-c.html"} {
		html := mustReadPage(t, name)

		requireMarkers(t, html,
			`限时赠送 2 次个人周运解读，月运解读为 VIP 专享。`,
			`当前账号已开通 VIP，可使用完整解读、周运更新与月运专享权益。`,
			`当前账号为免费账户，可领取 2 次限时周运，月运解读需开通 VIP。`,
			`个人周运解读`,
			`个人月运解读`,
			`新用户可限时领取 2 次个人周运解读。`,
			`解锁持续更新的个人周运与 VIP 专享月运解读。`,
			`月运解读与深度专题将作为 VIP 专享能力开放。`,
		)
	}
}

func TestLegalPagesAndFooterLinksExist(t *testing.T) {
	for _, name := range []string{"index.html", "index.preview-c.html"} {
		html := mustReadPage(t, name)

		requireMarkers(t, html,
			`class="site-footer"`,
			`href="/terms.html"`,
			`href="/privacy.html"`,
			`href="/disclaimer.html"`,
			`href="/refund.html"`,
			`用户协议`,
			`隐私政策`,
			`免责声明`,
			`支付与退款说明`,
		)
	}

	type legalPageExpectation struct {
		name    string
		title   string
		intro   string
		markers []string
	}

	expectations := []legalPageExpectation{
		{
			name:  "terms.html",
			title: `用户协议 | 观我 Seelio`,
			intro: `本协议用于说明 观我 Seelio 的使用规则、会员服务范围以及用户的基本权利义务。`,
			markers: []string{
				`本命盘生成`,
				`会员导出权益`,
				`会员方案和服务策略进行调整`,
			},
		},
		{
			name:  "privacy.html",
			title: `隐私政策 | 观我 Seelio`,
			intro: `本政策用于说明 观我 Seelio 收集、使用和保护用户信息的基本方式。`,
		},
		{
			name:  "disclaimer.html",
			title: `免责声明 | 观我 Seelio`,
			intro: `本页面用于说明 观我 Seelio 占星内容的适用边界。`,
			markers: []string{
				`本命盘解读`,
				`娱乐性阅读`,
				`责任限制`,
			},
		},
		{
			name:  "refund.html",
			title: `支付与退款说明 | 观我 Seelio`,
			intro: `本页面用于说明 观我 Seelio 第一版会员支付和退款的基本规则。`,
		},
	}

	for _, expectation := range expectations {
		html := mustReadPage(t, expectation.name)
		requireMarkers(t, html,
			expectation.title,
			`>观我 Seelio<`,
			`返回首页`,
			`最后更新`,
			expectation.intro,
		)
		requireMarkers(t, html, expectation.markers...)

		for _, blocked := range []string{`Astrolabe`, `ASTROLABE`} {
			if strings.Contains(html, blocked) {
				t.Fatalf("expected %s to remove %q", expectation.name, blocked)
			}
		}
	}
}

func TestVIPPageAndEntryLinksExist(t *testing.T) {
	for _, name := range []string{"index.html", "index.preview-c.html"} {
		html := mustReadPage(t, name)
		requireMarkers(t, html,
			`href="/vip.html"`,
			`立即开通 VIP`,
		)
	}

	html := mustReadPage(t, "vip.html")
	requireMarkers(t, html,
		`<title>VIP会员 | 观我 Seelio</title>`,
		`>观我 Seelio<`,
		`VIP会员`,
		`<span class="hero-brand-line">向内求索，与星辰对话。</span>`,
		`<span class="hero-copy-secondary">`,
		"return `${phone.slice(0, 3)}****`;",
		`setAuthStatus('验证码已发送到开发日志：');`,
		`VIP 专享的个人月运解读`,
		`月卡`,
		`季卡`,
		`年卡`,
		`12.9`,
		`29.9`,
		`99`,
		`VIP会员权益`,
		`下载 JSON / PDF 报告并复制完整文本`,
		`完整版本命解读`,
		`个人周运持续更新`,
		`个人月运解读 VIP 专享`,
		`限时赠送 2 次个人周运解读`,
		`id="vip-auth-entry-button"`,
		`id="vip-auth-modal"`,
		`id="vip-order-feedback"`,
		`data-plan-code="vip_monthly"`,
		`data-plan-code="vip_quarterly"`,
		`data-plan-code="vip_yearly"`,
		`fetch('/api/v1/me')`,
		`fetch('/api/v1/auth/request-code'`,
		`fetch('/api/v1/auth/verify-code'`,
		`fetch('/api/v1/billing/orders'`,
		"fetch(`/api/v1/billing/orders/${order.order.id}/mock-pay`",
	)
}

func TestVIPPageReflectsWeeklyAndMonthlyBenefits(t *testing.T) {
	html := mustReadPage(t, "vip.html")

	requireMarkers(t, html,
		`<title>VIP会员 | 观我 Seelio</title>`,
		`>观我 Seelio<`,
		`<p class="hero-text">`,
		`<span class="hero-brand-line">向内求索，与星辰对话。</span>`,
		`<span class="hero-copy-secondary">`,
		"return `${phone.slice(0, 3)}****`;",
		`setAuthStatus('验证码已发送到开发日志：');`,
		`解锁完整本命解读`,
		`VIP 专享的个人月运解读`,
		`</p>`,
		`限时赠送 2 次个人周运解读`,
		`完整版本命解读`,
		`个人周运持续更新`,
		`个人月运解读 VIP 专享`,
		`完整历史记录与会员中心管理`,
		`class="benefit-intro"`,
		`class="benefit-highlight"`,
	)
}
