package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/HITSZ-OpenAuto/hoa-news/internal/report"
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
		if err := report.News(orgName, publicRepos); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate daily news: %v\n", err)
			os.Exit(1)
		}

	case "summary":
		if err := report.Summary(orgName, publicRepos); err != nil {
			if errors.Is(err, report.ErrNoWeeklyCommits) {
				log.Printf("Summary skipped: %v", err)
				return
			}
			fmt.Fprintf(os.Stderr, "Failed to generate weekly summary: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nUsage: go run cmd/main.go <news|summary>\n", os.Args[1])
		os.Exit(2)
	}
}

// 从环境变量接收公开仓库列表，返回字典方便查询
func LoadPublicRepos() (map[string]struct{}, error) {
	reposJSON := os.Getenv("repos_array")
	if reposJSON == "" {
		return map[string]struct{}{}, errors.New("Environment variable repos_array not found, please set it first.")
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
