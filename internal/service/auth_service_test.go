package service

import (
	"testing"
	"time"

	"multi-platform-distribution/internal/auth"

	"github.com/google/uuid"
)

func TestTokenManagerIssueAndVerify(t *testing.T) {
	manager := auth.NewTokenManager("test-secret", "test-issuer", time.Hour)
	adminID := uuid.New()

	token, expiresAt, err := manager.Issue(adminID, "admin@example.com")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	if token == "" {
		t.Fatal("expected token")
	}
	if !expiresAt.After(time.Now()) {
		t.Fatalf("expected future expiry, got %s", expiresAt)
	}

	claims, err := manager.Verify(token)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}
	if claims.AdminID != adminID {
		t.Fatalf("expected admin id %s, got %s", adminID, claims.AdminID)
	}
	if claims.Email != "admin@example.com" {
		t.Fatalf("expected email, got %s", claims.Email)
	}
}
