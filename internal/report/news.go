// 更新速递，更新最近一天的 commit/PR/issue
package report

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/HITSZ-OpenAuto/hoa-news/internal/github"
	"github.com/HITSZ-OpenAuto/hoa-news/internal/utils"
)

// 复用 summary.go 的 CommitEntry

func News(orgName string, publicRepos map[string]struct{}) {
	issues, err := github.SearchIssues(orgName, 100)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get issues: %v\n", err)
		os.Exit(1)
	}
	prs, err := github.SearchPullRequests(orgName, 100)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get pull requests: %v\n", err)
		os.Exit(1)
	}

	issues = filterByPublicRepos(issues, publicRepos)
	prs = filterByPublicRepos(prs, publicRepos)

	if err := UpdateDailyReport("news/daily.mdx", orgName, publicRepos, issues, prs); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to update daily report: %v\n", err)
		os.Exit(1)
	}
}

func UpdateDailyReport(path string, orgName string, publicRepos map[string]struct{}, issues []github.Item, prs []github.Item) error {
	var buf strings.Builder

	// Front matter
	fm, err := utils.GenerateFrontMatter(
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
		return fmt.Errorf("failed to generate front matter: %w", err)
	}

	buf.WriteString("---\n")
	buf.WriteString(fm)
	buf.WriteString("---\n\n")

	// Commits
	buf.WriteString("## 今日更新\n\n")

	startTime := time.Now().Add(-24 * time.Hour)
	commits := make([]CommitEntry, 0)

	for repo := range publicRepos {
		repoCommits, err := github.ListCommitsSince(orgName, repo, startTime.Format(time.RFC3339))
		if err != nil {
			log.Printf("Failed to fetch commits for %s: %v", repo, err)
			continue
		}
		for _, commit := range repoCommits {
			authorName := commit.Commit.Author.Name
			authorLogin := ""
			if commit.Author != nil {
				authorLogin = commit.Author.Login
			}
			if utils.IsBot(authorName, authorLogin) {
				continue
			}
			date, err := time.Parse(time.RFC3339, commit.Commit.Author.Date)
			if err != nil {
				continue
			}
			date = date.Add(8 * time.Hour) // Convert to BJT
			commits = append(commits, CommitEntry{
				AuthorName:  authorName,
				AuthorLogin: authorLogin,
				Date:        date,
				Message:     commit.Commit.Message,
				RepoName:    repo,
			})
		}
	}

	if len(commits) == 0 {
		buf.WriteString("暂无更新\n\n")
	} else {
		for _, commit := range commits {
			message := strings.Split(commit.Message, "\n")[0]
			fmt.Fprintf(&buf,
				"- %s 在 [%s](https://github.com/%s/%s) 中提交了信息：%s (%s)\n\n",
				commit.AuthorName, commit.RepoName, orgName, commit.RepoName, message,
				commit.Date.Format("15:04"))
		}
	}

	// Issues
	buf.WriteString("## 待解决的 Issues\n\n")

	if len(issues) == 0 {
		buf.WriteString("暂无待解决的 Issues\n\n")
	} else {
		for _, issue := range issues {
			fmt.Fprintf(&buf, "### [%s](%s)\n\n", issue.Title, issue.URL)
			fmt.Fprintf(&buf, "- **仓库**: %s\n", issue.Repository.Name)
			fmt.Fprintf(&buf, "- **创建于**: %s\n", utils.UTCToBJT(issue.CreatedAt))
			fmt.Fprintf(&buf, "- **作者**: %s\n", issue.Author.Login)
			if len(issue.Labels) > 0 {
				labels := make([]string, 0, len(issue.Labels))
				for _, label := range issue.Labels {
					labels = append(labels, label.Name)
				}
				fmt.Fprintf(&buf, "- **标签**: %s\n", strings.Join(labels, ", "))
			}
			buf.WriteString("\n")
		}
	}

	// Pull Requests
	buf.WriteString("## 待合并的 Pull Requests\n\n")

	if len(prs) == 0 {
		buf.WriteString("暂无待合并的 Pull Requests\n\n")
	} else {
		for _, pr := range prs {
			fmt.Fprintf(&buf, "### [%s](%s)\n\n", pr.Title, pr.URL)
			fmt.Fprintf(&buf, "- **仓库**: %s\n", pr.Repository.Name)
			fmt.Fprintf(&buf, "- **创建于**: %s\n", utils.UTCToBJT(pr.CreatedAt))
			fmt.Fprintf(&buf, "- **作者**: %s\n", pr.Author.Login)
			if len(pr.Labels) > 0 {
				labels := make([]string, 0, len(pr.Labels))
				for _, label := range pr.Labels {
					labels = append(labels, label.Name)
				}
				fmt.Fprintf(&buf, "- **标签**: %s\n", strings.Join(labels, ", "))
			}
			buf.WriteString("\n")
		}
	}

	return os.WriteFile(path, []byte(buf.String()), 0o644)
}

func filterByPublicRepos(items []github.Item, publicRepos map[string]struct{}) []github.Item {
	if len(publicRepos) == 0 {
		return items
	}
	filtered := make([]github.Item, 0, len(items))
	for _, item := range items {
		if _, ok := publicRepos[item.Repository.Name]; ok {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
