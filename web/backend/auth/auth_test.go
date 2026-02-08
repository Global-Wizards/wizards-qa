package auth

import (
	"testing"

	"github.com/Global-Wizards/wizards-qa/web/backend/store"
)

const testSecret = "test-secret-key-for-jwt"

func testUser() *store.User {
	return &store.User{
		ID:    "user-123",
		Email: "test@example.com",
		Role:  "member",
	}
}

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("mysecretpassword")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword returned empty string")
	}
	if hash == "mysecretpassword" {
		t.Fatal("HashPassword returned plaintext")
	}

	if !CheckPassword(hash, "mysecretpassword") {
		t.Error("CheckPassword should return true for correct password")
	}
	if CheckPassword(hash, "wrongpassword") {
		t.Error("CheckPassword should return false for wrong password")
	}
}

func TestGenerateTokens(t *testing.T) {
	user := testUser()
	access, refresh, err := GenerateTokens(user, testSecret)
	if err != nil {
		t.Fatalf("GenerateTokens failed: %v", err)
	}
	if access == "" {
		t.Error("access token is empty")
	}
	if refresh == "" {
		t.Error("refresh token is empty")
	}
	if access == refresh {
		t.Error("access and refresh tokens should differ")
	}
}

func TestValidateAccessToken(t *testing.T) {
	user := testUser()
	access, _, err := GenerateTokens(user, testSecret)
	if err != nil {
		t.Fatalf("GenerateTokens failed: %v", err)
	}

	claims, err := ValidateAccessToken(access, testSecret)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("UserID = %q, want %q", claims.UserID, user.ID)
	}
	if claims.Email != user.Email {
		t.Errorf("Email = %q, want %q", claims.Email, user.Email)
	}
	if claims.Role != user.Role {
		t.Errorf("Role = %q, want %q", claims.Role, user.Role)
	}
}

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	user := testUser()
	access, _, _ := GenerateTokens(user, testSecret)

	_, err := ValidateAccessToken(access, "wrong-secret")
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestValidateAccessToken_RefreshTokenRejected(t *testing.T) {
	user := testUser()
	_, refresh, _ := GenerateTokens(user, testSecret)

	_, err := ValidateAccessToken(refresh, testSecret)
	if err == nil {
		t.Fatal("expected error when validating refresh token as access token")
	}
}

func TestValidateRefreshToken(t *testing.T) {
	user := testUser()
	_, refresh, err := GenerateTokens(user, testSecret)
	if err != nil {
		t.Fatalf("GenerateTokens failed: %v", err)
	}

	claims, err := ValidateRefreshToken(refresh, testSecret)
	if err != nil {
		t.Fatalf("ValidateRefreshToken failed: %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("UserID = %q, want %q", claims.UserID, user.ID)
	}
}

func TestValidateRefreshToken_AccessTokenRejected(t *testing.T) {
	user := testUser()
	access, _, _ := GenerateTokens(user, testSecret)

	_, err := ValidateRefreshToken(access, testSecret)
	if err == nil {
		t.Fatal("expected error when validating access token as refresh token")
	}
}

func TestValidateAccessToken_InvalidString(t *testing.T) {
	_, err := ValidateAccessToken("not-a-valid-token", testSecret)
	if err == nil {
		t.Fatal("expected error for invalid token string")
	}
}
