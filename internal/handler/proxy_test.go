package handler

import (
	"net/http"
	"testing"

	"windsurf-gateway/internal/proxy"
)

func TestExtractGatewayUserToken(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{
			name:   "gateway bearer token",
			header: "Bearer ws-abc123",
			want:   "ws-abc123",
		},
		{
			name:   "gateway basic token",
			header: "Basic ws-abc123",
			want:   "ws-abc123",
		},
		{
			name:   "gateway duplicated basic token",
			header: "Basic ws-abc123-ws-abc123",
			want:   "ws-abc123",
		},
		{
			name:   "bearer upstream token is ignored",
			header: "Bearer sk-ws-01-backend-token",
			want:   "",
		},
		{
			name:   "basic placeholder is ignored",
			header: "Basic sk-ws-01-gateway-placeholder-sk-ws-01-gateway-placeholder",
			want:   "",
		},
		{
			name:   "raw token is ignored",
			header: "ws-abc123",
			want:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractGatewayUserToken(tc.header); got != tc.want {
				t.Fatalf("extractGatewayUserToken(%q) = %q, want %q", tc.header, got, tc.want)
			}
		})
	}
}

func TestClassifyProxyOutcomeOptional401DoesNotPenalizeToken(t *testing.T) {
	outcome := classifyProxyOutcome("/exa.api_server_pb.ApiServerService/CheckChatCapacity", &proxy.ProxyResponse{
		StatusCode: http.StatusUnauthorized,
	}, nil)

	if outcome.Penalize {
		t.Fatal("expected optional 401 not to penalize token")
	}
	if outcome.FailureCategory != "optional_upstream_401" {
		t.Fatalf("unexpected failure category: %s", outcome.FailureCategory)
	}
}

func TestBuildAnonymousAssignmentKeyUsesAuthAndIPOnly(t *testing.T) {
	keyA := buildAnonymousAssignmentKey("Basic sk-ws-01-client-abc123", "127.0.0.1")
	keyB := buildAnonymousAssignmentKey("Basic sk-ws-01-client-abc123", "127.0.0.2")
	keyC := buildAnonymousAssignmentKey("Basic sk-ws-01-client-other", "127.0.0.1")

	if keyA != keyB {
		t.Fatalf("expected per-client placeholder auth to be stable across IPs: %s vs %s", keyA, keyB)
	}
	if keyA == keyC {
		t.Fatalf("expected different placeholder auth to produce different keys: %s vs %s", keyA, keyC)
	}
}

func TestBuildAnonymousAssignmentKeyFallsBackToIPForLegacyPlaceholder(t *testing.T) {
	keyA := buildAnonymousAssignmentKey("Basic sk-ws-01-gateway-placeholder", "127.0.0.1")
	keyB := buildAnonymousAssignmentKey("Basic sk-ws-01-gateway-placeholder", "127.0.0.2")

	if keyA == keyB {
		t.Fatalf("expected legacy shared placeholder auth to keep IP in the key: %s vs %s", keyA, keyB)
	}
}
