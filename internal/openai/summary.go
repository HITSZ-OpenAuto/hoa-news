package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type summaryRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type summaryResponse struct {
	OutputText string `json:"output_text"`
}

func GenerateWeeklySummary(rawUpdates string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY is not set")
	}
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	prompt := fmt.Sprintf(`你将收到一周内学生们在各个课程仓库中的更新记录。  
请根据这些原始更新，生成一个简洁清晰的「每周更新摘要」，要求如下：  

1. 按照课程进行归类，不需要逐日分开。  
2. 同一门课程的多条类似更新请合并，避免重复啰嗦。  
3. 重点突出：
   - 新增的资料、作业、代码、讲义、教材、试卷等。
   - 对课程说明/文档的重要修改。
   - 有价值的合并更新（如 OpenCS 内容）。  
4. 对琐碎操作（如删除无意义的 README、文件改名、格式小改）只需一句话笼统概括。  
5. **如果一周内仅有一个仓库更新，请直接输出 "__NO_SUMMARY__"，不要生成摘要。**  
6. 输出风格应简洁明了，适合在新闻模块展示。  
7. 请以 "## 本周更新摘要" 作为第一行标题。

下面是原始更新内容：  

%s

请生成总结。`, rawUpdates)

	reqBody, err := json.Marshal(summaryRequest{
		Model: "gpt-5-mini",
		Input: prompt,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/responses", bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openai api error: %s", string(body))
	}
	var out summaryResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.OutputText, nil
}
