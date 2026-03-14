package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/HITSZ-OpenAuto/hoa-news/internal/config"
	"github.com/HITSZ-OpenAuto/hoa-news/internal/github"
	"github.com/HITSZ-OpenAuto/hoa-news/internal/report"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <daily|weekly>\n", os.Args[0])
		os.Exit(2)
	}
	publicRepos, err := github.LoadPublicRepos()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load public repos: %v\n", err)
		os.Exit(1)
	}
	log.Printf("Fetched %d public repos", len(publicRepos))

	switch os.Args[1] {
	case "daily":
		if err := report.Daily(config.OrgName, publicRepos); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate daily news: %v\n", err)
			os.Exit(1)
		}

	case "weekly":
		if err := report.Weekly(config.OrgName, publicRepos); err != nil {
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
