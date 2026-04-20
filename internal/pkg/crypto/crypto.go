package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

var (
	ErrInvalidKey    = errors.New("encryption key must be 32 bytes (base64 encoded)")
	ErrDecryptFailed = errors.New("decryption failed: invalid ciphertext or key")
)

// Encrypt 使用 AES-256-GCM 加密明文，返回 (密文 base64, nonce base64, error)
func Encrypt(plaintext string, keyBase64 string) (ciphertext string, nonce string, err error) {
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil || len(key) != 32 {
		return "", "", ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}

	nonceBytes := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonceBytes); err != nil {
		return "", "", err
	}

	encrypted := aesGCM.Seal(nil, nonceBytes, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(encrypted), base64.StdEncoding.EncodeToString(nonceBytes), nil
}

// Decrypt 使用 AES-256-GCM 解密
func Decrypt(ciphertextBase64 string, nonceBase64 string, keyBase64 string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil || len(key) != 32 {
		return "", ErrInvalidKey
	}

	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", ErrDecryptFailed
	}

	nonceBytes, err := base64.StdEncoding.DecodeString(nonceBase64)
	if err != nil {
		return "", ErrDecryptFailed
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	plaintext, err := aesGCM.Open(nil, nonceBytes, ciphertextBytes, nil)
	if err != nil {
		return "", ErrDecryptFailed
	}

	return string(plaintext), nil
}

// DecryptWithFallback 尝试用当前密钥解密，失败后用旧密钥解密（密钥轮转支持）
func DecryptWithFallback(ciphertextBase64, nonceBase64, currentKey, previousKey string) (string, error) {
	result, err := Decrypt(ciphertextBase64, nonceBase64, currentKey)
	if err == nil {
		return result, nil
	}
	if previousKey != "" {
		return Decrypt(ciphertextBase64, nonceBase64, previousKey)
	}
	return "", err
}
