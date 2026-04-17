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
		`填写你的出生日期、时间、省份与基础内容，获取你的本命盘说明书。`,
		`你的个人本命盘说明书，解读你的人生密码。`,
		`四大主题、依据链和质量指标会在这里集中展开。`,
		`class="todo-header-meta"`,
		`class="vip-badge vip-badge-roadmap"`,
		`更多功能均为会员能力，后续将逐步开放。`,
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
		`填写你的出生日期、时间、省份与基础内容，获取你的本命盘说明书。`,
		`你的个人本命盘说明书，解读你的人生密码。`,
		`四大主题、依据链和质量指标会在这里集中展开。`,
		`class="todo-header-meta"`,
		`class="vip-badge vip-badge-roadmap"`,
		`更多功能均为会员能力，后续将逐步开放。`,
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

	for _, page := range []string{"terms.html", "privacy.html", "disclaimer.html", "refund.html"} {
		html := mustReadPage(t, page)
		requireMarkers(t, html,
			`ASTROLABE`,
			`返回首页`,
			`最后更新`,
		)
	}
}
