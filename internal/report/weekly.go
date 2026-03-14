// 周报总结
package report

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/HITSZ-OpenAuto/hoa-news/internal/github"
	"github.com/HITSZ-OpenAuto/hoa-news/internal/openai"
	"github.com/HITSZ-OpenAuto/hoa-news/internal/utils"
	"gopkg.in/yaml.v3"
)

const summaryGoroutineLimit = 20 // 并发限制，避免过多协程导致触发 GitHub 限流

var ErrNoWeeklyCommits = errors.New("no commits found in the given period of time")

// CommitEntry 表示一条 commit 记录。
type CommitEntry struct {
	AuthorName  string
	AuthorLogin string
	Date        time.Time
	Message     string
	RepoName    string
}

// SummaryContext 保存一次周报生成的运行上下文，
// 包括时间窗口（UTC/BJT）和输出路径，避免各阶段重复计算。
type SummaryContext struct {
	NowBJT          time.Time // 当前北京时间（UTC+8）
	StartTime       time.Time // commit 查询起始时间
	WeeklyDir       string    // 周报输出目录，如 news/weekly/weekly-2026-02-15
	ReportPath      string    // 周报文件路径，如 ./index.md
	WeeklyIndexPath string    // 周报索引文件路径
}

// WeeklyAggregate 保存数据聚合阶段的结果，
// 供后续渲染 markdown 和生成摘要使用。
type WeeklyAggregate struct {
	Commits  []CommitEntry     // 过滤 bot 后的 commit 列表
	RepoName map[string]string // repo 名 -> 课程名的映射（无课程名则回退为 repo 名）
}

// Summary 是周报生成的入口函数，编排流程：
// 构建上下文 → 聚合数据 → 渲染内容 → 写入文件。
func Weekly(orgName string, publicRepos map[string]struct{}) error {
	ctx := buildSummaryContext(time.Now().UTC())
	agg := collectWeeklyData(ctx, orgName, publicRepos)

	if len(agg.Commits) == 0 {
		return ErrNoWeeklyCommits
	}

	frontMatter, err := GenerateWeeklyFrontMatter(ctx.StartTime, ctx.NowBJT)
	if err != nil {
		return fmt.Errorf("failed to generate front matter: %w", err)
	}
	markdownReport := BuildMarkdown(agg.Commits, agg.RepoName, orgName)

	summarySection := generateSummarySection(markdownReport)

	var finalReport strings.Builder
	fmt.Fprintf(&finalReport, "---\n%s---\n\n", frontMatter)
	if summarySection != "" {
		finalReport.WriteString(summarySection)
		finalReport.WriteString("\n\n")
	}
	finalReport.WriteString(markdownReport)

	if err := os.MkdirAll(ctx.WeeklyDir, 0o755); err != nil {
		return fmt.Errorf("failed to create weekly directory %q: %w", ctx.WeeklyDir, err)
	}
	if err := os.WriteFile(ctx.ReportPath, []byte(finalReport.String()), 0o644); err != nil {
		return fmt.Errorf("failed to write weekly report %q: %w", ctx.ReportPath, err)
	}

	if err := WriteWeeklyIndex(ctx.WeeklyIndexPath, ctx.NowBJT); err != nil {
		return fmt.Errorf("failed to update weekly index %q: %w", ctx.WeeklyIndexPath, err)
	}

	return nil
}

// buildSummaryContext 根据当前 UTC 时间计算时间窗口和输出路径，
// 返回贯穿整个流程的 SummaryContext。
func buildSummaryContext(nowUTC time.Time) SummaryContext {
	nowBJT := nowUTC.In(utils.BeijingTimeZone)
	start := time.Date(
		nowBJT.Year(), nowBJT.Month(), nowBJT.Day(),
		0, 0, 0, 0, utils.BeijingTimeZone,
	).AddDate(0, 0, -7)

	weeklyDir := fmt.Sprintf("news/weekly/weekly-%s", start.Format("2006-01-02"))

	return SummaryContext{
		NowBJT:          nowBJT,
		StartTime:       start,
		WeeklyDir:       weeklyDir,
		ReportPath:      weeklyDir + "/index.md",
		WeeklyIndexPath: "news/weekly/index.md",
	}
}

// collectWeeklyData 遍历所有公开仓库，拉取时间窗口内的 commit，
// 过滤 bot 提交，并尝试获取课程名称，返回聚合结果。
func collectWeeklyData(ctx SummaryContext, orgName string, publicRepos map[string]struct{}) WeeklyAggregate {
	var (
		mu sync.Mutex
		wg sync.WaitGroup
		// 所有协程可见、共享的信息，需要保护
		commits   = make([]CommitEntry, 0)  // 过滤 bot 后的 commit 列表，无序
		repoNames = make(map[string]string) // repo 名 -> 课程名的映射
		goLimit   = make(chan struct{}, summaryGoroutineLimit)
	)

	for repo := range publicRepos {
		wg.Add(1)
		go func(repo string) {
			defer wg.Done()

			goLimit <- struct{}{}        // 获取限流令牌
			defer func() { <-goLimit }() // 释放令牌

			repoCommits, err := github.ListCommitsSince(orgName, repo, ctx.StartTime.Format(time.RFC3339))
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
				date = date.In(utils.BeijingTimeZone)
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
			} // 仅当存在有效提交时，尝试获取课程名称，减少不必要的 API 调用

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
	return WeeklyAggregate{
		Commits:  commits,
		RepoName: repoNames,
	}
}

// generateSummarySection 调用 OpenAI 生成周报摘要段。
// 如果调用失败或返回 __NO_SUMMARY__，则返回空字符串（不插入摘要）。
func generateSummarySection(markdownReport string) string {
	summaryText, err := openai.GenerateWeeklySummary(markdownReport)
	if err != nil {
		log.Printf("AI summary generation failed: %v, using original report instead.", err)
		return ""
	}

	if summaryText == "__NO_SUMMARY__" {
		return ""
	}

	// 对 AI 生成的摘要进行清理，移除控制字符和多余空白，确保在 Markdown 中安全显示。
	lines := strings.Split(summaryText, "\n")
	for i, line := range lines {
		lines[i] = utils.SanitizeLinkLabel(line)
	}

	return strings.Join(lines, "\n")
}

// WriteWeeklyIndex 更新周报索引文件的标题、日期、描述。
func WriteWeeklyIndex(path string, now time.Time) error {
	fm := struct {
		Title       string `yaml:"title"`
		Date        string `yaml:"date"`
		Description string `yaml:"description"`
	}{
		Title:       "AUTO 周报",
		Date:        now.Format("2006-01-02"),
		Description: fmt.Sprintf("AUTO 周报是由 ChatGPT 每周五发布的一份简报，最近更新于 %s。", now.Format("2006-01-02")),
	}
	out, err := yaml.Marshal(&fm)
	if err != nil {
		return err
	}
	content := fmt.Sprintf("---\n%s---\n", string(out))
	return os.WriteFile(path, []byte(content), 0o644)
}

// BuildMarkdown 将 commit 列表按日期降序渲染为 markdown 格式的「更新内容」段落。
func BuildMarkdown(commits []CommitEntry, repoTitles map[string]string, orgName string) string {
	if len(commits) == 0 {
		return ""
	}
	// 按日期降序排序，最新的 commit 在前面
	sort.Slice(commits, func(i, j int) bool {
		return commits[i].Date.After(commits[j].Date)
	})

	var b strings.Builder
	b.WriteString("## 更新内容\n\n")

	var prevDate string // 用于检测日期变化，按天分组显示
	for _, commit := range commits {
		dateStr := commit.Date.Format("2006-01-02")
		if dateStr != prevDate { // 说明是新的一天了，插入新日期标题
			fmt.Fprintf(&b, "### %s (%d.%d)\n\n", utils.ChineseWeekday(commit.Date), commit.Date.Month(), commit.Date.Day())
			prevDate = dateStr
		}
		title := repoTitles[commit.RepoName]
		if title == "" {
			title = commit.RepoName
		}
		title = utils.SanitizeInlineText(title)
		author := utils.SanitizeInlineText(commit.AuthorName)
		message := utils.SanitizeInlineText(strings.Split(commit.Message, "\n")[0]) // commit message 可能有多行补充信息，只取第一行作为摘要
		fmt.Fprintf(&b, "- %s 在 [%s](https://github.com/%s/%s) 中提交了信息：%s\n\n", author, title, orgName, commit.RepoName, message)
	}
	return b.String()
}

// GenerateWeeklyFrontMatter 生成周报的 YAML front matter。
func GenerateWeeklyFrontMatter(startDate time.Time, now time.Time) (string, error) {
	return utils.GenerateFrontMatter(
		fmt.Sprintf("AUTO 周报 %s - %s", startDate.Format("2006-01-02"), now.Format("2006-01-02")),
		now.Format("2006-01-02"),
		fmt.Sprintf("涵盖 %s 至 %s 的更新", startDate.Format("2006-01-02"), now.Format("2006-01-02")),
		[]utils.Author{{
			Name:  "ChatGPT",
			Link:  "https://github.com/openai",
			Image: "https://github.com/openai.png",
		}})
}

// fetchCourseName 从仓库的 tag 文件中提取课程名称。
func fetchCourseName(orgName, repoName string) (string, error) {
	text, err := github.GetRawTag(orgName, repoName)
	if err != nil {
		return "", err
	}
	_, after, found := strings.Cut(text, "name:")
	if !found {
		return "", fmt.Errorf("name not found")
	}
	return strings.TrimSpace(after), nil
}
