package report

import (
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
	tmpFile := t.TempDir() + "/daily.mdx"
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
	tmpFile := t.TempDir() + "/daily.mdx"
	orgName := "test-org"
	publicRepos := make(map[string]struct{})

	issues := []github.Item{
		{
			Title: "Test Issue",
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
			Title: "Test PR",
			URL:   "https://github.com/test-org/repo/pull/1",
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
}
