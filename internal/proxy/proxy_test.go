package proxy

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"windsurf-gateway/internal/config"
	"windsurf-gateway/internal/database"
)

func TestCreateRequestSkipsMalformedProtoRewrite(t *testing.T) {
	svc := NewProxyService(&config.Config{
		Proxy: config.ProxyConfig{
			Timeout:             30 * time.Second,
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     30 * time.Second,
			PrivacyMode:         true,
		},
	})

	token := &database.Token{
		ID:    "token-1",
		Token: "sk-ws-01-backend-token",
	}
	originalBody := []byte{0x07, 0x01, 0x02}
	req, err := svc.createRequest(context.Background(), &ProxyRequest{
		Token:       token,
		Method:      http.MethodPost,
		Path:        "/exa.api_server_pb.ApiServerService/GetStatus",
		Headers:     http.Header{"Cookie": []string{"a=b"}, "X-Forwarded-For": []string{"1.2.3.4"}},
		Body:        originalBody,
		ContentType: "application/proto",
		UserAgent:   "connect-go/1.18.1 (go1.26.1)",
		ClientIP:    "127.0.0.1",
	}, "https://server.codeium.com/exa.api_server_pb.ApiServerService/GetStatus")
	if err != nil {
		t.Fatalf("createRequest returned error: %v", err)
	}

	if req.ContentLength != int64(len(originalBody)) {
		t.Fatalf("unexpected content length: got %d want %d", req.ContentLength, len(originalBody))
	}
	if auth := req.Header.Get("Authorization"); auth != "Bearer "+token.Token {
		t.Fatalf("unexpected authorization header: %q", auth)
	}
	if got := req.Header.Get("User-Agent"); got != "connect-go/1.18.1 (go1.26.1)" {
		t.Fatalf("expected real upstream user-agent passthrough, got %q", got)
	}
	if got := req.Header.Get("Accept"); got != "" {
		t.Fatalf("expected Accept header to remain absent when client omitted it, got %q", got)
	}
	if got := req.Header.Get("Origin"); got != "" {
		t.Fatalf("expected Origin header to remain absent, got %q", got)
	}
	if got := req.Header.Get("Referer"); got != "" {
		t.Fatalf("expected Referer header to remain absent, got %q", got)
	}
	if got := req.Header.Get("X-Request-Session-Id"); len(got) != 64 {
		t.Fatalf("expected stable upstream session id, got %q", got)
	}
	if got := req.Header.Get("Cookie"); got != "" {
		t.Fatalf("expected cookie header to be stripped, got %q", got)
	}
	if got := req.Header.Get("X-Forwarded-For"); got != "" {
		t.Fatalf("expected X-Forwarded-For to be stripped in privacy mode, got %q", got)
	}
	if got := req.Header.Get("X-Real-Ip"); got != "" {
		t.Fatalf("expected X-Real-IP to be stripped in privacy mode, got %q", got)
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}
	if string(body) != string(originalBody) {
		t.Fatalf("request body changed unexpectedly: got %v want %v", body, originalBody)
	}
}
