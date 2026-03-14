package report

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/HITSZ-OpenAuto/hoa-news/internal/github"
	"github.com/HITSZ-OpenAuto/hoa-news/internal/utils"
)

func TestGenerateDailyFrontMatter(t *testing.T) {
	result, err := utils.GenerateFrontMatter(
		"AUTO 更新速递",
		time.Now().UTC().Format("2006-01-02"),
		"每日更新",
		[]utils.Author{{
			Name:  "github-actions[bot]",
			Link:  "https://github.com/features/actions",
			Image: "https://avatars.githubusercontent.com/in/15368",
		}},
	)
	if err != nil {
		t.Fatalf("GenerateFrontMatter() returned error: %v", err)
	}

	// Check required fields
	requiredFields := []string{
		"title:",
		"AUTO 更新速递",
		"date:",
		"authors:",
		"github-actions[bot]",
		"description:",
		"每日更新",
		"excludeSearch:",
		"draft:",
	}

	for _, field := range requiredFields {
		if !strings.Contains(result, field) {
			t.Errorf("GenerateDailyFrontMatter() output missing %q", field)
		}
	}

	// Verify it contains today's date
	today := time.Now().UTC().Format("2006-01-02")
	if !strings.Contains(result, today) {
		t.Errorf("GenerateDailyFrontMatter() output missing today's date %q", today)
	}
}

func TestUpdateDailyReport_EmptyData(t *testing.T) {
	// Test with empty issues and PRs
	tmpFile := t.TempDir() + "/daily.md"
	orgName := "test-org"
	publicRepos := make(map[string]struct{})
	var issues []github.Item
	var prs []github.Item

	err := UpdateDailyReport(tmpFile, orgName, publicRepos, issues, prs)
	if err != nil {
		t.Fatalf("UpdateDailyReport() returned error: %v", err)
	}
}

func TestUpdateDailyReport_WithData(t *testing.T) {
	// Test with sample issues and PRs
	tmpFile := t.TempDir() + "/daily.md"
	orgName := "test-org"
	publicRepos := make(map[string]struct{})

	issues := []github.Item{
		{
			Title: "增加 25 秋考试信息 (#39)",
			URL:   "https://github.com/test-org/repo/issues/1",
			Repository: github.Repository{
				Name: "test-repo",
			},
			Author: github.Author{
				Login: "testuser",
			},
			CreatedAt: "2026-02-13T10:00:00Z",
			Labels: []github.Label{
				{Name: "bug"},
				{Name: "enhancement"},
			},
		},
	}

	prs := []github.Item{
		{
			Title: "Fix [Parser] (#10)",
			URL:   "https://evil.com/steal",
			Repository: github.Repository{
				Name: "test-repo",
			},
			Author: github.Author{
				Login: "contributor",
			},
			CreatedAt: "2026-02-13T11:00:00Z",
			Labels: []github.Label{
				{Name: "feature"},
			},
		},
	}

	err := UpdateDailyReport(tmpFile, orgName, publicRepos, issues, prs)
	if err != nil {
		t.Fatalf("UpdateDailyReport() returned error: %v", err)
	}

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	got := string(content)

	if !strings.Contains(got, "### [增加 25 秋考试信息 \\(#39\\)](https://github.com/test-org/repo/issues/1)") {
		t.Errorf("expected issue link, got:\n%s", got)
	}

	if !strings.Contains(got, "### [Fix \\[Parser\\] \\(#10\\)](https://evil.com/steal)") {
		t.Errorf("expected PR link to keep original URL, got:\n%s", got)
	}
}

func TestUpdateDailyReport_EscapeMDXPayloadInTitles(t *testing.T) {
	tmpFile := t.TempDir() + "/daily.md"
	orgName := "test-org"
	publicRepos := make(map[string]struct{})

	issues := []github.Item{
		{
			Title: "<script>alert(1)</script>",
			URL:   "https://github.com/test-org/repo/issues/2",
			Repository: github.Repository{
				Name: "test-repo",
			},
			Author: github.Author{
				Login: "attacker",
			},
			CreatedAt: "2026-02-13T10:00:00Z",
		},
	}

	prs := []github.Item{
		{
			Title: "{alert(1)}",
			URL:   "https://github.com/test-org/repo/pull/2",
			Repository: github.Repository{
				Name: "test-repo",
			},
			Author: github.Author{
				Login: "attacker",
			},
			CreatedAt: "2026-02-13T11:00:00Z",
		},
	}

	if err := UpdateDailyReport(tmpFile, orgName, publicRepos, issues, prs); err != nil {
		t.Fatalf("UpdateDailyReport() returned error: %v", err)
	}

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	got := string(content)

	if strings.Contains(got, "<script>alert(1)</script>") {
		t.Fatalf("expected issue title to be escaped, got:\n%s", got)
	}
	if strings.Contains(got, "{alert(1)}") {
		t.Fatalf("expected pr title to be escaped, got:\n%s", got)
	}

	if !strings.Contains(got, "&lt;script&gt;alert\\(1\\)&lt;/script&gt;") {
		t.Fatalf("escaped issue title not found, got:\n%s", got)
	}
	if !strings.Contains(got, "&#123;alert\\(1\\)&#125;") {
		t.Fatalf("escaped pr title not found, got:\n%s", got)
	}
}

func TestExtractBody(t *testing.T) {
	t.Run("WithFrontMatter", func(t *testing.T) {
		in := "---\ntitle: t\n---\n\nhello\nworld\n"
		want := "\nhello\nworld\n"
		if got := extractBody(in); got != want {
			t.Fatalf("extractBody() = %q, want %q", got, want)
		}
	})

	t.Run("WithoutFrontMatter", func(t *testing.T) {
		in := "hello\r\nworld\r\n"
		want := "hello\nworld\n"
		if got := extractBody(in); got != want {
			t.Fatalf("extractBody() = %q, want %q", got, want)
		}
	})

	t.Run("MalformedFrontMatter", func(t *testing.T) {
		in := "---\ntitle: t\nno-close\n"
		want := "---\ntitle: t\nno-close\n"
		if got := extractBody(in); got != want {
			t.Fatalf("extractBody() = %q, want %q", got, want)
		}
	})
}

func TestNormalizeBody(t *testing.T) {
	a := "line1  \r\nline2\t\r\n\r\n"
	b := "line1\nline2\n"
	if normalizeBody(a) != normalizeBody(b) {
		t.Fatalf("normalizeBody should treat trailing spaces and CRLF as equivalent")
	}
}

func TestUpdateDailyReport_UnchangedBodySkipsRewrite(t *testing.T) {
	tmpFile := t.TempDir() + "/daily.md"
	orgName := "test-org"
	publicRepos := make(map[string]struct{})
	var issues []github.Item
	var prs []github.Item

	existingBody := buildDailyBody(orgName, nil, map[string]string{}, nil, nil)
	// The old file intentionally uses CRLF + trailing spaces; normalizeBody should still match.
	existingBody = strings.ReplaceAll(existingBody, "\n", "  \r\n")
	existing := "---\n" +
		"title: AUTO 更新速递\n" +
		"date: \"1999-01-01\"\n" +
		"---\n\n" +
		existingBody
	if err := os.WriteFile(tmpFile, []byte(existing), 0o644); err != nil {
		t.Fatalf("failed to seed existing daily report: %v", err)
	}

	if err := UpdateDailyReport(tmpFile, orgName, publicRepos, issues, prs); err != nil {
		t.Fatalf("UpdateDailyReport() returned error: %v", err)
	}

	after, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if string(after) != existing {
		t.Fatalf("expected unchanged file when body is equivalent")
	}
	if !strings.Contains(string(after), "date: \"1999-01-01\"") {
		t.Fatalf("date should not update when body is unchanged")
	}
}

func TestUpdateDailyReport_SortsIssuesAndPRsDeterministically(t *testing.T) {
	tmpFile := t.TempDir() + "/daily.md"
	orgName := "test-org"
	publicRepos := make(map[string]struct{})

	issues := []github.Item{
		{
			Title:     "older",
			URL:       "https://example.com/issues/3",
			CreatedAt: "2026-02-12T10:00:00Z",
			Repository: github.Repository{
				Name: "repo-z",
			},
			Author: github.Author{Login: "u1"},
		},
		{
			Title:     "same-time-b",
			URL:       "https://example.com/issues/2",
			CreatedAt: "2026-02-13T10:00:00Z",
			Repository: github.Repository{
				Name: "repo-b",
			},
			Author: github.Author{Login: "u2"},
		},
		{
			Title:     "same-time-a",
			URL:       "https://example.com/issues/1",
			CreatedAt: "2026-02-13T10:00:00Z",
			Repository: github.Repository{
				Name: "repo-a",
			},
			Author: github.Author{Login: "u3"},
		},
	}

	prs := []github.Item{
		{
			Title:     "bad-time",
			URL:       "https://example.com/pr/3",
			CreatedAt: "not-a-time",
			Repository: github.Repository{
				Name: "repo-a",
			},
			Author: github.Author{Login: "u1"},
		},
		{
			Title:     "newest",
			URL:       "https://example.com/pr/2",
			CreatedAt: "2026-02-14T10:00:00Z",
			Repository: github.Repository{
				Name: "repo-z",
			},
			Author: github.Author{Login: "u2"},
		},
		{
			Title:     "older",
			URL:       "https://example.com/pr/1",
			CreatedAt: "2026-02-13T10:00:00Z",
			Repository: github.Repository{
				Name: "repo-b",
			},
			Author: github.Author{Login: "u3"},
		},
	}

	if err := UpdateDailyReport(tmpFile, orgName, publicRepos, issues, prs); err != nil {
		t.Fatalf("UpdateDailyReport() returned error: %v", err)
	}

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	got := string(content)

	issueA := strings.Index(got, "### [same-time-a](https://example.com/issues/1)")
	issueB := strings.Index(got, "### [same-time-b](https://example.com/issues/2)")
	issueOlder := strings.Index(got, "### [older](https://example.com/issues/3)")
	if !(issueA >= 0 && issueB >= 0 && issueOlder >= 0) {
		t.Fatalf("issue entries missing, got:\n%s", got)
	}
	if !(issueA < issueB && issueB < issueOlder) {
		t.Fatalf("issues are not sorted deterministically by createdAt/repo/title")
	}

	prNewest := strings.Index(got, "### [newest](https://example.com/pr/2)")
	prOlder := strings.Index(got, "### [older](https://example.com/pr/1)")
	prBadTime := strings.Index(got, "### [bad-time](https://example.com/pr/3)")
	if !(prNewest >= 0 && prOlder >= 0 && prBadTime >= 0) {
		t.Fatalf("pr entries missing, got:\n%s", got)
	}
	if !(prNewest < prOlder && prOlder < prBadTime) {
		t.Fatalf("prs are not sorted deterministically with invalid timestamps last")
	}
}

func TestIsSubstantivelyEqual(t *testing.T) {
	t.Run("DateOnlyChange", func(t *testing.T) {
		oldContent := "---\n" +
			"title: AUTO 更新速递\n" +
			"date: \"1999-01-01\"\n" +
			"---\n\n" +
			"## 今日更新\n\n暂无更新\n\n"
		newBody := "## 今日更新\n\n暂无更新\n\n"
		if !isSubstantivelyEqual(oldContent, newBody) {
			t.Fatalf("expected date-only changes to be substantively equal")
		}
	})

	t.Run("WhitespaceOnlyChange", func(t *testing.T) {
		oldContent := "---\r\n" +
			"title: AUTO 更新速递\r\n" +
			"date: \"1999-01-01\"\r\n" +
			"---\r\n\r\n" +
			"## 今日更新  \r\n\r\n暂无更新\t\r\n"
		newBody := "## 今日更新\n\n暂无更新\n"
		if !isSubstantivelyEqual(oldContent, newBody) {
			t.Fatalf("expected newline/trailing-space differences to be substantively equal")
		}
	})

	t.Run("BodyChanged", func(t *testing.T) {
		oldContent := "---\n" +
			"title: AUTO 更新速递\n" +
			"date: \"1999-01-01\"\n" +
			"---\n\n" +
			"## 今日更新\n\nalpha\n"
		newBody := "## 今日更新\n\nbeta\n"
		if isSubstantivelyEqual(oldContent, newBody) {
			t.Fatalf("expected changed body to be not equal")
		}
	})

	t.Run("NoFrontMatter", func(t *testing.T) {
		oldContent := "## 今日更新\n\n暂无更新\n"
		newBody := "## 今日更新\n\n暂无更新\n"
		if !isSubstantivelyEqual(oldContent, newBody) {
			t.Fatalf("expected same body without front matter to be equal")
		}
	})
}

func TestExtractBody_AdditionalCases(t *testing.T) {
	t.Run("FrontMatterNoBody", func(t *testing.T) {
		in := "---\ntitle: t\n---\n"
		if got := extractBody(in); got != "" {
			t.Fatalf("extractBody() = %q, want empty body", got)
		}
	})

	t.Run("BodyContainsDelimiterLine", func(t *testing.T) {
		in := "---\ntitle: t\n---\nline1\n---\nline2\n"
		want := "line1\n---\nline2\n"
		if got := extractBody(in); got != want {
			t.Fatalf("extractBody() = %q, want %q", got, want)
		}
	})
}

func TestNormalizeBody_WhitespaceOnly(t *testing.T) {
	in := "  \t\r\n\t\r\n"
	if got := normalizeBody(in); got != "" {
		t.Fatalf("normalizeBody() = %q, want empty string", got)
	}
}

func TestUpdateDailyReport_BodyChangedRewritesFile(t *testing.T) {
	tmpFile := t.TempDir() + "/daily.md"
	orgName := "test-org"
	publicRepos := make(map[string]struct{})
	var prs []github.Item

	existing := "---\n" +
		"title: AUTO 更新速递\n" +
		"date: \"1999-01-01\"\n" +
		"---\n\n" +
		buildDailyBody(orgName, nil, map[string]string{}, nil, nil)
	if err := os.WriteFile(tmpFile, []byte(existing), 0o644); err != nil {
		t.Fatalf("failed to seed existing daily report: %v", err)
	}

	issues := []github.Item{
		{
			Title:     "new issue",
			URL:       "https://example.com/issues/1",
			CreatedAt: "2026-02-13T10:00:00Z",
			Repository: github.Repository{
				Name: "repo-a",
			},
			Author: github.Author{
				Login: "u1",
			},
		},
	}

	expectedDate := time.Now().UTC().Format("2006-01-02")
	if err := UpdateDailyReport(tmpFile, orgName, publicRepos, issues, prs); err != nil {
		t.Fatalf("UpdateDailyReport() returned error: %v", err)
	}

	after, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	got := string(after)

	if got == existing {
		t.Fatalf("expected file to be rewritten when body changes")
	}
	if strings.Contains(got, "date: \"1999-01-01\"") {
		t.Fatalf("expected date to be refreshed after rewrite")
	}
	if !strings.Contains(got, "date: \""+expectedDate+"\"") {
		t.Fatalf("expected rewritten file to include today's UTC date")
	}
	if !strings.Contains(got, "### [new issue](https://example.com/issues/1)") {
		t.Fatalf("expected new issue content in rewritten file")
	}
}

func TestUpdateDailyReport_ReadFileError(t *testing.T) {
	// Passing a directory path triggers a non-IsNotExist read error.
	path := t.TempDir()
	orgName := "test-org"
	publicRepos := make(map[string]struct{})
	var issues []github.Item
	var prs []github.Item

	err := UpdateDailyReport(path, orgName, publicRepos, issues, prs)
	if err == nil {
		t.Fatalf("expected read error when path is a directory")
	}
	if !strings.Contains(err.Error(), "failed to read existing daily report") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildDailyBody_DeterministicAcrossRuns(t *testing.T) {
	orgName := "test-org"

	date := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)
	commits := []CommitEntry{
		{AuthorName: "zoe", Date: date, Message: "m1", RepoName: "repo-a"},
		{AuthorName: "ann", Date: date, Message: "m2", RepoName: "repo-a"},
		{AuthorName: "bob", Date: date, Message: "m0", RepoName: "repo-b"},
		{AuthorName: "alice", Date: date, Message: "m1", RepoName: "repo-a"},
	}
	issues := []github.Item{
		{
			Title:     "later",
			URL:       "https://example.com/issues/2",
			CreatedAt: "2026-02-14T10:00:00Z",
			Repository: github.Repository{
				Name: "repo-b",
			},
		},
		{
			Title:     "early",
			URL:       "https://example.com/issues/1",
			CreatedAt: "2026-02-13T10:00:00Z",
			Repository: github.Repository{
				Name: "repo-a",
			},
		},
	}
	prs := []github.Item{
		{
			Title:     "bad-time",
			URL:       "https://example.com/pr/3",
			CreatedAt: "not-a-time",
			Repository: github.Repository{
				Name: "repo-a",
			},
		},
		{
			Title:     "newer",
			URL:       "https://example.com/pr/2",
			CreatedAt: "2026-02-14T10:00:00Z",
			Repository: github.Repository{
				Name: "repo-z",
			},
		},
	}

	body1 := buildDailyBody(
		orgName,
		cloneCommits(commits),
		map[string]string{},
		cloneItems(issues),
		cloneItems(prs),
	)
	body2 := buildDailyBody(
		orgName,
		cloneCommits(commits),
		map[string]string{},
		cloneItems(issues),
		cloneItems(prs),
	)
	if body1 != body2 {
		t.Fatalf("buildDailyBody output is not deterministic across runs")
	}
}

func TestBuildDailyBody_SortsCommitsWithTieBreakers(t *testing.T) {
	orgName := "test-org"
	date := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)
	commits := []CommitEntry{
		{AuthorName: "zoe", Date: date, Message: "m1", RepoName: "repo-a"},
		{AuthorName: "ann", Date: date, Message: "m2", RepoName: "repo-a"},
		{AuthorName: "bob", Date: date, Message: "m0", RepoName: "repo-b"},
		{AuthorName: "alice", Date: date, Message: "m1", RepoName: "repo-a"},
	}

	body := buildDailyBody(orgName, commits, map[string]string{}, nil, nil)

	iAlice := strings.Index(body, "alice 在 [repo-a](https://github.com/test-org/repo-a) 中提交了信息：m1")
	iZoe := strings.Index(body, "zoe 在 [repo-a](https://github.com/test-org/repo-a) 中提交了信息：m1")
	iAnn := strings.Index(body, "ann 在 [repo-a](https://github.com/test-org/repo-a) 中提交了信息：m2")
	iBob := strings.Index(body, "bob 在 [repo-b](https://github.com/test-org/repo-b) 中提交了信息：m0")
	if !(iAlice >= 0 && iZoe >= 0 && iAnn >= 0 && iBob >= 0) {
		t.Fatalf("commit lines missing, got:\n%s", body)
	}
	if !(iAlice < iZoe && iZoe < iAnn && iAnn < iBob) {
		t.Fatalf("commit ordering with tie-breakers is incorrect")
	}
}

func cloneItems(items []github.Item) []github.Item {
	out := make([]github.Item, len(items))
	copy(out, items)
	for i := range out {
		if len(items[i].Labels) > 0 {
			out[i].Labels = append([]github.Label(nil), items[i].Labels...)
		}
	}
	return out
}

func cloneCommits(commits []CommitEntry) []CommitEntry {
	out := make([]CommitEntry, len(commits))
	copy(out, commits)
	return out
}
