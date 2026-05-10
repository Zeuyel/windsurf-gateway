package service

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"windsurf-gateway/internal/database"
)

const (
	quotaSyncRequestPath      = "/exa.seat_management_pb.SeatManagementService/GetUserStatus"
	quotaSyncConnectProtoType = "application/connect+proto"
	quotaSyncRequestTimeout   = 30 * time.Second
	quotaSyncDefaultOrigin    = "https://windsurf.com"
	quotaSyncDefaultReferer   = "https://windsurf.com/"
	quotaSyncGatewayUserAgent = "WindsurfGateway/QuotaSync"
)

func (s *TokenService) SyncQuotaSnapshot(id string) (*database.Token, error) {
	token, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if err := s.syncQuotaSnapshotForToken(token); err != nil {
		return nil, err
	}
	return s.GetByID(id)
}

func (s *TokenService) SyncAllQuotaSnapshots() (int, int, []string, error) {
	tokens, err := s.GetActiveTokens()
	if err != nil {
		return 0, 0, nil, err
	}

	success := 0
	failed := 0
	messages := make([]string, 0, len(tokens))

	for i := range tokens {
		if err := s.syncQuotaSnapshotForToken(&tokens[i]); err != nil {
			failed++
			messages = append(messages, fmt.Sprintf("%s: %v", tokens[i].Name, err))
			continue
		}
		success++
	}

	return success, failed, messages, nil
}

func (s *TokenService) syncQuotaSnapshotForToken(token *database.Token) error {
	headers, payload, err := s.fetchUserStatusSnapshot(token)
	if err != nil {
		return err
	}
	return s.UpdateQuotaFromGetUserStatusResponse(token.ID, headers, payload)
}

func (s *TokenService) fetchUserStatusSnapshot(token *database.Token) (http.Header, []byte, error) {
	if token == nil {
		return nil, nil, fmt.Errorf("token is required")
	}
	if strings.TrimSpace(token.Token) == "" {
		return nil, nil, fmt.Errorf("backend token is empty")
	}

	targetURL, err := buildQuotaSyncTargetURL(token.TenantAddress)
	if err != nil {
		return nil, nil, err
	}

	client, err := newTokenQuotaHTTPClient(token, quotaSyncRequestTimeout)
	if err != nil {
		return nil, nil, err
	}

	// Connect protocol empty message frame.
	body := []byte{0, 0, 0, 0, 0}

	resp, err := doRequestWithRetry(client, 2, func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", quotaSyncConnectProtoType)
		req.Header.Set("Accept", quotaSyncConnectProtoType)
		req.Header.Set("Connect-Protocol-Version", "1")
		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set("Authorization", "Bearer "+token.Token)
		req.Header.Set("X-Api-Key", token.Token)
		req.Header.Set("User-Agent", quotaSyncGatewayUserAgent)
		applyHeaders(req.Header, defaultBrowserHeaders(quotaSyncDefaultOrigin, quotaSyncDefaultReferer))
		return req, nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("request GetUserStatus failed: %w", err)
	}
	defer resp.Body.Close()

	payload, err := ioReadAll(resp)
	if err != nil {
		return nil, nil, fmt.Errorf("read GetUserStatus response failed: %w", err)
	}
	if !isSuccessStatus(resp.StatusCode) {
		return nil, nil, fmt.Errorf("GetUserStatus failed(%d)", resp.StatusCode)
	}

	return resp.Header.Clone(), payload, nil
}

func buildQuotaSyncTargetURL(tenantAddress string) (string, error) {
	tenantAddress = strings.TrimSpace(tenantAddress)
	if tenantAddress == "" {
		return "", fmt.Errorf("tenant address is empty")
	}
	if !strings.HasPrefix(tenantAddress, "http://") && !strings.HasPrefix(tenantAddress, "https://") {
		tenantAddress = "https://" + tenantAddress
	}

	baseURL, err := url.Parse(tenantAddress)
	if err != nil {
		return "", fmt.Errorf("invalid tenant address: %w", err)
	}

	basePath := strings.TrimSuffix(baseURL.Path, "/")
	targetPath := quotaSyncRequestPath
	if basePath != "" {
		targetPath = basePath + quotaSyncRequestPath
	}

	return (&url.URL{
		Scheme: baseURL.Scheme,
		Host:   baseURL.Host,
		Path:   targetPath,
	}).String(), nil
}

func newTokenQuotaHTTPClient(token *database.Token, timeout time.Duration) (*http.Client, error) {
	client := newExternalHTTPClient(timeout)
	if token == nil || token.ProxyURL == nil || strings.TrimSpace(*token.ProxyURL) == "" {
		return client, nil
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		return client, nil
	}
	proxyURL, err := url.Parse(strings.TrimSpace(*token.ProxyURL))
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %w", err)
	}
	transport.Proxy = http.ProxyURL(proxyURL)
	return client, nil
}
