// Package signature подписывает данные HMAC-SHA256 и проверяет подписи.
package signature

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// Header — имя HTTP-заголовка с подписью.
const Header = "HashSHA256"

// Sum возвращает HMAC-SHA256 от data в hex.
func Sum(key string, data []byte) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

// Valid проверяет, что got совпадает с подписью data.
func Valid(key string, data []byte, got string) bool {
	gotSum, err := hex.DecodeString(got)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(data)
	return hmac.Equal(mac.Sum(nil), gotSum)
}
