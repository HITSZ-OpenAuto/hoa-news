package report

import (
	"strings"
	"testing"
	"time"
)

func TestBuildMarkdown(t *testing.T) {
	repoTitles := map[string]string{
		"MATH1002":  "高等数学A（下）",
		"CS1001":    "程序设计基础",
		"PHYS1001A": "大学物理A（上）",
	}
	orgName := "HITSZ-OpenAuto"

	tests := []struct {
		name     string
		commits  []CommitEntry
		expected []string // Strings that should appear in output
	}{
		{
			name:     "Empty commits",
			commits:  []CommitEntry{},
			expected: []string{},
		},
		{
			name: "Single commit",
			commits: []CommitEntry{
				{
					AuthorName:  "张三",
					AuthorLogin: "zhangsan",
					Date:        time.Date(2026, 2, 13, 10, 30, 0, 0, time.UTC),
					Message:     "更新实验报告",
					RepoName:    "MATH1002",
				},
			},
			expected: []string{
				"## 更新内容",
				"张三",
				"高等数学A（下）",
				"更新实验报告",
				"https://github.com/HITSZ-OpenAuto/MATH1002",
			},
		},
		{
			name: "Multiple commits on same day",
			commits: []CommitEntry{
				{
					AuthorName:  "李四",
					AuthorLogin: "lisi",
					Date:        time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC),
					Message:     "添加课件",
					RepoName:    "CS1001",
				},
				{
					AuthorName:  "王五",
					AuthorLogin: "wangwu",
					Date:        time.Date(2026, 2, 13, 14, 30, 0, 0, time.UTC),
					Message:     "修复错误",
					RepoName:    "PHYS1001A",
				},
			},
			expected: []string{
				"## 更新内容",
				"李四",
				"程序设计基础",
				"添加课件",
				"王五",
				"大学物理A（上）",
				"修复错误",
			},
		},
		{
			name: "Commits on different days",
			commits: []CommitEntry{
				{
					AuthorName:  "用户A",
					AuthorLogin: "usera",
					Date:        time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC),
					Message:     "周四的更新",
					RepoName:    "MATH1002",
				},
				{
					AuthorName:  "用户B",
					AuthorLogin: "userb",
					Date:        time.Date(2026, 2, 12, 15, 0, 0, 0, time.UTC),
					Message:     "周三的更新",
					RepoName:    "CS1001",
				},
			},
			expected: []string{
				"## 更新内容",
				"周四的更新",
				"周三的更新",
			},
		},
		{
			name: "Repo without title mapping",
			commits: []CommitEntry{
				{
					AuthorName:  "测试用户",
					AuthorLogin: "testuser",
					Date:        time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC),
					Message:     "测试提交",
					RepoName:    "UNKNOWN_REPO",
				},
			},
			expected: []string{
				"## 更新内容",
				"测试用户",
				"UNKNOWN_REPO", // Should use repo name when no title
				"测试提交",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildMarkdown(tt.commits, repoTitles, orgName)

			if len(tt.commits) == 0 {
				if result != "" {
					t.Errorf("BuildMarkdown() with empty commits should return empty string, got %q", result)
				}
				return
			}

			for _, expected := range tt.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("BuildMarkdown() output missing expected string %q\nGot:\n%s", expected, result)
				}
			}
		})
	}
}

func TestGenerateWeeklyFrontMatter(t *testing.T) {
	startDate := time.Date(2026, 2, 6, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 2, 13, 0, 0, 0, 0, time.UTC)

	result, err := GenerateWeeklyFrontMatter(startDate, endDate)
	if err != nil {
		t.Fatalf("GenerateWeeklyFrontMatter() returned error: %v", err)
	}

	// Check required fields
	requiredFields := []string{
		"title:",
		"AUTO 周报 2026-02-06 - 2026-02-13",
		"date:",
		"authors:",
		"ChatGPT",
		"description:",
		"涵盖 2026-02-06 至 2026-02-13 的更新",
		"excludeSearch:",
		"draft:",
	}

	for _, field := range requiredFields {
		if !strings.Contains(result, field) {
			t.Errorf("GenerateWeeklyFrontMatter() output missing %q", field)
		}
	}
}

func TestWriteWeeklyIndex(t *testing.T) {
	tmpFile := t.TempDir() + "/_index.zh-cn.md"
	now := time.Date(2026, 2, 13, 12, 0, 0, 0, time.UTC)

	err := WriteWeeklyIndex(tmpFile, now)
	if err != nil {
		t.Fatalf("WriteWeeklyIndex() returned error: %v", err)
	}
}

func TestBuildMarkdown_Sorting(t *testing.T) {
	// Test that commits are sorted by date (newest first)
	commits := []CommitEntry{
		{
			AuthorName: "User A",
			Date:       time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC),
			Message:    "Old commit",
			RepoName:   "repo1",
		},
		{
			AuthorName: "User B",
			Date:       time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC),
			Message:    "New commit",
			RepoName:   "repo2",
		},
		{
			AuthorName: "User C",
			Date:       time.Date(2026, 2, 11, 10, 0, 0, 0, time.UTC),
			Message:    "Middle commit",
			RepoName:   "repo3",
		},
	}

	result := BuildMarkdown(commits, make(map[string]string), "test-org")

	// Verify that "New commit" appears before "Old commit" in the output
	newIndex := strings.Index(result, "New commit")
	oldIndex := strings.Index(result, "Old commit")

	if newIndex == -1 || oldIndex == -1 {
		t.Fatalf("BuildMarkdown() output missing commit messages")
	}

	if newIndex > oldIndex {
		t.Errorf("BuildMarkdown() commits not sorted correctly: newer commit should appear first")
	}
}
