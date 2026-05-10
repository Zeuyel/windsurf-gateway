package handler

import "testing"

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
