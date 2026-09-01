package security

import (
	"bytes"
	"testing"
)

// Seção 17: a API Key da OpenAI deve ser cifrada com AES-256-GCM e nunca
// deve ser possível recuperá-la com uma chave mestra diferente da usada
// para cifrar.
func testKey(fill byte) []byte {
	key := make([]byte, 32)
	for i := range key {
		key[i] = fill
	}
	return key
}

func TestEncryptDecryptAPIKey_RoundTrip(t *testing.T) {
	key := testKey(0x01)
	plaintext := "sk-test-1234567890abcdef"

	ciphertext, nonce, err := EncryptAPIKey(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptAPIKey failed: %v", err)
	}
	if len(ciphertext) == 0 || len(nonce) == 0 {
		t.Fatal("expected non-empty ciphertext and nonce")
	}
	if bytes.Contains(ciphertext, []byte(plaintext)) {
		t.Fatal("ciphertext must not contain the plaintext API key")
	}

	decrypted, err := DecryptAPIKey(ciphertext, nonce, key)
	if err != nil {
		t.Fatalf("DecryptAPIKey failed: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("expected decrypted value %q, got %q", plaintext, decrypted)
	}
}

func TestEncryptAPIKey_DifferentNoncePerCall(t *testing.T) {
	key := testKey(0x02)
	_, nonce1, err := EncryptAPIKey("chave-a", key)
	if err != nil {
		t.Fatalf("EncryptAPIKey failed: %v", err)
	}
	_, nonce2, err := EncryptAPIKey("chave-a", key)
	if err != nil {
		t.Fatalf("EncryptAPIKey failed: %v", err)
	}
	if bytes.Equal(nonce1, nonce2) {
		t.Fatal("expected a different random nonce on each call")
	}
}

func TestDecryptAPIKey_FailsWithWrongKey(t *testing.T) {
	correctKey := testKey(0x03)
	wrongKey := testKey(0x04)

	ciphertext, nonce, err := EncryptAPIKey("segredo", correctKey)
	if err != nil {
		t.Fatalf("EncryptAPIKey failed: %v", err)
	}

	if _, err := DecryptAPIKey(ciphertext, nonce, wrongKey); err == nil {
		t.Fatal("expected decryption to fail with the wrong key")
	}
}

func TestDecryptAPIKey_FailsWithTamperedCiphertext(t *testing.T) {
	key := testKey(0x05)
	ciphertext, nonce, err := EncryptAPIKey("segredo", key)
	if err != nil {
		t.Fatalf("EncryptAPIKey failed: %v", err)
	}
	tampered := make([]byte, len(ciphertext))
	copy(tampered, ciphertext)
	tampered[0] ^= 0xFF

	if _, err := DecryptAPIKey(tampered, nonce, key); err == nil {
		t.Fatal("expected decryption to fail for tampered ciphertext (GCM must detect this)")
	}
}
