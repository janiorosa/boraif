package security

import "testing"

func TestHashPassword_RoundTrip(t *testing.T) {
	hash, err := HashPassword("uma-senha-forte-123")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "" {
		t.Fatal("expected a non-empty hash")
	}
	if hash == "uma-senha-forte-123" {
		t.Fatal("hash must not equal the plaintext password")
	}
	if !CheckPassword(hash, "uma-senha-forte-123") {
		t.Fatal("CheckPassword should accept the correct password")
	}
}

func TestCheckPassword_RejectsWrongPassword(t *testing.T) {
	hash, err := HashPassword("senha-correta")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if CheckPassword(hash, "senha-errada") {
		t.Fatal("CheckPassword should reject an incorrect password")
	}
}
