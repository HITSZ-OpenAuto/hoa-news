// 周报总结
package report

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/HITSZ-OpenAuto/hoa-news/internal/github"
	"github.com/HITSZ-OpenAuto/hoa-news/internal/openai"
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

type SummaryContext struct {
	NowUTC           time.Time
	NowBJT           time.Time
	StartTime        time.Time
	DisplayStartTime time.Time
	WeeklyDir        string
	ReportPath       string
	WeeklyIndexPath  string
}

type WeeklyAggregate struct {
	Commits    []CommitEntry
	RepoTitles map[string]string
}

func Summary(orgName string, publicRepos map[string]struct{}) {

	ctx := buildSummaryContext(time.Now().UTC())
	agg := collectWeeklyData(ctx, orgName, publicRepos)

	if len(agg.Commits) == 0 {
		log.Println("No commits found in the given period of time")
		return
	}

	markdownReport := BuildMarkdown(agg.Commits, agg.RepoTitles, orgName)
	frontMatter, err := GenerateWeeklyFrontMatter(ctx.DisplayStartTime, ctx.NowBJT)
	if err != nil {
		log.Fatalf("Failed to generate front matter: %v", err)
	}

	summarySection := generateSummarySection(markdownReport)
	finalReport := assembleWeeklyReport(frontMatter, summarySection, markdownReport)

	if err := writeWeeklyArtifacts(ctx, finalReport); err != nil {
		log.Fatalf("Failed to write report: %v", err)
	}

	if err := WriteWeeklyIndex(ctx.WeeklyIndexPath, ctx.NowBJT); err != nil {
		log.Printf("Failed to update weekly index: %v", err)
	}
}

func buildSummaryContext(nowUTC time.Time) SummaryContext {
	startTime, displayStartTime := calculateStartTimeAt(nowUTC)
	nowBJT := nowUTC.Add(8 * time.Hour)
	weeklyDir := fmt.Sprintf("news/weekly/weekly-%s", displayStartTime.Format("2006-01-02"))

	return SummaryContext{
		NowUTC:           nowUTC,
		NowBJT:           nowBJT,
		StartTime:        startTime,
		DisplayStartTime: displayStartTime,
		WeeklyDir:        weeklyDir,
		ReportPath:       weeklyDir + "/index.mdx",
		WeeklyIndexPath:  "news/weekly/index.md",
	}
}

func collectWeeklyData(ctx SummaryContext, orgName string, publicRepos map[string]struct{}) WeeklyAggregate {
	commits := make([]CommitEntry, 0)
	repoTitles := make(map[string]string)

	for repo := range publicRepos {
		repoCommits, err := github.ListCommitsSince(orgName, repo, ctx.StartTime.Format(time.RFC3339))
		if err != nil {
			log.Printf("Failed to fetch commits for %s: %v", repo, err)
			continue
		}

		containsManual := false
		for _, commit := range repoCommits {
			authorName := commit.Commit.Author.Name
			authorLogin := ""
			if commit.Author != nil {
				authorLogin = commit.Author.Login
			}
			if utils.IsBot(authorName, authorLogin) {
				continue
			}
			containsManual = true

			date, err := time.Parse(time.RFC3339, commit.Commit.Author.Date)
			if err != nil {
				continue
			}
			date = date.Add(8 * time.Hour)

			commits = append(commits, CommitEntry{
				AuthorName:  authorName,
				AuthorLogin: authorLogin,
				Date:        date,
				Message:     commit.Commit.Message,
				RepoName:    repo,
			})
		}

		if containsManual {
			if name, err := fetchCourseName(orgName, repo); err == nil && name != "" {
				repoTitles[repo] = name
			}
		}
	}

	return WeeklyAggregate{
		Commits:    commits,
		RepoTitles: repoTitles,
	}
}

func generateSummarySection(markdownReport string) string {
	summaryText, err := openai.GenerateWeeklySummary(markdownReport)
	if err != nil {
		log.Printf("Summary generation failed: %v, using full report instead.", err)
		return ""
	}

	if summaryText == "__NO_SUMMARY__" {
		return ""
	}

	return summaryText
}

func assembleWeeklyReport(frontMatter, summarySection, markdownReport string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "---\n%s---\n\n", frontMatter)
	if summarySection != "" {
		b.WriteString(summarySection)
		b.WriteString("\n\n")
	}
	b.WriteString(markdownReport)
	return b.String()
}

func writeWeeklyArtifacts(ctx SummaryContext, finalReport string) error {
	if err := os.MkdirAll(ctx.WeeklyDir, 0o755); err != nil {
		return fmt.Errorf("failed to create weekly directory: %w", err)
	}
	if err := utils.WriteReport(ctx.ReportPath, finalReport); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}
	return nil
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
			fmt.Fprintf(&b, "### %s (%d.%d)\n\n", utils.ChineseWeekday(commit.Date), commit.Date.Month(), commit.Date.Day())
			prevDate = dateStr
		}
		title := repoTitles[commit.RepoName]
		if title == "" {
			title = commit.RepoName
		}
		message := strings.Split(commit.Message, "\n")[0]
		fmt.Fprintf(&b, "- %s 在 [%s](https://github.com/%s/%s) 中提交了信息：%s\n\n", commit.AuthorName, title, orgName, commit.RepoName, message)
	}
	return b.String()
}

func GenerateWeeklyFrontMatter(startDate time.Time, now time.Time) (string, error) {
	title := fmt.Sprintf("AUTO 周报 %s - %s", startDate.Format("2006-01-02"), now.Format("2006-01-02"))
	description := fmt.Sprintf("涵盖 %s 至 %s 的更新", startDate.Format("2006-01-02"), now.Format("2006-01-02"))
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

func calculateStartTimeAt(nowUTC time.Time) (time.Time, time.Time) {
	delta := 7 * 24 * time.Hour
	start := nowUTC.Add(-delta)
	startOfDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	return startOfDay, startOfDay
}

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
