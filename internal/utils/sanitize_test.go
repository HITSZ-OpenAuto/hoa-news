package utils

import "testing"

func TestSanitizeInlineText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Keep common human style",
			input:    "增加 25 秋考试信息 (#39)",
			expected: "增加 25 秋考试信息 (#39)",
		},
		{
			name:     "Fold line breaks and tabs",
			input:    "line1\r\nline2\tline3",
			expected: "line1 line2 line3",
		},
		{
			name:     "Drop control chars",
			input:    "abc\x00def",
			expected: "abcdef",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeInlineText(tt.input)
			if got != tt.expected {
				t.Fatalf("SanitizeInlineText() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSanitizeLinkLabel(t *testing.T) {
	input := "Fix [Parser] (#10)"
	expected := "Fix \\[Parser\\] (#10)"
	got := SanitizeLinkLabel(input)
	if got != expected {
		t.Fatalf("SanitizeLinkLabel() = %q, want %q", got, expected)
	}
}

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name  string
		raw   string
		allow bool
	}{
		{
			name:  "Allow github",
			raw:   "https://github.com/HITSZ-OpenAuto/hoa-news/pull/1",
			allow: true,
		},
		{
			name:  "Allow arbitrary host",
			raw:   "https://docs.hoa.moe/guide",
			allow: true,
		},
		{
			name:  "Allow non-https for compatibility",
			raw:   "http://github.com/HITSZ-OpenAuto/hoa-news",
			allow: true,
		},
		{
			name:  "Allow non-whitelist host",
			raw:   "https://evil.com/phishing",
			allow: true,
		},
		{
			name:  "Reject control characters",
			raw:   "https://github.com/a\tb",
			allow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := SanitizeURL(tt.raw)
			if ok != tt.allow {
				t.Fatalf("SanitizeURL() allow=%v, want %v", ok, tt.allow)
			}
		})
	}
}

func TestRenderSafeMarkdownLink(t *testing.T) {
	label := "Fix [Parser] (#10)"

	gotAllowed := RenderSafeMarkdownLink(label, "https://github.com/test/repo/pull/1")
	wantAllowed := "[Fix \\[Parser\\] (#10)](https://github.com/test/repo/pull/1)"
	if gotAllowed != wantAllowed {
		t.Fatalf("RenderSafeMarkdownLink() allowed = %q, want %q", gotAllowed, wantAllowed)
	}

	gotAnyHost := RenderSafeMarkdownLink(label, "https://evil.com/steal")
	wantAnyHost := "[Fix \\[Parser\\] (#10)](https://evil.com/steal)"
	if gotAnyHost != wantAnyHost {
		t.Fatalf("RenderSafeMarkdownLink() any host = %q, want %q", gotAnyHost, wantAnyHost)
	}

	gotInvalid := RenderSafeMarkdownLink(label, "https://github.com/a\nb")
	wantInvalid := "Fix \\[Parser\\] (#10)"
	if gotInvalid != wantInvalid {
		t.Fatalf("RenderSafeMarkdownLink() invalid = %q, want %q", gotInvalid, wantInvalid)
	}
}
