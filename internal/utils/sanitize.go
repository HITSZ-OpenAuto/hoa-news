package utils

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"
)

// SanitizeInlineText 移除字符串中的控制字符，并将连续的空白字符替换为单个空格。
func SanitizeInlineText(s string) string {
	if s == "" {
		return ""
	}

	var (
		b            strings.Builder
		pendingSpace bool
	)
	b.Grow(len(s))

	for _, r := range s {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			pendingSpace = true
		case unicode.IsControl(r):
			// Drop other control characters.
		default:
			if pendingSpace && b.Len() > 0 {
				b.WriteByte(' ')
			}
			pendingSpace = false
			b.WriteRune(r)
		}
	}

	return strings.TrimSpace(b.String())
}

// SanitizeLinkLabel 对链接标签进行清理，移除控制字符并转义 Markdown 特殊字符。
// 目前转义：反斜杠和方括号。
func SanitizeLinkLabel(s string) string {
	safe := SanitizeInlineText(s)
	return strings.NewReplacer(
		`\`, `\\`,
		`[`, `\[`,
		`]`, `\]`,
	).Replace(safe)
}

// SanitizeURL 清理 URL 字符串，去除首尾空白并验证其格式是否正确。返回清理后的 URL 和一个布尔值表示是否有效。
func SanitizeURL(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false
	}
	if strings.ContainsAny(trimmed, "\r\n\t") {
		return "", false
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed == nil {
		return "", false
	}

	return parsed.String(), true
}

// RenderSafeMarkdownLink 生成一个安全的 Markdown 链接，标签和 URL 都经过清理和验证。如果 URL 无效，则返回只有标签的文本。
func RenderSafeMarkdownLink(label, rawURL string) string {
	safeLabel := SanitizeLinkLabel(label)
	safeURL, ok := SanitizeURL(rawURL)
	if !ok {
		return safeLabel
	}
	return fmt.Sprintf("[%s](%s)", safeLabel, safeURL)
}
