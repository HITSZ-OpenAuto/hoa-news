package github

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Label struct {
	Name string `json:"name"`
}

type Author struct {
	Login string `json:"login"`
}

type Repository struct {
	Name string `json:"name"`
}

type Item struct {
	Title      string     `json:"title"`
	URL        string     `json:"url"`
	Repository Repository `json:"repository"`
	CreatedAt  string     `json:"createdAt"`
	Author     Author     `json:"author"`
	Labels     []Label    `json:"labels"`
}

type Repo struct {
	Name    string `json:"name"`
	Private bool   `json:"private"`
}

type Commit struct {
	Commit struct {
		Author struct {
			Name string `json:"name"`
			Date string `json:"date"`
		} `json:"author"`
		Message string `json:"message"`
	} `json:"commit"`
	Author *struct {
		Login string `json:"login"`
	} `json:"author"`
}

func ghCommand(args []string) ([]byte, error) {
	cmd := exec.Command("gh", args...)
	cmd.Env = os.Environ()
	if PAT := os.Getenv("PERSONAL_ACCESS_TOKEN"); PAT != "" {
		cmd.Env = append(cmd.Env, "GH_TOKEN="+PAT)
	} // 事实上 PAT 环境变量会在 main 中被检查是否存在，这里只是双保险

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, errors.New(stderr.String())
	}
	return output, nil
}

func SearchIssues(orgName string, limit int) ([]Item, error) {
	args := []string{
		"search", "issues",
		"--json", "title,url,repository,createdAt,author,labels",
		"--state", "open",
		"--owner", orgName,
		"--limit", fmt.Sprintf("%d", limit),
	}
	output, err := ghCommand(args)
	if err != nil {
		return nil, err
	}
	var items []Item
	if err := json.Unmarshal(output, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func SearchPullRequests(orgName string, limit int) ([]Item, error) {
	args := []string{
		"search", "prs",
		"--json", "title,url,repository,createdAt,author,labels",
		"--state", "open",
		"--owner", orgName,
		"--limit", fmt.Sprintf("%d", limit),
	}
	output, err := ghCommand(args)
	if err != nil {
		return nil, err
	}
	var items []Item
	if err := json.Unmarshal(output, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func ListCommitsSince(orgName, repoName, since_RFC3339 string) ([]Commit, error) {
	args := []string{
		"api",
		fmt.Sprintf("/repos/%s/%s/commits", orgName, repoName),
		"--method", "GET",
		"--paginate",
		"-f", "per_page=100",
		"-f", "since=" + since_RFC3339,
		"--jq",
		".[]",
	}
	output, err := ghCommand(args)
	if err != nil {
		return nil, err
	}
	return parseNDJSON[Commit](output)
}

func GetRawTag(orgName, repoName string) (string, error) {
	args := []string{
		"api",
		fmt.Sprintf("/repos/%s/%s/contents/tag.txt", orgName, repoName),
		"-H",
		"Accept: application/vnd.github.raw",
	}
	output, err := ghCommand(args)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func parseNDJSON[T Commit | Repo](data []byte) ([]T, error) {
	items := make([]T, 0)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var item T
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
