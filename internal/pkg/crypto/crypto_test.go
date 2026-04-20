package crypto

import (
	"encoding/base64"
	"testing"
)

// 生成一个 32 字节的测试密钥（base64 编码）
func generateTestKey() string {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	return base64.StdEncoding.EncodeToString(key)
}

func TestEncryptDecrypt(t *testing.T) {
	key := generateTestKey()
	plaintext := "Hello, World!"

	ciphertext, nonce, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(ciphertext, nonce, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Decrypted text doesn't match: got %q, want %q", decrypted, plaintext)
	}
}

func TestDifferentNonceProducesDifferentCiphertext(t *testing.T) {
	key := generateTestKey()
	plaintext := "Test message"

	ciphertext1, nonce1, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("First encrypt failed: %v", err)
	}

	ciphertext2, nonce2, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Second encrypt failed: %v", err)
	}

	if ciphertext1 == ciphertext2 {
		t.Error("Same plaintext should produce different ciphertexts with different nonces")
	}

	if nonce1 == nonce2 {
		t.Error("Nonce should be different for each encryption")
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	key := generateTestKey()
	wrongKey := base64.StdEncoding.EncodeToString(make([]byte, 32))
	plaintext := "Secret message"

	ciphertext, nonce, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = Decrypt(ciphertext, nonce, wrongKey)
	if err != ErrDecryptFailed {
		t.Errorf("Expected ErrDecryptFailed, got: %v", err)
	}
}

func TestDecryptWithFallback(t *testing.T) {
	oldKey := generateTestKey()
	newKey := base64.StdEncoding.EncodeToString(make([]byte, 32))
	plaintext := "Fallback test message"

	// 使用旧密钥加密
	ciphertext, nonce, err := Encrypt(plaintext, oldKey)
	if err != nil {
		t.Fatalf("Encrypt with old key failed: %v", err)
	}

	// 尝试用新密钥解密，失败后用旧密钥
	decrypted, err := DecryptWithFallback(ciphertext, nonce, newKey, oldKey)
	if err != nil {
		t.Fatalf("DecryptWithFallback failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Decrypted text doesn't match: got %q, want %q", decrypted, plaintext)
	}
}

func TestDecryptWithFallbackNoPreviousKey(t *testing.T) {
	key := generateTestKey()
	wrongKey := base64.StdEncoding.EncodeToString(make([]byte, 32))
	plaintext := "Test message"

	ciphertext, nonce, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// 使用错误的当前密钥，且没有旧密钥
	_, err = DecryptWithFallback(ciphertext, nonce, wrongKey, "")
	if err != ErrDecryptFailed {
		t.Errorf("Expected ErrDecryptFailed, got: %v", err)
	}
}

func TestInvalidKey(t *testing.T) {
	// 测试非 base64 编码的密钥
	_, _, err := Encrypt("test", "invalid-key")
	if err != ErrInvalidKey {
		t.Errorf("Expected ErrInvalidKey for invalid base64, got: %v", err)
	}

	// 测试长度不正确的密钥（不是 32 字节）
	shortKey := base64.StdEncoding.EncodeToString(make([]byte, 16))
	_, _, err = Encrypt("test", shortKey)
	if err != ErrInvalidKey {
		t.Errorf("Expected ErrInvalidKey for short key, got: %v", err)
	}
}
