package main

import (
	"fmt"
	"os"

	"github.com/HITSZ-OpenAuto/hoa-news/internal/github"
	"github.com/HITSZ-OpenAuto/hoa-news/internal/report"
)

func main() {
	orgName := os.Getenv("ORG_NAME")
	if orgName == "" {
		fmt.Fprintln(os.Stderr, "Environment variable ORG_NAME not found, please set it first.")
		os.Exit(1)
	}

	publicRepos, err := report.LoadPublicRepos()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse repos_array: %v\n", err)
		os.Exit(1)
	}

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

	if err := report.UpdateDailyReport("news/daily.mdx", issues, prs); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to update daily report: %v\n", err)
		os.Exit(1)
	}
}

// 对组织里的全部 issues/PR 进行过滤，只保留属于公开仓库的
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
