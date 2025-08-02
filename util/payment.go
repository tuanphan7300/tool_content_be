package util

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// VerifySepaySignature xác thực chữ ký từ Sepay
func VerifySepaySignature(data map[string]interface{}, signature, secretKey string) bool {
	// Tạo chuỗi để sign
	var keys []string
	for k := range data {
		if k != "signature" { // Bỏ qua field signature
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var signString strings.Builder
	for _, key := range keys {
		if value, exists := data[key]; exists && value != nil {
			signString.WriteString(fmt.Sprintf("%s=%v&", key, value))
		}
	}

	// Bỏ dấu & cuối cùng
	signStr := strings.TrimSuffix(signString.String(), "&")

	// Tạo HMAC-SHA256
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(signStr))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	return strings.EqualFold(signature, expectedSignature)
}

// GenerateSepaySignature tạo chữ ký để gửi đến Sepay
func GenerateSepaySignature(data map[string]interface{}, secretKey string) string {
	// Tạo chuỗi để sign
	var keys []string
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var signString strings.Builder
	for _, key := range keys {
		if value, exists := data[key]; exists && value != nil {
			signString.WriteString(fmt.Sprintf("%s=%v&", key, value))
		}
	}

	// Bỏ dấu & cuối cùng
	signStr := strings.TrimSuffix(signString.String(), "&")

	// Tạo HMAC-SHA256
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(signStr))
	return hex.EncodeToString(h.Sum(nil))
}
