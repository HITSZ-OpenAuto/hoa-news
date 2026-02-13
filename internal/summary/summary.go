// 周报总结
package summary

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/HITSZ-OpenAuto/hoa-news/internal/utils"
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

func GenerateWeeklyFrontMatter(startDate time.Time, now time.Time) (string, error) {
	title := fmt.Sprintf("AUTO 周报 %s - %s", startDate.Format("2006-01-02"), now.Format("2006-01-02"))
	description := fmt.Sprintf("本周报涵盖 %s 至 %s 期间的更新", startDate.Format("2006-01-02"), now.Format("2006-01-02"))
	date := time.Now().UTC().Format("2006-01-02")
	authors := []utils.Author{{
		Name:  "ChatGPT",
		Link:  "https://github.com/openai",
		Image: "https://github.com/openai.png",
	}}
	return utils.GenerateFrontMatter(title, date, description, authors)
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
