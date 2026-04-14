// 更新速递，更新最近一天的 commit/PR/issue
package report

import (
	"errors"
	"fmt"
	"io/fs"
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

const newsGoroutineLimit = 80 // 并发限制，避免过多协程触发 GitHub 限流

func Daily(orgName string, publicRepos map[string]struct{}) error {
	issues, err := github.SearchIssues(orgName, 150)
	if err != nil {
		return fmt.Errorf("failed to get issues: %w", err)
	}
	log.Printf("Fetched issues: %d", len(issues))
	prs, err := github.SearchPullRequests(orgName, 150)
	if err != nil {
		return fmt.Errorf("failed to get pull requests: %w", err)
	}
	log.Printf("Fetched pull requests: %d", len(prs))

	issues = filterByPublicRepos(issues, publicRepos)
	prs = filterByPublicRepos(prs, publicRepos)
	log.Printf("Filtered by public repos, issues=%d, pull requests=%d", len(issues), len(prs))

	issues = filterBracketedIssues(issues)
	log.Printf("Filtered bracketed issues, issues=%d", len(issues))

	if err := UpdateDailyReport("news/daily.md", orgName, publicRepos, issues, prs); err != nil {
		return fmt.Errorf("failed to update daily report: %w", err)
	}

	return nil
}

func UpdateDailyReport(path string, orgName string, publicRepos map[string]struct{}, issues []github.Item, prs []github.Item) error {
	startTime := time.Now().Add(-24 * time.Hour)

	var (
		mu        sync.Mutex
		wg        sync.WaitGroup
		commits   = make([]CommitEntry, 0)
		repoNames = make(map[string]string)                 // repo 名 -> 课程名的映射
		goLimit   = make(chan struct{}, newsGoroutineLimit) // 限制并发数，避免过多请求导致失败
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
				if !utils.IsChineseCommit(commit.Commit.Message) {
					continue // 只保留中文提交，过滤代码提交等非中文信息
				}

				date, err := time.Parse(time.RFC3339, commit.Commit.Author.Date)
				if err != nil {
					continue
				}
				date = date.In(utils.BeijingTimeZone) // Convert to BJT
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

			var courseName string
			if name, err := fetchCourseName(orgName, repo); err == nil && name != "" {
				courseName = name
			}

			mu.Lock()
			commits = append(commits, localCommits...)
			if courseName != "" {
				repoNames[repo] = courseName
			}
			mu.Unlock()

			log.Printf("Finished commits process for %s", repo)
		}(repo)
	}
	wg.Wait()

	log.Printf("Commit collection complete, %d total valid commits", len(commits))

	body := buildDailyBody(orgName, commits, repoNames, issues, prs)

	oldContent, readErr := os.ReadFile(path)
	if readErr == nil {
		// If no commits today, preserve old commits section instead of showing 暂无更新
		if len(commits) == 0 {
			oldBody := extractBody(string(oldContent))
			if oldSection := extractSection(oldBody, "## 最近更新"); oldSection != "" {
				body = strings.Replace(body, "## 最近更新\n\n暂无更新\n\n", "## 最近更新\n\n"+oldSection, 1)
			}
		}
		if isSubstantivelyEqual(string(oldContent), body) {
			log.Printf("Daily report body unchanged, skip rewriting %s", path)
			return nil
		}
	} else if !errors.Is(readErr, fs.ErrNotExist) {
		return fmt.Errorf("failed to read existing daily report %q: %w", path, readErr)
	}

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

	var final strings.Builder
	final.WriteString("---\n")
	final.WriteString(fm)
	final.WriteString("---\n\n")
	final.WriteString(body)

	return os.WriteFile(path, []byte(final.String()), 0o644)
}

func buildDailyBody(orgName string, commits []CommitEntry, repoNames map[string]string, issues []github.Item, prs []github.Item) string {
	// 按照确定的规则进行排序，确保对于相同的更新内容，生成的报告内容顺序一致
	// 便于后续比对前后的内容
	sortCommits(commits)
	sortItems(issues)
	sortItems(prs)

	var buf strings.Builder

	// Commits
	buf.WriteString("## 最近更新\n\n")
	if len(commits) == 0 {
		buf.WriteString("暂无更新\n\n")
	} else {
		for _, commit := range commits {
			author := utils.SanitizeInlineText(commit.AuthorName)
			repoName := repoNames[commit.RepoName]
			if repoName == "" {
				repoName = commit.RepoName
			}
			repoName = utils.SanitizeInlineText(repoName)
			message := utils.SanitizeInlineText(strings.Split(commit.Message, "\n")[0])
			fmt.Fprintf(&buf,
				"- %s 在 [%s](https://github.com/%s/%s) 中提交了信息：%s (%s)\n\n",
				author, repoName, orgName, commit.RepoName, message, commit.Date.Format("15:04"))
		}
	}

	// Issues
	buf.WriteString("## 待解决的 Issues\n\n")
	if len(issues) == 0 {
		buf.WriteString("暂无待解决的 Issues\n\n")
	} else {
		for _, issue := range issues {
			fmt.Fprintf(&buf, "### %s\n\n", utils.RenderSafeMarkdownLink(issue.Title, issue.URL))
			fmt.Fprintf(&buf, "- **仓库**: %s\n", utils.SanitizeInlineText(issue.Repository.Name))
			fmt.Fprintf(&buf, "- **创建于**: %s\n", utils.UTCToBJT(issue.CreatedAt))
			fmt.Fprintf(&buf, "- **作者**: %s\n", utils.SanitizeInlineText(issue.Author.Login))
			if len(issue.Labels) > 0 {
				labels := make([]string, 0, len(issue.Labels))
				for _, label := range issue.Labels {
					labels = append(labels, utils.SanitizeInlineText(label.Name))
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
			fmt.Fprintf(&buf, "### %s\n\n", utils.RenderSafeMarkdownLink(pr.Title, pr.URL))
			fmt.Fprintf(&buf, "- **仓库**: %s\n", utils.SanitizeInlineText(pr.Repository.Name))
			fmt.Fprintf(&buf, "- **创建于**: %s\n", utils.UTCToBJT(pr.CreatedAt))
			fmt.Fprintf(&buf, "- **作者**: %s\n", utils.SanitizeInlineText(pr.Author.Login))
			if len(pr.Labels) > 0 {
				labels := make([]string, 0, len(pr.Labels))
				for _, label := range pr.Labels {
					labels = append(labels, utils.SanitizeInlineText(label.Name))
				}
				fmt.Fprintf(&buf, "- **标签**: %s\n", strings.Join(labels, ", "))
			}
			buf.WriteString("\n")
		}
	}

	return buf.String()
}

// 排序优先级：按照时间（从新到旧）、仓库名、提交信息、作者名、作者登录名排序
func sortCommits(commits []CommitEntry) {
	sort.Slice(commits, func(i, j int) bool {
		if !commits[i].Date.Equal(commits[j].Date) {
			return commits[i].Date.After(commits[j].Date)
		}
		if commits[i].RepoName != commits[j].RepoName {
			return commits[i].RepoName < commits[j].RepoName
		}

		// TODO 可以进行信息预处理，降低排序时的计算量
		mi := strings.Split(commits[i].Message, "\n")[0]
		mj := strings.Split(commits[j].Message, "\n")[0]
		if mi != mj {
			return mi < mj
		}
		if commits[i].AuthorName != commits[j].AuthorName {
			return commits[i].AuthorName < commits[j].AuthorName
		}
		return commits[i].AuthorLogin < commits[j].AuthorLogin
	})
}

// 对 Issues 和 Pull Requests 按照创建时间（从新到旧）、仓库名、标题、URL 进行排序。
func sortItems(items []github.Item) {
	sort.Slice(items, func(i, j int) bool {
		// TODO 可以进行信息预处理，降低排序时的计算量
		ti, okI := parseCreatedAt(items[i].CreatedAt)
		tj, okJ := parseCreatedAt(items[j].CreatedAt)
		if okI && okJ && !ti.Equal(tj) {
			return ti.After(tj)
		} else if okI != okJ {
			return okI
		}

		if items[i].Repository.Name != items[j].Repository.Name {
			return items[i].Repository.Name < items[j].Repository.Name
		}
		if items[i].Title != items[j].Title {
			return items[i].Title < items[j].Title
		}
		if items[i].URL != items[j].URL {
			return items[i].URL < items[j].URL
		}
		return items[i].CreatedAt > items[j].CreatedAt
	})
}

// 解析 RFC3339 格式的 CreatedAt 字符串字段，返回时间和解析是否成功的标志。
func parseCreatedAt(s string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// filterBracketedIssues 过滤掉标题完全被 【】 包裹的 issue（例如 【AUTO3000】），这类 issue 通常是自动生成的占位符。
func filterBracketedIssues(issues []github.Item) []github.Item {
	filtered := make([]github.Item, 0, len(issues))
	for _, issue := range issues {
		if strings.HasPrefix(issue.Title, "【") && strings.HasSuffix(issue.Title, "】") {
			continue
		}
		filtered = append(filtered, issue)
	}
	return filtered
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

// 判断日报内容是否实质相同：提取内容主体并规范化后进行字符串的相等比对，避免因为格式差异导致误判
func isSubstantivelyEqual(oldContent, newBody string) bool {
	return normalizeBody(extractBody(oldContent)) == normalizeBody(newBody)
}

// 提取日报内容主体：去掉可能存在的 front matter 和前后的空白，便于比对内容是否实质变化
func extractBody(content string) string {
	content = normalizeNewlines(content)
	if !strings.HasPrefix(content, "---\n") {
		return content
	}

	rest := content[len("---\n"):]
	offset := 0
	for {
		lineEnd := strings.IndexByte(rest[offset:], '\n')
		if lineEnd == -1 {
			if rest[offset:] == "---" {
				return ""
			}
			return content
		}
		line := rest[offset : offset+lineEnd]
		if line == "---" {
			return rest[offset+lineEnd+1:]
		}
		offset += lineEnd + 1
		if offset >= len(rest) {
			return content
		}
	}
}

// 规范化内容：统一换行符、去掉行尾空白、去掉前后多余空行，确保内容实质相同的情况下不会因为格式差异导致比对结果不同
func normalizeBody(content string) string {
	content = normalizeNewlines(content)
	if content == "" {
		return ""
	}

	lines := strings.Split(content, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	normalized := strings.Join(lines, "\n")
	normalized = strings.Trim(normalized, "\n")
	if normalized == "" {
		return ""
	}
	return normalized + "\n"
}

func normalizeNewlines(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	return strings.ReplaceAll(content, "\r", "\n")
}

// extractSection extracts the content of a section from a body, given its header (e.g. "## 最近更新").
// Returns the content after the header up to the next "## " section, or empty string if not found.
func extractSection(body, header string) string {
	target := header + "\n\n"
	idx := strings.Index(body, target)
	if idx == -1 {
		return ""
	}
	rest := body[idx+len(target):]
	nextHeader := strings.Index(rest, "\n## ")
	if nextHeader == -1 {
		return rest
	}
	return rest[:nextHeader+1]
}
