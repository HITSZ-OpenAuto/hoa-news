package report

import (
	"encoding/json"
	"os"
)

// 解析公开仓库列表，返回字典方便查询
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
