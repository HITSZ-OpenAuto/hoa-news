package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const timeZoneOffset = 8 * time.Hour

// GitHub CLI JSON structures

type ghLabel struct {
	Name string `json:"name"`
}

type ghAuthor struct {
	Login string `json:"login"`
}

type ghRepository struct {
	Name string `json:"name"`
}

type ghItem struct {
	Title      string       `json:"title"`
	URL        string       `json:"url"`
	Repository ghRepository `json:"repository"`
	CreatedAt  string       `json:"createdAt"`
	Author     ghAuthor     `json:"author"`
	Labels     []ghLabel    `json:"labels"`
}

func runGHCommand(args []string) ([]byte, error) {
	cmd := exec.Command("gh", args...)
	cmd.Env = os.Environ()
	if token := os.Getenv("PERSONAL_ACCESS_TOKEN"); token != "" {
		cmd.Env = append(cmd.Env, "GH_TOKEN="+token)
	}
	return cmd.Output()
}

func getOrgIssues(orgName string) ([]ghItem, error) {
	args := []string{
		"search",
		"issues",
		"--json",
		"title,url,repository,createdAt,author,labels",
		"--state",
		"open",
		"--owner",
		orgName,
		"--limit",
		"1000",
	}
	output, err := runGHCommand(args)
	if err != nil {
		return nil, err
	}
	var items []ghItem
	if err := json.Unmarshal(output, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func getOrgPullRequests(orgName string) ([]ghItem, error) {
	args := []string{
		"search",
		"prs",
		"--json",
		"title,url,repository,createdAt,author,labels",
		"--state",
		"open",
		"--owner",
		orgName,
		"--limit",
		"1000",
	}
	output, err := runGHCommand(args)
	if err != nil {
		return nil, err
	}
	var items []ghItem
	if err := json.Unmarshal(output, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func utcToBJT(utcTimeStr string) string {
	utcTime, err := time.Parse(time.RFC3339, utcTimeStr)
	if err != nil {
		return utcTimeStr
	}
	bjt := utcTime.Add(timeZoneOffset)
	return bjt.Format("2006-01-02 15:04:05")
}

func filterByPublicRepos(items []ghItem, publicRepos map[string]struct{}) []ghItem {
	if len(publicRepos) == 0 {
		return items
	}
	filtered := make([]ghItem, 0, len(items))
	for _, item := range items {
		if _, ok := publicRepos[item.Repository.Name]; ok {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func loadPublicRepos() (map[string]struct{}, error) {
	reposJSON := os.Getenv("repos_array")
	if reposJSON == "" {
		return map[string]struct{}{}, nil
	}
	var repos []string
	if err := json.Unmarshal([]byte(reposJSON), &repos); err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(repos))
	for _, name := range repos {
		set[name] = struct{}{}
	}
	return set, nil
}

func updateDailyReport(issues []ghItem, prs []ghItem) error {
	path := "news/daily.mdx"
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

func main() {
	orgName := os.Getenv("ORG_NAME")
	if orgName == "" {
		fmt.Fprintln(os.Stderr, "Environment variable ORG_NAME not found, please set it first.")
		os.Exit(1)
	}

	publicRepos, err := loadPublicRepos()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse repos_array: %v\n", err)
		os.Exit(1)
	}

	issues, err := getOrgIssues(orgName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get issues: %v\n", err)
		os.Exit(1)
	}
	prs, err := getOrgPullRequests(orgName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get pull requests: %v\n", err)
		os.Exit(1)
	}

	issues = filterByPublicRepos(issues, publicRepos)
	prs = filterByPublicRepos(prs, publicRepos)

	if err := updateDailyReport(issues, prs); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to update daily report: %v\n", err)
		os.Exit(1)
	}
}
