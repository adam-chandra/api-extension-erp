package hash_test

import (
	"testing"

	"github.com/extension-erp/be-extension-erp/pkg/hash"
)

func TestPassword_ReturnsHash(t *testing.T) {
	hashed, err := hash.Password("secret123")
	if err != nil {
		t.Fatalf("Password() error = %v", err)
	}
	if hashed == "" {
		t.Error("Password() returned empty hash")
	}
	if hashed == "secret123" {
		t.Error("Password() must not return plaintext")
	}
}

func TestPassword_DifferentHashEachCall(t *testing.T) {
	h1, _ := hash.Password("same-password")
	h2, _ := hash.Password("same-password")
	if h1 == h2 {
		t.Error("Password() should produce different hashes each call (bcrypt salting)")
	}
}

func TestCompare_Correct(t *testing.T) {
	plain := "my-secret-pass"
	hashed, _ := hash.Password(plain)
	if !hash.Compare(hashed, plain) {
		t.Error("Compare() should return true for correct password")
	}
}

func TestCompare_Wrong(t *testing.T) {
	hashed, _ := hash.Password("correct-pass")
	if hash.Compare(hashed, "wrong-pass") {
		t.Error("Compare() should return false for wrong password")
	}
}

func TestCompare_EmptyHash(t *testing.T) {
	if hash.Compare("", "anything") {
		t.Error("Compare() should return false for empty hash")
	}
}

func TestCompare_EmptyPassword(t *testing.T) {
	hashed, _ := hash.Password("real-pass")
	if hash.Compare(hashed, "") {
		t.Error("Compare() should return false when plain password is empty")
	}
}
