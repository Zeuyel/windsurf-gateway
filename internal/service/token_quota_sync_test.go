package service

import (
	"testing"

	"windsurf-gateway/internal/database"
)

func TestBuildQuotaSyncUserStatusBodyUsesPlainProto(t *testing.T) {
	token := &database.Token{
		ID:            "token-1",
		Name:          "test",
		TenantAddress: "https://server.codeium.com",
		Token:         "sk-ws-01-backend",
	}

	body := buildQuotaSyncUserStatusBody(token)
	if len(body) == 0 {
		t.Fatal("expected non-empty request body")
	}
	if body[0] != 0x0a {
		t.Fatalf("expected plain proto metadata field prefix 0x0a, got 0x%02x", body[0])
	}
}
