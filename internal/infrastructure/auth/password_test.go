package auth

import "testing"

func TestHashAndCheckPassword_Success(t *testing.T) {
	hash, err := HashPassword("s3cr3t-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "s3cr3t-password" {
		t.Fatal("hash should not equal the plaintext password")
	}

	if err := CheckPassword(hash, "s3cr3t-password"); err != nil {
		t.Errorf("CheckPassword() error = %v, want nil", err)
	}
}

func TestCheckPassword_WrongPassword(t *testing.T) {
	hash, err := HashPassword("s3cr3t-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if err := CheckPassword(hash, "wrong-password"); err == nil {
		t.Fatal("expected error for mismatched password")
	}
}
