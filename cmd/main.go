package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/HITSZ-OpenAuto/hoa-news/internal/runner"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <news|summary>\n", os.Args[0])
		os.Exit(2)
	}
	orgName := os.Getenv("ORG_NAME")
	if orgName == "" {
		fmt.Fprintln(os.Stderr, "Environment variable ORG_NAME not found, please set it first.")
		os.Exit(1)
	}
	publicRepos, err := LoadPublicRepos()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse repos_array: %v\n", err)
		os.Exit(1)
	}
	PAT := os.Getenv("PERSONAL_ACCESS_TOKEN")
	if PAT == "" {
		fmt.Fprintln(os.Stderr, "Environment variable PERSONAL_ACCESS_TOKEN not found, please set it first.")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "news":
		runner.News(orgName, publicRepos)

	case "summary":
		newsType := os.Getenv("NEWS_TYPE")
		if newsType == "" {
			fmt.Fprintln(os.Stderr, "Environment variable NEWS_TYPE not found, please set it first.")
			os.Exit(1)
		}
		runner.Summary(newsType, orgName, publicRepos)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nUsage: %s <news|summary>\n", os.Args[1], os.Args[0])
		os.Exit(2)
	}
}

// 从环境变量接收公开仓库列表，返回字典方便查询
func LoadPublicRepos() (map[string]struct{}, error) {
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
