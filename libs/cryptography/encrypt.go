package cryptography

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"

	"jaxpay-go-backend/libs/config"
)

func aead() (cipher.AEAD, error) {
	k := config.Get().EncryptionKey
	if k == "" {
		return nil, errors.New("ENCRYPTION_KEY is not set")
	}
	sum := sha256.Sum256([]byte(k))
	block, e := aes.NewCipher(sum[:])
	if e != nil {
		return nil, e
	}
	return cipher.NewGCM(block)
}

// Encrypt mengenkripsi secret (AES-256-GCM) untuk disimpan di DB.
func Encrypt(plain string) (string, error) {
	g, e := aead()
	if e != nil {
		return "", e
	}
	nonce := make([]byte, g.NonceSize())
	if _, e = rand.Read(nonce); e != nil {
		return "", e
	}
	return base64.StdEncoding.EncodeToString(g.Seal(nonce, nonce, []byte(plain), nil)), nil
}

func Decrypt(enc string) (string, error) {
	g, e := aead()
	if e != nil {
		return "", e
	}
	raw, e := base64.StdEncoding.DecodeString(enc)
	if e != nil {
		return "", e
	}
	if len(raw) < g.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	plain, e := g.Open(nil, raw[:g.NonceSize()], raw[g.NonceSize():], nil)
	if e != nil {
		return "", e
	}
	return string(plain), nil
}
