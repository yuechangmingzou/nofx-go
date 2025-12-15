package utils

import (
	"regexp"
	"strings"
)

// SanitizeString 脱敏字符串中的敏感信息
func SanitizeString(text string) string {
	if text == "" {
		return text
	}

	patterns := []struct {
		pattern     *regexp.Regexp
		replacement string
	}{
		{regexp.MustCompile(`(?i)api[_-]?key["']?\s*[:=]\s*["']?([a-zA-Z0-9_-]{10,})["']?`), `api_key="***"`},
		{regexp.MustCompile(`(?i)secret[_-]?key["']?\s*[:=]\s*["']?([a-zA-Z0-9_-]{10,})["']?`), `secret_key="***"`},
		{regexp.MustCompile(`(?i)password["']?\s*[:=]\s*["']?([^"']+)["']?`), `password="***"`},
		{regexp.MustCompile(`(?i)token["']?\s*[:=]\s*["']?([a-zA-Z0-9_-]{20,})["']?`), `token="***"`},
		{regexp.MustCompile(`(?i)authorization["']?\s*[:=]\s*["']?([^"']+)["']?`), `authorization="***"`},
		{regexp.MustCompile(`(?i)Bearer\s+([A-Za-z0-9\-_\.=]{20,})`), `Bearer ***`},
		{regexp.MustCompile(`(?i)Basic\s+([A-Za-z0-9\-_\.=]{10,})`), `Basic ***`},
		{regexp.MustCompile(`sk-[A-Za-z0-9\-_\.]{20,}`), `sk-***`},
		{regexp.MustCompile(`sk_proj-[A-Za-z0-9\-_\.]{20,}`), `sk_proj-***`},
		{regexp.MustCompile(`([?&])(api[_-]?key|secret[_-]?key|token|password|auth)=([A-Za-z0-9\-_\.=]{10,})`), `${1}${2}=***`},
		{regexp.MustCompile(`(https?://[^\s]+[?&](?:api[_-]?key|secret[_-]?key|token|password|auth)=)([A-Za-z0-9\-_\.=]{10,})`), `${1}***`},
		{regexp.MustCompile(`\b([A-Za-z0-9]{40,})\b`), `***`},
	}

	result := text
	for _, p := range patterns {
		result = p.pattern.ReplaceAllString(result, p.replacement)
	}
	return result
}

// NormalizeSymbol 规范化交易对符号
func NormalizeSymbol(symbol string) string {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	// 移除可能的空格和特殊字符
	symbol = strings.ReplaceAll(symbol, " ", "")
	symbol = strings.ReplaceAll(symbol, "-", "")
	return symbol
}

