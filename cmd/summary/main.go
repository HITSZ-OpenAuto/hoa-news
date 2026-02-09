package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HITSZ-OpenAuto/hoa-news/internal/githubapi"
	"github.com/HITSZ-OpenAuto/hoa-news/internal/news"
	"github.com/HITSZ-OpenAuto/hoa-news/internal/openai"
	"github.com/HITSZ-OpenAuto/hoa-news/internal/report"
)

func main() {
	token := os.Getenv("PERSONAL_ACCESS_TOKEN")
	orgName := os.Getenv("ORG_NAME")
	newsType := os.Getenv("NEWS_TYPE")
	if token == "" || orgName == "" || newsType == "" {
		log.Fatal("Missing required environment variables: PERSONAL_ACCESS_TOKEN, ORG_NAME, NEWS_TYPE")
	}

	client := githubapi.NewClient(token)
	startTime, displayStartTime := calculateStartTime(newsType)

	repos, err := report.LoadPublicRepos()
	if err != nil {
		log.Fatalf("Failed to parse repos_array: %v", err)
	}

	commits := make([]news.CommitEntry, 0)
	repoTitles := make(map[string]string)
	for repo := range repos {
		if repo == "hoa-moe" || repo == ".github" {
			continue
		}
		repoCommits, err := listCommitsSince(client, orgName, repo, startTime)
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
			if !news.IsBot(authorName, authorLogin) {
				containsManual = true
			}
			date, err := time.Parse(time.RFC3339, commit.Commit.Author.Date)
			if err != nil {
				continue
			}
			date = date.Add(8 * time.Hour)
			commits = append(commits, news.CommitEntry{
				AuthorName:  authorName,
				AuthorLogin: authorLogin,
				Date:        date,
				Message:     commit.Commit.Message,
				RepoName:    repo,
			})
		}
		if containsManual {
			if name, err := fetchCourseName(client, orgName, repo); err == nil && name != "" {
				repoTitles[repo] = name
			}
		}
	}

	if len(commits) == 0 {
		log.Println("No commits found in the given period of time")
		return
	}

	markdownReport := news.BuildMarkdown(commits, repoTitles, orgName)

	var frontMatter string
	if newsType == "weekly" {
		nowBJT := time.Now().UTC().Add(8 * time.Hour)
		fm, err := news.GenerateWeeklyFrontMatter(displayStartTime, nowBJT)
		if err != nil {
			log.Fatalf("Failed to generate front matter: %v", err)
		}
		frontMatter = fm
	} else {
		fm, err := news.GenerateDailyFrontMatter()
		if err != nil {
			log.Fatalf("Failed to generate front matter: %v", err)
		}
		frontMatter = fm
	}

	finalReport := fmt.Sprintf("---\n%s---\n\n", frontMatter)

	if newsType == "weekly" {
		if summary, err := openai.GenerateWeeklySummary(markdownReport); err == nil {
			if summary != "__NO_SUMMARY__" {
				finalReport += summary + "\n\n"
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
		if err := news.WriteReport(reportPath, finalReport); err != nil {
			log.Fatalf("Failed to write report: %v", err)
		}
		if err := news.WriteWeeklyIndex(filepath.Join("news", "weekly", "_index.zh-cn.md"), time.Now().UTC().Add(8*time.Hour)); err != nil {
			log.Printf("Failed to update weekly index: %v", err)
		}
	} else {
		finalReport += markdownReport
		if err := news.WriteReport(filepath.Join("news", "daily.mdx"), finalReport); err != nil {
			log.Fatalf("Failed to write report: %v", err)
		}
	}
}

func httpNewRequest(url string) (*http.Request, error) {
	return http.NewRequest(http.MethodGet, url, nil)
}

func readAll(resp *http.Response) ([]byte, error) {
	return io.ReadAll(resp.Body)
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

func listCommitsSince(client *githubapi.Client, orgName, repoName string, since time.Time) ([]githubapi.Commit, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/commits", orgName, repoName)
	params := url.Values{}
	params.Set("since", since.Format(time.RFC3339))
	params.Set("per_page", "100")

	commits := make([]githubapi.Commit, 0)
	nextURL := client.BaseURL + endpoint + "?" + params.Encode()
	for nextURL != "" {
		var page []githubapi.Commit
		link, err := client.GetJSON(nextURL, nil, &page)
		if err != nil {
			return nil, err
		}
		commits = append(commits, page...)
		nextURL = githubapi.ParseNextLink(link)
	}
	return commits, nil
}

func fetchCourseName(client *githubapi.Client, orgName, repoName string) (string, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main/tag.txt", orgName, repoName)
	req, err := httpNewRequest(url)
	if err != nil {
		return "", err
	}
	resp, err := client.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("tag not found")
	}
	body, err := readAll(resp)
	if err != nil {
		return "", err
	}
	text := string(body)
	idx := strings.Index(text, "name:")
	if idx < 0 {
		return "", fmt.Errorf("name not found")
	}
	name := strings.TrimSpace(text[idx+len("name:"):])
	return name, nil
}
