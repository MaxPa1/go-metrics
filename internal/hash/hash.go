package hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

const Header = "HashSHA256"

func Sum(key string, data []byte) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

func Valid(key string, data []byte, got string) bool {
	gotSum, err := hex.DecodeString(got)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(data)
	return hmac.Equal(mac.Sum(nil), gotSum)
}
