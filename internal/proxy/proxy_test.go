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
		Headers:     make(http.Header),
		Body:        originalBody,
		ContentType: "application/proto",
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
	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}
	if string(body) != string(originalBody) {
		t.Fatalf("request body changed unexpectedly: got %v want %v", body, originalBody)
	}
}
