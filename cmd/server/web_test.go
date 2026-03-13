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
