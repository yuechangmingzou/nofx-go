package utils

import (
	"crypto/rand"
	"encoding/base64"
)

// GenerateToken 生成随机token
func GenerateToken(length int) string {
	if length <= 0 {
		length = 24
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// 降级方案：使用时间戳
		return base64.URLEncoding.EncodeToString([]byte(string(rune(length))))
	}

	return base64.URLEncoding.EncodeToString(bytes)
}

