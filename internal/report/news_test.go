package report

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/HITSZ-OpenAuto/hoa-news/internal/github"
	"github.com/HITSZ-OpenAuto/hoa-news/internal/utils"
)

func TestGenerateDailyFrontMatter(t *testing.T) {
	result, err := utils.GenerateFrontMatter(
		"AUTO 更新速递",
		time.Now().UTC().Format("2006-01-02"),
		"每日更新",
		[]utils.Author{{
			Name:  "github-actions[bot]",
			Link:  "https://github.com/features/actions",
			Image: "https://avatars.githubusercontent.com/in/15368",
		}},
	)
	if err != nil {
		t.Fatalf("GenerateFrontMatter() returned error: %v", err)
	}

	// Check required fields
	requiredFields := []string{
		"title:",
		"AUTO 更新速递",
		"date:",
		"authors:",
		"github-actions[bot]",
		"description:",
		"每日更新",
		"excludeSearch:",
		"draft:",
	}

	for _, field := range requiredFields {
		if !strings.Contains(result, field) {
			t.Errorf("GenerateDailyFrontMatter() output missing %q", field)
		}
	}

	// Verify it contains today's date
	today := time.Now().UTC().Format("2006-01-02")
	if !strings.Contains(result, today) {
		t.Errorf("GenerateDailyFrontMatter() output missing today's date %q", today)
	}
}

func TestUpdateDailyReport_EmptyData(t *testing.T) {
	// Test with empty issues and PRs
	tmpFile := t.TempDir() + "/daily.md"
	orgName := "test-org"
	publicRepos := make(map[string]struct{})
	var issues []github.Item
	var prs []github.Item

	err := UpdateDailyReport(tmpFile, orgName, publicRepos, issues, prs)
	if err != nil {
		t.Fatalf("UpdateDailyReport() returned error: %v", err)
	}
}

func TestUpdateDailyReport_WithData(t *testing.T) {
	// Test with sample issues and PRs
	tmpFile := t.TempDir() + "/daily.md"
	orgName := "test-org"
	publicRepos := make(map[string]struct{})

	issues := []github.Item{
		{
			Title: "增加 25 秋考试信息 (#39)",
			URL:   "https://github.com/test-org/repo/issues/1",
			Repository: github.Repository{
				Name: "test-repo",
			},
			Author: github.Author{
				Login: "testuser",
			},
			CreatedAt: "2026-02-13T10:00:00Z",
			Labels: []github.Label{
				{Name: "bug"},
				{Name: "enhancement"},
			},
		},
	}

	prs := []github.Item{
		{
			Title: "Fix [Parser] (#10)",
			URL:   "https://evil.com/steal",
			Repository: github.Repository{
				Name: "test-repo",
			},
			Author: github.Author{
				Login: "contributor",
			},
			CreatedAt: "2026-02-13T11:00:00Z",
			Labels: []github.Label{
				{Name: "feature"},
			},
		},
	}

	err := UpdateDailyReport(tmpFile, orgName, publicRepos, issues, prs)
	if err != nil {
		t.Fatalf("UpdateDailyReport() returned error: %v", err)
	}

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	got := string(content)

	if !strings.Contains(got, "### [增加 25 秋考试信息 \\(#39\\)](https://github.com/test-org/repo/issues/1)") {
		t.Errorf("expected issue link, got:\n%s", got)
	}

	if !strings.Contains(got, "### [Fix \\[Parser\\] \\(#10\\)](https://evil.com/steal)") {
		t.Errorf("expected PR link to keep original URL, got:\n%s", got)
	}
}

func TestUpdateDailyReport_EscapeMDXPayloadInTitles(t *testing.T) {
	tmpFile := t.TempDir() + "/daily.md"
	orgName := "test-org"
	publicRepos := make(map[string]struct{})

	issues := []github.Item{
		{
			Title: "<script>alert(1)</script>",
			URL:   "https://github.com/test-org/repo/issues/2",
			Repository: github.Repository{
				Name: "test-repo",
			},
			Author: github.Author{
				Login: "attacker",
			},
			CreatedAt: "2026-02-13T10:00:00Z",
		},
	}

	prs := []github.Item{
		{
			Title: "{alert(1)}",
			URL:   "https://github.com/test-org/repo/pull/2",
			Repository: github.Repository{
				Name: "test-repo",
			},
			Author: github.Author{
				Login: "attacker",
			},
			CreatedAt: "2026-02-13T11:00:00Z",
		},
	}

	if err := UpdateDailyReport(tmpFile, orgName, publicRepos, issues, prs); err != nil {
		t.Fatalf("UpdateDailyReport() returned error: %v", err)
	}

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	got := string(content)

	if strings.Contains(got, "<script>alert(1)</script>") {
		t.Fatalf("expected issue title to be escaped, got:\n%s", got)
	}
	if strings.Contains(got, "{alert(1)}") {
		t.Fatalf("expected pr title to be escaped, got:\n%s", got)
	}

	if !strings.Contains(got, "&lt;script&gt;alert\\(1\\)&lt;/script&gt;") {
		t.Fatalf("escaped issue title not found, got:\n%s", got)
	}
	if !strings.Contains(got, "&#123;alert\\(1\\)&#125;") {
		t.Fatalf("escaped pr title not found, got:\n%s", got)
	}
}
