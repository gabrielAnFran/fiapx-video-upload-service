package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateAndValidateToken_Success(t *testing.T) {
	token, err := GenerateToken("user-1", "user@example.com", "secret")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	claims, err := ValidateToken(token, "secret")
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if claims.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-1")
	}
	if claims.Email != "user@example.com" {
		t.Errorf("Email = %q, want %q", claims.Email, "user@example.com")
	}
	if claims.Issuer != tokenIssuer {
		t.Errorf("Issuer = %q, want %q", claims.Issuer, tokenIssuer)
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	token, err := GenerateToken("user-1", "user@example.com", "secret")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	if _, err := ValidateToken(token, "different-secret"); err == nil {
		t.Fatal("expected error validating token with wrong secret")
	}
}

func TestValidateToken_Expired(t *testing.T) {
	claims := &Claims{
		UserID: "user-1",
		Email:  "user@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Issuer:    tokenIssuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	if _, err := ValidateToken(signed, "secret"); err == nil {
		t.Fatal("expected error validating expired token")
	}
}

func TestValidateToken_Malformed(t *testing.T) {
	if _, err := ValidateToken("not-a-jwt", "secret"); err == nil {
		t.Fatal("expected error validating malformed token")
	}
}
