package news

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type CommitEntry struct {
	AuthorName  string
	AuthorLogin string
	Date        time.Time
	Message     string
	RepoName    string
}

var weekdays = []string{"周一", "周二", "周三", "周四", "周五", "周六", "周日"}

func chineseWeekday(t time.Time) string {
	return weekdays[int(t.Weekday()+6)%7]
}

func BuildMarkdown(commits []CommitEntry, repoTitles map[string]string, orgName string) string {
	if len(commits) == 0 {
		return ""
	}
	sort.Slice(commits, func(i, j int) bool {
		return commits[i].Date.After(commits[j].Date)
	})
	var b strings.Builder
	b.WriteString("## 更新内容\n\n")

	var prevDate string
	for _, commit := range commits {
		dateStr := commit.Date.Format("2006-01-02")
		if dateStr != prevDate {
			b.WriteString(fmt.Sprintf("### %s (%d.%d)\n\n", chineseWeekday(commit.Date), commit.Date.Month(), commit.Date.Day()))
			prevDate = dateStr
		}
		title := repoTitles[commit.RepoName]
		if title == "" {
			title = commit.RepoName
		}
		message := strings.Split(commit.Message, "\n")[0]
		b.WriteString(fmt.Sprintf("- %s 在 [%s](https://github.com/%s/%s) 中提交了信息：%s\n\n", commit.AuthorName, title, orgName, commit.RepoName, message))
	}
	return b.String()
}

func WriteReport(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

type frontMatterWeekly struct {
	Title         string   `yaml:"title"`
	Date          string   `yaml:"date"`
	Authors       []author `yaml:"authors"`
	ExcludeSearch bool     `yaml:"excludeSearch"`
	Draft         bool     `yaml:"draft"`
}

type frontMatterDaily struct {
	Title         string   `yaml:"title"`
	Date          string   `yaml:"date"`
	Authors       []author `yaml:"authors"`
	Description   string   `yaml:"description"`
	ExcludeSearch bool     `yaml:"excludeSearch"`
	Draft         bool     `yaml:"draft"`
}

type author struct {
	Name  string `yaml:"name"`
	Link  string `yaml:"link"`
	Image string `yaml:"image"`
}

func GenerateWeeklyFrontMatter(startDate time.Time, now time.Time) (string, error) {
	fm := frontMatterWeekly{
		Title: fmt.Sprintf("AUTO 周报 %s - %s", startDate.Format("2006-01-02"), now.Format("2006-01-02")),
		Date:  time.Now().UTC().Format("2006-01-02"),
		Authors: []author{{
			Name:  "ChatGPT",
			Link:  "https://github.com/openai",
			Image: "https://github.com/openai.png",
		}},
		ExcludeSearch: false,
		Draft:         false,
	}
	out, err := yaml.Marshal(&fm)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func GenerateDailyFrontMatter() (string, error) {
	fm := frontMatterDaily{
		Title: "AUTO 更新速递",
		Date:  time.Now().UTC().Format("2006-01-02"),
		Authors: []author{{
			Name:  "github-actions[bot]",
			Link:  "https://github.com/features/actions",
			Image: "https://avatars.githubusercontent.com/in/15368",
		}},
		Description:   "每日更新",
		ExcludeSearch: false,
		Draft:         false,
	}
	out, err := yaml.Marshal(&fm)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func WriteWeeklyIndex(path string, now time.Time) error {
	fm := struct {
		Title       string `yaml:"title"`
		Date        string `yaml:"date"`
		Description string `yaml:"description"`
	}{
		Title:       "AUTO 周报",
		Date:        time.Now().UTC().Format("2006-01-02"),
		Description: fmt.Sprintf("AUTO 周报是由 ChatGPT 每周五发布的一份简报，最近更新于 %s。", now.Format("2006-01-02")),
	}
	out, err := yaml.Marshal(&fm)
	if err != nil {
		return err
	}
	content := fmt.Sprintf("---\n%s---\n", string(out))
	return os.WriteFile(path, []byte(content), 0o644)
}
