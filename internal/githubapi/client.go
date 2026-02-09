package githubapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	HTTP    *http.Client
	Token   string
	BaseURL string
}

func NewClient(token string) *Client {
	return &Client{
		HTTP: &http.Client{
			Timeout: 30 * time.Second,
		},
		Token:   token,
		BaseURL: "https://api.github.com",
	}
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Accept", "application/vnd.github+json")
	if c.Token != "" {
		req.Header.Set("Authorization", "token "+c.Token)
	}
	return c.HTTP.Do(req)
}

func (c *Client) GetJSON(endpoint string, params url.Values, out any) (string, error) {
	base, err := url.Parse(c.BaseURL)
	if err != nil {
		return "", err
	}
	path, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	full := base.ResolveReference(path)
	if params != nil {
		full.RawQuery = params.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, full.String(), nil)
	if err != nil {
		return "", err
	}
	resp, err := c.do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github api error: %s", string(body))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return "", err
	}
	return resp.Header.Get("Link"), nil
}

func ParseNextLink(linkHeader string) string {
	if linkHeader == "" {
		return ""
	}
	parts := strings.Split(linkHeader, ",")
	for _, part := range parts {
		sections := strings.Split(strings.TrimSpace(part), ";")
		if len(sections) < 2 {
			continue
		}
		urlPart := strings.TrimSpace(sections[0])
		relPart := strings.TrimSpace(sections[1])
		if relPart == "rel=\"next\"" {
			return strings.Trim(urlPart, "<>")
		}
	}
	return ""
}
