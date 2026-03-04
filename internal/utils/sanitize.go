package utils

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"
)

var (
	// htmlEntityReplacer 用于将特殊字符转换为 HTML 实体，以防止在 Markdown 中被误解析。
	htmlEntityReplacer = strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"{", "&#123;",
		"}", "&#125;",
	)
	//  markdownLabelReplacer 用于转义 Markdown 链接标签中的特殊字符，防止它们被误解析。
	markdownLabelReplacer = strings.NewReplacer(
		`\`, `\\`,
		`[`, `\[`,
		`]`, `\]`,
		`(`, `\(`,
		`)`, `\)`,
		`*`, `\*`,
		`_`, `\_`,
		"`", "\\`",
	)
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
			pendingSpace = true // 将换行、回车和制表符视为空格，避免它们直接出现在输出中。
		case unicode.IsControl(r):
			// 跳过其他控制字符
		default:
			// 遇到普通字符，如果前面有 pendingSpace 且当前不是开头，先写一个空格
			if pendingSpace && b.Len() > 0 {
				b.WriteByte(' ')
			}
			pendingSpace = false // 重置 pendingSpace
			b.WriteRune(r)
		}
	}

	safe := strings.TrimSpace(b.String())
	return htmlEntityReplacer.Replace(safe)
}

// SanitizeLinkLabel 对链接标签进行清理，移除控制字符并转义 Markdown 特殊字符。
// 目前转义：反斜杠、方括号、圆括号、星号、下划线和反引号。
func SanitizeLinkLabel(s string) string {
	safe := SanitizeInlineText(s)
	return markdownLabelReplacer.Replace(safe)
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
