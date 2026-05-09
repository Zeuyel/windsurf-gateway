package proxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

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
	Token         *database.Token
	Method        string
	Path          string
	Headers       http.Header
	Body          []byte
	ClientIP      string
	UserAgent     string
	TenantAddress string
}

type ProxyResponse struct {
	StatusCode   int
	Headers      http.Header
	Body         []byte
	Size         int64
	Latency      time.Duration
	ErrorMessage string
}

func (p *ProxyService) hasProxyConfigured(token *database.Token) bool {
	return token != nil && token.ProxyURL != nil && *token.ProxyURL != ""
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

func (p *ProxyService) ForwardStream(ctx context.Context, req *ProxyRequest, w http.ResponseWriter) ([]byte, error) {
	startTime := time.Now()

	targetURL, err := p.buildTargetURL(req.TenantAddress, req.Path)
	if err != nil {
		return nil, fmt.Errorf("build target URL: %w", err)
	}

	client, err := p.getOrCreateClient(req.Token, 30*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	httpReq, err := p.createRequest(ctx, req, targetURL)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	logger.Infof("[Proxy] Streaming %s %s", req.Method, targetURL)

	resp, err := client.Do(httpReq)
	if err != nil {
		logger.Errorf("[Proxy] Request failed: %v, latency: %v", err, time.Since(startTime))
		return nil, fmt.Errorf("forward request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		logger.Warnf("[Proxy] Token banned: %d", resp.StatusCode)
		return nil, fmt.Errorf("token banned: status %d", resp.StatusCode)
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	streamCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	bufferSize := 16 * 1024
	if strings.Contains(req.Path, "chat") || strings.Contains(req.Path, "stream") {
		bufferSize = 8 * 1024
	}
	buffer := make([]byte, bufferSize)
	var captured []byte
	totalBytes := int64(0)
	lastDataTime := time.Now()
	var headersWritten bool

	for {
		select {
		case <-streamCtx.Done():
			logger.Infof("[Proxy] Stream cancelled: %d bytes, %v", totalBytes, time.Since(startTime))
			return captured, nil
		default:
		}

		if time.Since(lastDataTime) > 2*time.Minute {
			logger.Infof("[Proxy] Stream idle timeout")
			break
		}

		n, err := resp.Body.Read(buffer)
		if n > 0 {
			lastDataTime = time.Now()
			totalBytes += int64(n)
			captured = append(captured, buffer[:n]...)

			if !headersWritten {
				for key, values := range resp.Header {
					if strings.ToLower(key) == "content-length" {
						continue
					}
					for _, value := range values {
						w.Header().Add(key, value)
					}
				}
				w.Header().Set("Transfer-Encoding", "chunked")
				w.Header().Set("Cache-Control", "no-cache")
				w.Header().Set("Connection", "keep-alive")
				w.WriteHeader(resp.StatusCode)
				headersWritten = true
			}

			if _, err := w.Write(buffer[:n]); err != nil {
				logger.Infof("[Proxy] Client disconnected: %d bytes", totalBytes)
				return captured, nil
			}
			flusher.Flush()
		}

		if err != nil {
			if err == io.EOF {
				if !headersWritten {
					for key, values := range resp.Header {
						if strings.ToLower(key) == "content-length" {
							continue
						}
						for _, value := range values {
							w.Header().Add(key, value)
						}
					}
					w.Header().Set("Transfer-Encoding", "chunked")
					w.WriteHeader(resp.StatusCode)
				}
				logger.Infof("[Proxy] Stream complete: %d bytes, %v", totalBytes, time.Since(startTime))
				break
			}
			logger.Errorf("[Proxy] Stream read error: %v", err)
			return captured, fmt.Errorf("stream read: %w", err)
		}
	}

	flusher.Flush()
	return captured, nil
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
	} else {
		if cleanTarget == "" {
			finalPath = basePath + "/"
		} else {
			finalPath = basePath + "/" + cleanTarget
		}
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
	var body io.Reader
	if len(req.Body) > 0 {
		body = bytes.NewReader(req.Body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, targetURL, body)
	if err != nil {
		return nil, err
	}

	for key, values := range req.Headers {
		if p.shouldSkipHeader(key) {
			continue
		}
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}

	if req.Token != nil && req.Token.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.Token.Token)
	}

	if targetURL, err := url.Parse(req.TenantAddress); err == nil {
		httpReq.Header.Set("host", targetURL.Host)
		httpReq.Host = targetURL.Host
	}

	if req.UserAgent != "" {
		httpReq.Header.Set("User-Agent", req.UserAgent)
	}

	if req.ClientIP != "" {
		httpReq.Header.Set("X-Forwarded-For", req.ClientIP)
		httpReq.Header.Set("X-Real-IP", req.ClientIP)
	}

	return httpReq, nil
}

func (p *ProxyService) shouldSkipHeader(key string) bool {
	key = strings.ToLower(key)
	skipHeaders := []string{
		"proxy-connection", "proxy-authenticate", "proxy-authorization",
		"authorization", "te", "trailers", "transfer-encoding",
		"cf-ray", "cf-visitor", "cf-connecting-ip", "cf-ipcountry",
	}
	for _, h := range skipHeaders {
		if key == h {
			return true
		}
	}
	return false
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
