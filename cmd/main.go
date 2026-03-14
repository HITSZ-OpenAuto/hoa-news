package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/HITSZ-OpenAuto/hoa-news/internal/github"
	"github.com/HITSZ-OpenAuto/hoa-news/internal/report"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <daily|weekly>\n", os.Args[0])
		os.Exit(2)
	}
	orgName := os.Getenv("ORG_NAME")
	if orgName == "" {
		fmt.Fprintln(os.Stderr, "Environment variable ORG_NAME not found, please set it first.")
		os.Exit(1)
	}
	publicRepos, err := github.LoadPublicRepos()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load public repos: %v\n", err)
		os.Exit(1)
	}
	log.Printf("Fetched %d public repos", len(publicRepos))
	PAT := os.Getenv("PERSONAL_ACCESS_TOKEN")
	if PAT == "" {
		fmt.Fprintln(os.Stderr, "Environment variable PERSONAL_ACCESS_TOKEN not found, please set it first.")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "daily":
		if err := report.Daily(orgName, publicRepos); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate daily news: %v\n", err)
			os.Exit(1)
		}

	case "weekly":
		if err := report.Weekly(orgName, publicRepos); err != nil {
			if errors.Is(err, report.ErrNoWeeklyCommits) {
				log.Printf("Summary skipped: %v", err)
				return
			}
			fmt.Fprintf(os.Stderr, "Failed to generate weekly summary: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nUsage: go run cmd/main.go <daily|weekly>\n", os.Args[1])
		os.Exit(2)
	}
}
