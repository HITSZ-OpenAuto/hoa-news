package utils

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestIsBot(t *testing.T) {
	tests := []struct {
		name        string
		authorName  string
		authorLogin string
		expected    bool
	}{
		{"GitHub Actions by name", "github-actions", "", true},
		{"GitHub Actions by login", "", "github-actions[bot]", true},
		{"Dependabot by name", "dependabot[bot]", "", true},
		{"Renovate by login", "", "renovate", true},
		{"Custom bot with [bot] suffix", "custom-bot[bot]", "", true},
		{"Human user", "John Doe", "johndoe", false},
		{"Empty strings", "", "", false},
		{"Case insensitive", "GITHUB-ACTIONS", "", true},
		{"Whitespace handling", "  github-actions  ", "", true},
		{"Both name and login are bots", "github-actions", "dependabot[bot]", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBot(tt.authorName, tt.authorLogin)
			if result != tt.expected {
				t.Errorf("IsBot(%q, %q) = %v, expected %v", tt.authorName, tt.authorLogin, result, tt.expected)
			}
		})
	}
}

func TestGenerateFrontMatter(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		date        string
		description string
		authors     []Author
		checkFields []string // Fields to check in YAML output
	}{
		{
			name:        "Weekly report with description",
			title:       "AUTO 周报 2026-02-06 - 2026-02-13",
			date:        "2026-02-13",
			description: "本周报涵盖 2026-02-06 至 2026-02-13 期间的更新",
			authors: []Author{{
				Name:  "ChatGPT",
				Link:  "https://github.com/openai",
				Image: "https://github.com/openai.png",
			}},
			checkFields: []string{"title:", "date:", "authors:", "description:", "excludeSearch:", "draft:"},
		},
		{
			name:        "Daily report with description",
			title:       "AUTO 更新速递",
			date:        "2026-02-13",
			description: "每日更新",
			authors: []Author{{
				Name:  "github-actions[bot]",
				Link:  "https://github.com/features/actions",
				Image: "https://avatars.githubusercontent.com/in/15368",
			}},
			checkFields: []string{"title:", "date:", "authors:", "description:", "excludeSearch:", "draft:"},
		},
		{
			name:        "Empty description",
			title:       "Test Report",
			date:        "2026-01-01",
			description: "",
			authors: []Author{{
				Name:  "Test User",
				Link:  "https://example.com",
				Image: "https://example.com/avatar.png",
			}},
			checkFields: []string{"title:", "date:", "authors:", "excludeSearch:", "draft:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateFrontMatter(tt.title, tt.date, tt.description, tt.authors)
			if err != nil {
				t.Fatalf("GenerateFrontMatter() returned error: %v", err)
			}

			// Check that all expected fields are present
			for _, field := range tt.checkFields {
				if !strings.Contains(result, field) {
					t.Errorf("GenerateFrontMatter() output missing field %q", field)
				}
			}

			// Check that title and date values are present
			if !strings.Contains(result, tt.title) {
				t.Errorf("GenerateFrontMatter() output missing title %q", tt.title)
			}
			if !strings.Contains(result, tt.date) {
				t.Errorf("GenerateFrontMatter() output missing date %q", tt.date)
			}

			// Check author name is present
			if len(tt.authors) > 0 && !strings.Contains(result, tt.authors[0].Name) {
				t.Errorf("GenerateFrontMatter() output missing author name %q", tt.authors[0].Name)
			}

			// Check description if not empty
			if tt.description != "" && !strings.Contains(result, tt.description) {
				t.Errorf("GenerateFrontMatter() output missing description %q", tt.description)
			}
		})
	}
}

func TestUTCToBJT(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Valid UTC time",
			input:    "2026-02-13T10:30:00Z",
			expected: "2026-02-13 18:30:00",
		},
		{
			name:     "UTC midnight",
			input:    "2026-02-13T00:00:00Z",
			expected: "2026-02-13 08:00:00",
		},
		{
			name:     "Invalid time format",
			input:    "invalid-time",
			expected: "invalid-time",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UTCToBJT(tt.input)
			if result != tt.expected {
				t.Errorf("UTCToBJT(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestWriteReport(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "Markdown report",
			content: "# Test Report\n\nThis is test content.",
		},
		{
			name:    "Empty content",
			content: "",
		},
		{
			name:    "Multi-line content",
			content: "# Header\n\n## Section 1\nContent here.\n\n## Section 2\nMore content.",
		},
		{
			name:    "Unicode content",
			content: "测试报告\n\n这是中文内容。",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file for testing
			tmpFile := t.TempDir() + "/test_report.md"

			err := WriteReport(tmpFile, tt.content)
			if err != nil {
				t.Fatalf("WriteReport() returned error: %v", err)
			}

			// Read the file back and verify content
			readContent, err := os.ReadFile(tmpFile)
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}

			if string(readContent) != tt.content {
				t.Errorf("WriteReport() wrote incorrect content.\nExpected: %q\nGot: %q", tt.content, string(readContent))
			}

			// Verify file permissions
			info, err := os.Stat(tmpFile)
			if err != nil {
				t.Fatalf("Failed to stat file: %v", err)
			}

			expectedPerm := os.FileMode(0o644)
			if info.Mode().Perm() != expectedPerm {
				t.Errorf("WriteReport() set incorrect permissions. Expected: %v, Got: %v", expectedPerm, info.Mode().Perm())
			}
		})
	}
}

func TestChineseWeekday(t *testing.T) {
	tests := []struct {
		name     string
		date     time.Time
		expected string
	}{
		{
			name:     "Monday",
			date:     time.Date(2026, 2, 9, 0, 0, 0, 0, time.UTC),
			expected: "周一",
		},
		{
			name:     "Tuesday",
			date:     time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC),
			expected: "周二",
		},
		{
			name:     "Wednesday",
			date:     time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC),
			expected: "周三",
		},
		{
			name:     "Thursday",
			date:     time.Date(2026, 2, 12, 0, 0, 0, 0, time.UTC),
			expected: "周四",
		},
		{
			name:     "Friday",
			date:     time.Date(2026, 2, 13, 0, 0, 0, 0, time.UTC),
			expected: "周五",
		},
		{
			name:     "Saturday",
			date:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
			expected: "周六",
		},
		{
			name:     "Sunday",
			date:     time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
			expected: "周日",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ChineseWeekday(tt.date)
			if result != tt.expected {
				t.Errorf("chineseWeekday(%v) = %q, expected %q", tt.date, result, tt.expected)
			}
		})
	}
}
