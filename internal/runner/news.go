package runner

import (
	"fmt"
	"os"

	"github.com/HITSZ-OpenAuto/hoa-news/internal/github"
	"github.com/HITSZ-OpenAuto/hoa-news/internal/news"
)

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

	if err := news.UpdateDailyReport("news/daily.mdx", orgName, publicRepos, issues, prs); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to update daily report: %v\n", err)
		os.Exit(1)
	}
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
