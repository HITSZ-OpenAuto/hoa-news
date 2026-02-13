// 共享工具函数和结构体
package utils

import (
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var knownBots = map[string]struct{}{
	"GitHub Actions":      {},
	"github-actions":      {},
	"actions-user":        {},
	"github-actions[bot]": {},
	"dependabot":          {},
	"dependabot[bot]":     {},
	"renovate":            {},
	"renovate[bot]":       {},
}

// IsBot checks if a commit author is a bot
func IsBot(authorName, authorLogin string) bool {
	name := strings.ToLower(strings.TrimSpace(authorName))
	login := strings.ToLower(strings.TrimSpace(authorLogin))
	if name != "" {
		if _, ok := knownBots[name]; ok {
			return true
		}
		if strings.HasSuffix(name, "[bot]") {
			return true
		}
	}
	if login != "" {
		if _, ok := knownBots[login]; ok {
			return true
		}
		if strings.HasSuffix(login, "[bot]") {
			return true
		}
	}
	return false
}

// WriteReport writes content to a file
func WriteReport(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

// Author represents a document author
type Author struct {
	Name  string `yaml:"name"`
	Link  string `yaml:"link"`
	Image string `yaml:"image"`
}

// FrontMatter represents the unified front matter structure
type FrontMatter struct {
	Title         string   `yaml:"title"`
	Date          string   `yaml:"date"`
	Authors       []Author `yaml:"authors"`
	Description   string   `yaml:"description,omitempty"`
	ExcludeSearch bool     `yaml:"excludeSearch"`
	Draft         bool     `yaml:"draft"`
}

// GenerateFrontMatter generates a YAML front matter string
func GenerateFrontMatter(title, date, description string, authors []Author) (string, error) {
	fm := FrontMatter{
		Title:         title,
		Date:          date,
		Authors:       authors,
		Description:   description,
		ExcludeSearch: false,
		Draft:         false,
	}
	out, err := yaml.Marshal(&fm)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// UTCToBJT converts UTC time string to Beijing time string
func UTCToBJT(utcTimeStr string) string {
	utcTime, err := time.Parse(time.RFC3339, utcTimeStr)
	if err != nil {
		return utcTimeStr
	}
	bjt := utcTime.Add(8 * time.Hour)
	return bjt.Format("2006-01-02 15:04:05")
}
