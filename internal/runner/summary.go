package runner

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HITSZ-OpenAuto/hoa-news/internal/github"
	"github.com/HITSZ-OpenAuto/hoa-news/internal/openai"
	"github.com/HITSZ-OpenAuto/hoa-news/internal/summary"
	"github.com/HITSZ-OpenAuto/hoa-news/internal/utils"
)

func Summary(newsType, orgName string, publicRepos map[string]struct{}) {
	startTime, displayStartTime := calculateStartTime(newsType)

	commits := make([]summary.CommitEntry, 0)
	repoTitles := make(map[string]string)
	for repo := range publicRepos {
		repoCommits, err := github.ListCommitsSince(orgName, repo, startTime.Format(time.RFC3339))
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
			if !utils.IsBot(authorName, authorLogin) {
				containsManual = true
			}
			date, err := time.Parse(time.RFC3339, commit.Commit.Author.Date)
			if err != nil {
				continue
			}
			date = date.Add(8 * time.Hour)
			commits = append(commits, summary.CommitEntry{
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

	if len(commits) == 0 {
		log.Println("No commits found in the given period of time")
		return
	}

	markdownReport := summary.BuildMarkdown(commits, repoTitles, orgName)

	var frontMatter string
	if newsType == "weekly" {
		nowBJT := time.Now().UTC().Add(8 * time.Hour)
		fm, err := summary.GenerateWeeklyFrontMatter(displayStartTime, nowBJT)
		if err != nil {
			log.Fatalf("Failed to generate front matter: %v", err)
		}
		frontMatter = fm
	} else {
		log.Println("Daily news generation via summary runner is deprecated. Use 'news' command instead.")
		return
	}

	finalReport := fmt.Sprintf("---\n%s---\n\n", frontMatter)

	if newsType == "weekly" {
		if summaryText, err := openai.GenerateWeeklySummary(markdownReport); err == nil {
			if summaryText != "__NO_SUMMARY__" {
				finalReport += summaryText + "\n\n"
			}
		} else {
			log.Printf("Summary generation failed: %v, using full report instead.", err)
		}
		finalReport += markdownReport

		weeklyDir := filepath.Join("news", "weekly", fmt.Sprintf("weekly-%s", displayStartTime.Format("2006-01-02")))
		if err := os.MkdirAll(weeklyDir, 0o755); err != nil {
			log.Fatalf("Failed to create weekly directory: %v", err)
		}
		reportPath := filepath.Join(weeklyDir, "index.mdx")
		if err := utils.WriteReport(reportPath, finalReport); err != nil {
			log.Fatalf("Failed to write report: %v", err)
		}
		if err := summary.WriteWeeklyIndex(filepath.Join("news", "weekly", "_index.zh-cn.md"), time.Now().UTC().Add(8*time.Hour)); err != nil {
			log.Printf("Failed to update weekly index: %v", err)
		}
	} else {
		finalReport += markdownReport
		if err := utils.WriteReport(filepath.Join("news", "daily.mdx"), finalReport); err != nil {
			log.Fatalf("Failed to write report: %v", err)
		}
	}
}

func calculateStartTime(newsType string) (time.Time, time.Time) {
	var delta time.Duration
	if newsType == "weekly" {
		delta = 7 * 24 * time.Hour
	} else {
		delta = 24 * time.Hour
	}
	start := time.Now().UTC().Add(-delta)
	startOfDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	return startOfDay, start
}

func fetchCourseName(orgName, repoName string) (string, error) {
	text, err := github.GetRawTag(orgName, repoName)
	if err != nil {
		return "", err
	}
	idx := strings.Index(text, "name:")
	if idx < 0 {
		return "", fmt.Errorf("name not found")
	}
	name := strings.TrimSpace(text[idx+len("name:"):])
	return name, nil
}
