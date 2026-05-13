package proxy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"windsurf-gateway/internal/config"
	"windsurf-gateway/internal/database"
	"windsurf-gateway/internal/logger"
)

type ProxyService struct {
	client     *http.Client
	config     *config.Config
	clientPool map[string]*http.Client
	poolMutex  sync.RWMutex
}

func NewProxyService(cfg *config.Config) *ProxyService {
	transport := &http.Transport{
		MaxIdleConns:        cfg.Proxy.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.Proxy.MaxIdleConnsPerHost,
		IdleConnTimeout:     cfg.Proxy.IdleConnTimeout,
		ForceAttemptHTTP2:   false,
		TLSNextProto:        make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
		MaxConnsPerHost:     100,
	}

	client := &http.Client{
		Timeout:   cfg.Proxy.Timeout,
		Transport: transport,
	}

	return &ProxyService{
		client:     client,
		config:     cfg,
		clientPool: make(map[string]*http.Client),
	}
}

type ProxyRequest struct {
	Token               *database.Token
	Method              string
	Path                string
	Headers             http.Header
	Body                []byte
	CaptureResponseBody bool
	ContentType         string
	ClientIP            string
	UserAgent           string
	TenantAddress       string
}

type ProxyResponse struct {
	StatusCode   int
	Headers      http.Header
	Size         int64
	Latency      time.Duration
	ErrorMessage string
	BodySnippet  string
	CapturedBody []byte
}

const defaultGatewayUserAgent = "WindsurfGateway/1.0"

func (p *ProxyService) hasProxyConfigured(token *database.Token) bool {
	return token != nil && token.ProxyURL != nil && *token.ProxyURL != ""
}

func (p *ProxyService) BlockTelemetryEnabled() bool {
	return p != nil && p.config != nil && p.config.Proxy.BlockTelemetry
}

func (p *ProxyService) getClientPoolKey(token *database.Token) string {
	if !p.hasProxyConfigured(token) {
		return "direct"
	}
	return *token.ProxyURL
}

func (p *ProxyService) getOrCreateClient(token *database.Token, timeout time.Duration) (*http.Client, error) {
	poolKey := p.getClientPoolKey(token)

	p.poolMutex.RLock()
	if client, exists := p.clientPool[poolKey]; exists {
		p.poolMutex.RUnlock()
		client.Timeout = timeout
		return client, nil
	}
	p.poolMutex.RUnlock()

	p.poolMutex.Lock()
	defer p.poolMutex.Unlock()

	if client, exists := p.clientPool[poolKey]; exists {
		client.Timeout = timeout
		return client, nil
	}

	transport := p.client.Transport.(*http.Transport).Clone()
	if p.hasProxyConfigured(token) {
		proxyURL, err := url.Parse(*token.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	transport.ForceAttemptHTTP2 = false
	transport.TLSNextProto = make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)

	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
	p.clientPool[poolKey] = client
	return client, nil
}

func (p *ProxyService) ForwardStream(ctx context.Context, req *ProxyRequest, w http.ResponseWriter) (*ProxyResponse, error) {
	startTime := time.Now()

	targetURL, err := p.buildTargetURL(req.TenantAddress, req.Path)
	if err != nil {
		return &ProxyResponse{StatusCode: http.StatusBadGateway, Latency: time.Since(startTime), ErrorMessage: err.Error()}, fmt.Errorf("build target URL: %w", err)
	}

	client, err := p.getOrCreateClient(req.Token, p.config.Proxy.Timeout)
	if err != nil {
		return &ProxyResponse{StatusCode: http.StatusBadGateway, Latency: time.Since(startTime), ErrorMessage: err.Error()}, fmt.Errorf("create client: %w", err)
	}

	httpReq, err := p.createRequest(ctx, req, targetURL)
	if err != nil {
		return &ProxyResponse{StatusCode: http.StatusBadGateway, Latency: time.Since(startTime), ErrorMessage: err.Error()}, fmt.Errorf("create request: %w", err)
	}

	logger.Infof(
		"[Proxy] Forwarding %s %s via token=%s ua=%q content_type=%q accept=%q origin=%q referer=%q",
		req.Method,
		targetURL,
		req.Token.ID,
		httpReq.Header.Get("User-Agent"),
		httpReq.Header.Get("Content-Type"),
		httpReq.Header.Get("Accept"),
		httpReq.Header.Get("Origin"),
		httpReq.Header.Get("Referer"),
	)

	resp, err := client.Do(httpReq)
	if err != nil {
		latency := time.Since(startTime)
		logger.Errorf("[Proxy] Request failed: %v, latency=%v", err, latency)
		return &ProxyResponse{StatusCode: http.StatusBadGateway, Latency: latency, ErrorMessage: err.Error()}, fmt.Errorf("forward request: %w", err)
	}
	defer resp.Body.Close()

	copyResponseHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	flusher, _ := w.(http.Flusher)
	if flusher != nil {
		flusher.Flush()
	}

	bufferSize := 32 * 1024
	if strings.Contains(strings.ToLower(req.Path), "stream") || strings.Contains(strings.ToLower(req.Path), "chat") {
		bufferSize = 8 * 1024
	}
	buffer := make([]byte, bufferSize)
	var snippet bytes.Buffer
	var captured bytes.Buffer
	var totalBytes int64

	for {
		n, readErr := resp.Body.Read(buffer)
		if n > 0 {
			chunk := buffer[:n]
			totalBytes += int64(n)
			if snippet.Len() < 4096 {
				remaining := 4096 - snippet.Len()
				if remaining > n {
					remaining = n
				}
				snippet.Write(chunk[:remaining])
			}
			if req.CaptureResponseBody && captured.Len() < 256*1024 {
				remaining := 256*1024 - captured.Len()
				if remaining > n {
					remaining = n
				}
				captured.Write(chunk[:remaining])
			}
			if _, writeErr := w.Write(chunk); writeErr != nil {
				latency := time.Since(startTime)
				return &ProxyResponse{
					StatusCode:   resp.StatusCode,
					Headers:      resp.Header,
					Size:         totalBytes,
					Latency:      latency,
					ErrorMessage: writeErr.Error(),
					BodySnippet:  buildBodySnippet(resp.Header, snippet.Bytes()),
					CapturedBody: cloneBytes(captured.Bytes()),
				}, fmt.Errorf("write response: %w", writeErr)
			}
			if flusher != nil {
				flusher.Flush()
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				return &ProxyResponse{
					StatusCode:   resp.StatusCode,
					Headers:      resp.Header,
					Size:         totalBytes,
					Latency:      time.Since(startTime),
					BodySnippet:  buildBodySnippet(resp.Header, snippet.Bytes()),
					CapturedBody: cloneBytes(captured.Bytes()),
				}, nil
			}
			latency := time.Since(startTime)
			return &ProxyResponse{
				StatusCode:   resp.StatusCode,
				Headers:      resp.Header,
				Size:         totalBytes,
				Latency:      latency,
				ErrorMessage: readErr.Error(),
				BodySnippet:  buildBodySnippet(resp.Header, snippet.Bytes()),
				CapturedBody: cloneBytes(captured.Bytes()),
			}, fmt.Errorf("read upstream response: %w", readErr)
		}
	}
}

func buildBodySnippet(headers http.Header, body []byte) string {
	if len(body) == 0 {
		return ""
	}

	contentEncoding := strings.ToLower(headers.Get("Content-Encoding"))
	if strings.Contains(contentEncoding, "gzip") {
		return fmt.Sprintf("<gzip %d bytes>", len(body))
	}

	contentType := strings.ToLower(headers.Get("Content-Type"))
	switch {
	case strings.Contains(contentType, "application/proto"),
		strings.Contains(contentType, "application/protobuf"),
		strings.Contains(contentType, "application/connect+proto"),
		strings.Contains(contentType, "application/octet-stream"):
		return fmt.Sprintf("<binary %d bytes>", len(body))
	}

	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return ""
	}

	if utf8.Valid(trimmed) && isTextLikePayload(trimmed) {
		return string(trimmed)
	}

	return fmt.Sprintf("<binary %d bytes>", len(body))
}

func isTextLikePayload(body []byte) bool {
	sample := body
	if len(sample) > 512 {
		sample = sample[:512]
	}

	for _, b := range sample {
		if b == '\t' || b == '\n' || b == '\r' {
			continue
		}
		if b < 0x20 || b == 0x7f {
			return false
		}
	}

	return true
}

func cloneBytes(value []byte) []byte {
	if len(value) == 0 {
		return nil
	}
	out := make([]byte, len(value))
	copy(out, value)
	return out
}

func (p *ProxyService) buildUpstreamUserAgent(req *ProxyRequest) string {
	if p.config.Subscription.UserAgent != "" {
		return p.config.Subscription.UserAgent
	}
	if req != nil && strings.TrimSpace(req.UserAgent) != "" {
		return strings.TrimSpace(req.UserAgent)
	}
	return defaultGatewayUserAgent
}

func buildStableSessionID(token *database.Token) string {
	if token == nil {
		return ""
	}
	return stableProfileComponent("session", token, 64)
}

func buildStableDeviceFingerprint(token *database.Token) string {
	if token == nil {
		return ""
	}
	return stableProfileComponent("device", token, 64)
}

func stableProfileComponent(kind string, token *database.Token, length int) string {
	seed := stableProfileSeed(token)
	sum := sha256.Sum256([]byte(kind + "|" + seed))
	value := hex.EncodeToString(sum[:])
	if length > 0 && length < len(value) {
		return value[:length]
	}
	return value
}

func stableProfileSeed(token *database.Token) string {
	if token == nil {
		return ""
	}
	if seed := strings.TrimSpace(token.PrivacySeed); seed != "" {
		return seed
	}

	parts := []string{token.ID, token.Name, token.TenantAddress}
	if token.Email != nil && strings.TrimSpace(*token.Email) != "" {
		parts = append(parts, strings.ToLower(strings.TrimSpace(*token.Email)))
	}
	return strings.Join(parts, "|")
}

func (p *ProxyService) buildTargetURL(tenantAddress, path string) (string, error) {
	if !strings.HasPrefix(tenantAddress, "http://") && !strings.HasPrefix(tenantAddress, "https://") {
		tenantAddress = "https://" + tenantAddress
	}

	baseURL, err := url.Parse(tenantAddress)
	if err != nil {
		return "", fmt.Errorf("invalid tenant address: %w", err)
	}

	path = strings.TrimPrefix(path, "/proxy")
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	var targetPath, rawQuery string
	if idx := strings.Index(path, "?"); idx != -1 {
		targetPath = path[:idx]
		rawQuery = path[idx+1:]
	} else {
		targetPath = path
	}

	basePath := strings.TrimSuffix(baseURL.Path, "/")
	cleanTarget := strings.TrimPrefix(targetPath, "/")

	var finalPath string
	if basePath == "" {
		if cleanTarget == "" {
			finalPath = "/"
		} else {
			finalPath = "/" + cleanTarget
		}
	} else if cleanTarget == "" {
		finalPath = basePath + "/"
	} else {
		finalPath = basePath + "/" + cleanTarget
	}

	finalPath = strings.ReplaceAll(finalPath, "//", "/")
	if !strings.HasPrefix(finalPath, "/") {
		finalPath = "/" + finalPath
	}

	return (&url.URL{
		Scheme:   baseURL.Scheme,
		Host:     baseURL.Host,
		Path:     finalPath,
		RawQuery: rawQuery,
	}).String(), nil
}

func (p *ProxyService) createRequest(ctx context.Context, req *ProxyRequest, targetURL string) (*http.Request, error) {
	bodyBytes := req.Body
	if req.Token != nil && req.Token.Token != "" {
		rewrittenBody, err := rewriteUpstreamAuthPayload(req.Path, req.ContentType, req.Body, upstreamAuthProfile{
			BackendToken:            req.Token.Token,
			StableSessionID:         buildStableSessionID(req.Token),
			StableUserAgent:         p.buildUpstreamUserAgent(req),
			StableDeviceFingerprint: buildStableDeviceFingerprint(req.Token),
		})
		if err != nil {
			logger.Warnf("[Proxy] Skip auth payload rewrite for %s %s: %v", req.Method, req.Path, err)
		} else {
			bodyBytes = rewrittenBody
		}
	}

	var body io.Reader
	if len(bodyBytes) > 0 {
		body = bytes.NewReader(bodyBytes)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, targetURL, body)
	if err != nil {
		return nil, err
	}

	for key, values := range req.Headers {
		if p.shouldSkipRequestHeader(key) {
			continue
		}
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}

	if req.Token != nil && req.Token.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.Token.Token)
		httpReq.Header.Set("X-Api-Key", req.Token.Token)
	}
	if len(bodyBytes) > 0 {
		httpReq.ContentLength = int64(len(bodyBytes))
	}

	parsedTarget, err := url.Parse(targetURL)
	if err == nil {
		httpReq.Host = parsedTarget.Host
		httpReq.Header.Set("Host", parsedTarget.Host)
	}

	userAgent := p.buildUpstreamUserAgent(req)
	if userAgent != "" {
		httpReq.Header.Set("User-Agent", userAgent)
	}

	if !p.config.Proxy.PrivacyMode && req.ClientIP != "" {
		httpReq.Header.Set("X-Real-IP", req.ClientIP)
		existing := strings.TrimSpace(req.Headers.Get("X-Forwarded-For"))
		if existing != "" {
			httpReq.Header.Set("X-Forwarded-For", existing+", "+req.ClientIP)
		} else {
			httpReq.Header.Set("X-Forwarded-For", req.ClientIP)
		}
	}

	return httpReq, nil
}

func (p *ProxyService) shouldSkipRequestHeader(key string) bool {
	switch strings.ToLower(key) {
	case "authorization", "x-api-key", "host", "proxy-connection", "proxy-authenticate", "proxy-authorization", "te", "trailers", "transfer-encoding", "connection", "x-forwarded-for", "x-real-ip", "cookie", "x-request-session-id", "origin", "referer":
		return true
	default:
		return false
	}
}

func copyResponseHeaders(dst http.Header, src http.Header) {
	for key, values := range src {
		if shouldSkipResponseHeader(key) {
			continue
		}
		dst.Del(key)
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func shouldSkipResponseHeader(key string) bool {
	switch strings.ToLower(key) {
	case "connection", "keep-alive", "proxy-authenticate", "proxy-authorization", "te", "trailer", "transfer-encoding", "upgrade":
		return true
	default:
		return false
	}
}

func GetClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}
