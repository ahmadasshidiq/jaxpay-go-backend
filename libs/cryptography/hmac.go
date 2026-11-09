package cryptography

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

func HMACSHA256Hex(secret, msg string) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(msg))
	return hex.EncodeToString(m.Sum(nil))
}

func VerifyHMACSHA256(secret, msg, sigHex string) bool {
	return hmac.Equal([]byte(HMACSHA256Hex(secret, msg)), []byte(sigHex))
}

func RandomToken(n int) string {
	b := make([]byte, n)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
