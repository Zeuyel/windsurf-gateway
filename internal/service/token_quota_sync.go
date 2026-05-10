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
	quotaSyncRequestPath    = "/exa.seat_management_pb.SeatManagementService/GetPlanStatus"
	quotaSyncProtoType      = "application/proto"
	quotaSyncRequestTimeout = 30 * time.Second
	quotaSyncUserAgent      = "Go-http-client/1.1"
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
	headers, payload, err := s.fetchPlanStatusSnapshot(token)
	if err != nil {
		return err
	}
	return s.UpdateQuotaFromGetPlanStatusResponse(token.ID, headers, payload)
}

func (s *TokenService) fetchPlanStatusSnapshot(token *database.Token) (http.Header, []byte, error) {
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

	body := buildPlanStatusRequestBody(token.Token)
	resp, err := doRequestWithRetry(client, 2, func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", quotaSyncProtoType)
		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set("User-Agent", quotaSyncUserAgent)
		return req, nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("request GetPlanStatus failed: %w", err)
	}
	defer resp.Body.Close()

	payload, err := ioReadAll(resp)
	if err != nil {
		return nil, nil, fmt.Errorf("read GetPlanStatus response failed: %w", err)
	}
	if !isSuccessStatus(resp.StatusCode) {
		return nil, nil, fmt.Errorf("GetPlanStatus failed(%d)", resp.StatusCode)
	}
	return resp.Header.Clone(), payload, nil
}

func buildPlanStatusRequestBody(authToken string) []byte {
	body := make([]byte, 0, len(authToken)+8)
	appendProtoStringField(&body, 1, authToken)
	appendProtoBoolField(&body, 2, true)
	return body
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
	return (&url.URL{Scheme: baseURL.Scheme, Host: baseURL.Host, Path: targetPath}).String(), nil
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

func (s *TokenService) clearEmptyQuotaSnapshot(token *database.Token) error {
	if token == nil || !isEmptyQuotaSnapshot(token) {
		return nil
	}
	return s.db.Model(&database.Token{}).Where("id = ?", token.ID).Updates(emptyQuotaSnapshotUpdates()).Error
}

func emptyQuotaSnapshotUpdates() map[string]interface{} {
	return map[string]interface{}{
		"plan_name":                      "",
		"monthly_prompt_credits":         0,
		"monthly_flow_credits":           0,
		"monthly_flex_credits":           0,
		"available_prompt_credits":       0,
		"used_prompt_credits":            0,
		"available_flow_credits":         0,
		"used_flow_credits":              0,
		"available_flex_credits":         0,
		"used_flex_credits":              0,
		"daily_quota_remaining_percent":  0,
		"weekly_quota_remaining_percent": 0,
		"hide_daily_quota":               false,
		"hide_weekly_quota":              false,
		"daily_quota_reset_at":           nil,
		"weekly_quota_reset_at":          nil,
		"quota_updated_at":               nil,
	}
}

func isEmptyQuotaSnapshot(token *database.Token) bool {
	if token == nil || token.QuotaUpdatedAt == nil {
		return false
	}
	return token.PlanName == "" &&
		token.MonthlyPromptCredits == 0 &&
		token.MonthlyFlowCredits == 0 &&
		token.MonthlyFlexCredits == 0 &&
		token.AvailablePromptCredits == 0 &&
		token.UsedPromptCredits == 0 &&
		token.AvailableFlowCredits == 0 &&
		token.UsedFlowCredits == 0 &&
		token.AvailableFlexCredits == 0 &&
		token.UsedFlexCredits == 0 &&
		token.DailyQuotaRemainingPercent == 0 &&
		token.WeeklyQuotaRemainingPercent == 0 &&
		!token.HideDailyQuota &&
		!token.HideWeeklyQuota &&
		token.DailyQuotaResetAt == nil &&
		token.WeeklyQuotaResetAt == nil
}
