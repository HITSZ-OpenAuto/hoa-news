package report

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/HITSZ-OpenAuto/hoa-news/internal/github"
)

const timeZoneOffset = 8 * time.Hour

func utcToBJT(utcTimeStr string) string {
	utcTime, err := time.Parse(time.RFC3339, utcTimeStr)
	if err != nil {
		return utcTimeStr
	}
	bjt := utcTime.Add(timeZoneOffset)
	return bjt.Format("2006-01-02 15:04:05")
}

func UpdateDailyReport(path string, issues []github.Item, prs []github.Item) error {
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(contentBytes)
	if idx := strings.Index(content, "## 待解决的 Issues"); idx >= 0 {
		content = content[:idx]
	}

	var buf bytes.Buffer
	buf.WriteString(content)
	buf.WriteString("## 待解决的 Issues\n\n")

	if len(issues) == 0 {
		buf.WriteString("暂无待解决的 Issues\n\n")
	} else {
		for _, issue := range issues {
			buf.WriteString(fmt.Sprintf("### [%s](%s)\n\n", issue.Title, issue.URL))
			buf.WriteString(fmt.Sprintf("- **仓库**: %s\n", issue.Repository.Name))
			buf.WriteString(fmt.Sprintf("- **创建于**: %s\n", utcToBJT(issue.CreatedAt)))
			buf.WriteString(fmt.Sprintf("- **作者**: %s\n", issue.Author.Login))
			if len(issue.Labels) > 0 {
				labels := make([]string, 0, len(issue.Labels))
				for _, label := range issue.Labels {
					labels = append(labels, label.Name)
				}
				buf.WriteString(fmt.Sprintf("- **标签**: %s\n", strings.Join(labels, ", ")))
			}
			buf.WriteString("\n")
		}
	}

	buf.WriteString("## 待合并的 Pull Requests\n\n")

	if len(prs) == 0 {
		buf.WriteString("暂无待合并的 Pull Requests\n\n")
	} else {
		for _, pr := range prs {
			buf.WriteString(fmt.Sprintf("### [%s](%s)\n\n", pr.Title, pr.URL))
			buf.WriteString(fmt.Sprintf("- **仓库**: %s\n", pr.Repository.Name))
			buf.WriteString(fmt.Sprintf("- **创建于**: %s\n", utcToBJT(pr.CreatedAt)))
			buf.WriteString(fmt.Sprintf("- **作者**: %s\n", pr.Author.Login))
			if len(pr.Labels) > 0 {
				labels := make([]string, 0, len(pr.Labels))
				for _, label := range pr.Labels {
					labels = append(labels, label.Name)
				}
				buf.WriteString(fmt.Sprintf("- **标签**: %s\n", strings.Join(labels, ", ")))
			}
			buf.WriteString("\n")
		}
	}

	return os.WriteFile(path, buf.Bytes(), 0o644)
}
