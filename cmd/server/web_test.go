package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIndexPageIncludesRedesignStructure(t *testing.T) {
	indexPath := filepath.Join("..", "..", "web", "index.html")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}

	html := string(content)
	requiredMarkers := []string{
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
	}

	for _, marker := range requiredMarkers {
		if !strings.Contains(html, marker) {
			t.Fatalf("expected redesigned page marker %q", marker)
		}
	}
}

func TestIndexPageOmitsImplementationCommentary(t *testing.T) {
	indexPath := filepath.Join("..", "..", "web", "index.html")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}

	html := string(content)
	blockedPhrases := []string{
		"当前表单字段、接口协议和交互行为保持不变",
		"当前页面保留的核心内容",
		"同一个挂载点渲染",
		"沿用当前渲染逻辑",
		"当前展示内容",
		"黑白高对比风格",
		"自动滚动到这里",
		"TODO 占位",
	}

	for _, phrase := range blockedPhrases {
		if strings.Contains(html, phrase) {
			t.Fatalf("expected implementation commentary phrase to be removed: %q", phrase)
		}
	}
}

func TestIndexPageUsesReadableChineseForReportActions(t *testing.T) {
	indexPath := filepath.Join("..", "..", "web", "index.html")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}

	html := string(content)
	requiredPhrases := []string{
		"复制详情链接",
		"最近报告",
		"刷新",
		"点击某条历史记录，可直接回填结果到当前页面。当前是本地实例共享历史。",
		"暂无历史报告。先生成一次星盘后，这里会显示最近结果。",
		"请先生成报告或从历史记录中加载一份报告",
		"详情链接已复制",
		"加载报告失败",
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(html, phrase) {
			t.Fatalf("expected readable Chinese phrase %q", phrase)
		}
	}
}

func TestIndexPageUsesUpdatedNavigationLabels(t *testing.T) {
	indexPath := filepath.Join("..", "..", "web", "index.html")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}

	html := string(content)
	requiredPhrases := []string{
		">我的信息<",
		">我的星盘<",
		">本命解读<",
		">更多内容<",
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(html, phrase) {
			t.Fatalf("expected updated navigation phrase %q", phrase)
		}
	}
}

func TestChartSectionUsesCompactMyChartCopy(t *testing.T) {
	indexPath := filepath.Join("..", "..", "web", "index.html")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}

	html := string(content)
	requiredPhrases := []string{
		`<h2 class="section-title">我的星盘</h2>`,
		`class="chart-copy chart-copy-compact"`,
		`<h3>我的星盘</h3>`,
		`宫位、黄道、ASC / MC 与相位会在这里集中显示。`,
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(html, phrase) {
			t.Fatalf("expected compact chart section phrase %q", phrase)
		}
	}
}

func TestReadingSectionUsesCompactNatalReadingCopy(t *testing.T) {
	indexPath := filepath.Join("..", "..", "web", "index.html")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}

	html := string(content)
	requiredPhrases := []string{
		`<h2 class="section-title">本命解读</h2>`,
		`<h3>本命解读</h3>`,
		`summary-card-compact`,
		`四大主题、依据链和质量指标会在这里集中展开。`,
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(html, phrase) {
			t.Fatalf("expected compact reading section phrase %q", phrase)
		}
	}
}

func TestIndexPageUsesReadableUnifiedSelectStyles(t *testing.T) {
	indexPath := filepath.Join("..", "..", "web", "index.html")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}

	html := string(content)
	requiredMarkers := []string{
		`select {`,
		`background-color: rgba(12, 12, 16, 0.96);`,
		`color: #f5f7fb;`,
		`border: 1px solid rgba(255, 255, 255, 0.16);`,
		`box-shadow: 0 18px 40px rgba(0, 0, 0, 0.32);`,
		`select option {`,
		`background: #101217;`,
		`color: #f5f7fb;`,
	}

	for _, marker := range requiredMarkers {
		if !strings.Contains(html, marker) {
			t.Fatalf("expected unified select style marker %q", marker)
		}
	}
}
