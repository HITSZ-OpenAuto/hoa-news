package news

import "strings"

var knownBots = map[string]struct{}{
	"github-actions":      {},
	"actions-user":        {},
	"github-actions[bot]": {},
	"dependabot":          {},
	"dependabot[bot]":     {},
	"renovate":            {},
	"renovate[bot]":       {},
}

func IsBot(authorName, authorLogin string) bool {
	name := strings.ToLower(strings.TrimSpace(authorName))
	login := strings.ToLower(strings.TrimSpace(authorLogin))
	if name != "" {
		if _, ok := knownBots[name]; ok {
			return true
		}
		if strings.HasSuffix(name, "[bot]") {
			return true
		}
	}
	if login != "" {
		if _, ok := knownBots[login]; ok {
			return true
		}
		if strings.HasSuffix(login, "[bot]") {
			return true
		}
	}
	return false
}
