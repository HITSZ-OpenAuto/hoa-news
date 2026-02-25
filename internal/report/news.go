// 更新速递，更新最近一天的 commit/PR/issue
package report

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/HITSZ-OpenAuto/hoa-news/internal/github"
	"github.com/HITSZ-OpenAuto/hoa-news/internal/utils"
)

// 复用 summary.go 的 CommitEntry

const newsGoroutineLimit = 10 // 并发限制，避免过多协程导致触发 GitHub 限流

func News(orgName string, publicRepos map[string]struct{}) {
	issues, err := github.SearchIssues(orgName, 100)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get issues: %v\n", err)
		os.Exit(1)
	}
	log.Printf("Fetched issues: %d", len(issues))
	prs, err := github.SearchPullRequests(orgName, 100)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get pull requests: %v\n", err)
		os.Exit(1)
	}
	log.Printf("Fetched pull requests: %d", len(prs))

	issues = filterByPublicRepos(issues, publicRepos)
	prs = filterByPublicRepos(prs, publicRepos)
	log.Printf("Filtered by public repos, issues=%d, pull requests=%d", len(issues), len(prs))

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

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		commits = make([]CommitEntry, 0)
		goLimit = make(chan struct{}, newsGoroutineLimit) // 限制并发数，避免过多请求导致失败
	)

	for repo := range publicRepos {
		wg.Add(1)
		go func(repo string) {
			defer wg.Done()

			goLimit <- struct{}{}        // 获取限流令牌
			defer func() { <-goLimit }() // 释放令牌

			repoCommits, err := github.ListCommitsSince(orgName, repo, startTime.Format(time.RFC3339))
			if err != nil {
				log.Printf("Failed to fetch commits for %s: %v", repo, err)
				return
			}

			localCommits := make([]CommitEntry, 0, len(repoCommits)) // 区分本地和全局，是为了减少锁的粒度，提升性能
			for _, commit := range repoCommits {
				authorName := commit.Commit.Author.Name
				authorLogin := ""
				if commit.Author != nil {
					authorLogin = commit.Author.Login
				}
				if utils.IsBot(authorName, authorLogin) {
					continue // 过滤掉 bot 提交，比如 actions 自动生成的就不需要计数
				}

				date, err := time.Parse(time.RFC3339, commit.Commit.Author.Date)
				if err != nil {
					continue
				}
				date = date.In(beijingTimeZone) // Convert to BJT
				localCommits = append(localCommits, CommitEntry{
					AuthorName:  authorName,
					AuthorLogin: authorLogin,
					Date:        date,
					Message:     commit.Commit.Message,
					RepoName:    repo,
				})
			}

			if len(localCommits) == 0 {
				log.Printf("Finished commits process for %s (no valid commits)", repo)
				return
			}

			mu.Lock()
			commits = append(commits, localCommits...)
			mu.Unlock()

			log.Printf("Finished commits process for %s", repo)
		}(repo)
	}

	wg.Wait()

	// 按日期降序排序保证输出稳定，最新的 commit 在前面
	sort.Slice(commits, func(i, j int) bool {
		return commits[i].Date.After(commits[j].Date)
	})

	log.Printf("Commit collection complete, %d total valid commits", len(commits))

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
