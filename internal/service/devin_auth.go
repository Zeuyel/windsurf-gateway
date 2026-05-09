package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// DevinPasswordLoginRequest Devin 密码登录请求。
type DevinPasswordLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// DevinPasswordLoginResponse Devin 密码登录响应。
type DevinPasswordLoginResponse struct {
	Token  string `json:"token"`
	UserID string `json:"user_id,omitempty"`
	Email  string `json:"email,omitempty"`
}

// DevinWindsurfPostAuthResponse Devin Windsurf PostAuth 响应。
type DevinWindsurfPostAuthResponse struct {
	SessionToken string     `json:"session_token"`
	AuthToken    string     `json:"auth1_token,omitempty"`
	AccountID    string     `json:"account_id,omitempty"`
	PrimaryOrgID string     `json:"primary_org_id,omitempty"`
	Orgs         []DevinOrg `json:"orgs,omitempty"`
}

// DevinOrg Devin 组织信息。
type DevinOrg struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// DevinLoginResult Devin 登录结果。
type DevinLoginResult struct {
	SessionToken         string     `json:"session_token"`
	AuthToken            string     `json:"auth1_token"`
	AccountID            string     `json:"account_id,omitempty"`
	PrimaryOrgID         string     `json:"primary_org_id,omitempty"`
	Email                string     `json:"email,omitempty"`
	Orgs                 []DevinOrg `json:"orgs,omitempty"`
	RequiresOrgSelection bool       `json:"requires_org_selection"`
}

// DevinAuthService Devin 认证服务。
type DevinAuthService struct {
	httpClient *http.Client
}

func NewDevinAuthService() *DevinAuthService {
	return &DevinAuthService{httpClient: newExternalHTTPClient(30 * time.Second)}
}

func (s *DevinAuthService) PasswordLogin(email, password string) (*DevinPasswordLoginResponse, error) {
	return s.passwordLoginWithBaseURL(DevinAuthBaseURL, email, password, "https://windsurf.com", "https://windsurf.com/account/login")
}

func (s *DevinAuthService) NativePasswordLogin(email, password string) (*DevinPasswordLoginResponse, error) {
	return s.passwordLoginWithBaseURL(DevinAppAuthBaseURL, email, password, "https://app.devin.ai", "https://app.devin.ai/")
}

func (s *DevinAuthService) passwordLoginWithBaseURL(baseURL, email, password, origin, referer string) (*DevinPasswordLoginResponse, error) {
	url := fmt.Sprintf("%s/password/login", baseURL)
	request := DevinPasswordLoginRequest{Email: email, Password: password}
	jsonBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	applyHeaders(req.Header, defaultBrowserHeaders(origin, referer))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if !isSuccessStatus(resp.StatusCode) {
		body, _ := readResponseBody(resp)
		return nil, parseDevinError(resp.StatusCode, body)
	}

	var response DevinPasswordLoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}
	if strings.TrimSpace(response.Token) == "" {
		return nil, fmt.Errorf("Devin 登录响应未返回 auth1 token")
	}
	return &response, nil
}

func (s *DevinAuthService) WindsurfPostAuth(authToken, orgID string) (*DevinWindsurfPostAuthResponse, error) {
	url := fmt.Sprintf("%s/exa.seat_management_pb.SeatManagementService/WindsurfPostAuth", WindsurfBackendURL)
	body := encodeProtoStringField(1, authToken)
	if strings.TrimSpace(orgID) != "" {
		body = append(body, encodeProtoStringField(2, strings.TrimSpace(orgID))...)
	}

	resp, err := doRequestWithRetry(s.httpClient, 3, func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("create request failed: %w", err)
		}
		req.Header.Set("Content-Type", "application/proto")
		req.Header.Set("Connect-Protocol-Version", "1")
		req.Header.Set("X-Devin-Auth1-Token", authToken)
		applyHeaders(req.Header, defaultBrowserHeaders("https://windsurf.com", "https://windsurf.com/account/login"))
		req.Header.Set("Sec-Fetch-Site", "same-site")
		return req, nil
	})
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	payload, err := ioReadAll(resp)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}
	if !isSuccessStatus(resp.StatusCode) {
		return nil, fmt.Errorf("WindsurfPostAuth failed(%d): %s", resp.StatusCode, string(payload))
	}

	result, err := parseWindsurfPostAuthResponse(payload)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(result.AuthToken) == "" {
		result.AuthToken = authToken
	}
	return result, nil
}

func (s *DevinAuthService) LoginWithPassword(email, password, orgID string) (*DevinLoginResult, error) {
	loginResp, err := s.PasswordLogin(email, password)
	if err != nil {
		return nil, err
	}
	return s.postAuthToLoginResult(loginResp, orgID)
}

func (s *DevinAuthService) LoginNativeWithPassword(email, password, orgID string) (*DevinLoginResult, error) {
	loginResp, err := s.NativePasswordLogin(email, password)
	if err != nil {
		return nil, err
	}
	return s.postAuthToLoginResult(loginResp, orgID)
}

func (s *DevinAuthService) postAuthToLoginResult(loginResp *DevinPasswordLoginResponse, orgID string) (*DevinLoginResult, error) {
	const retryCount = 3
	const retryDelay = 1500 * time.Millisecond

	var postAuthResp *DevinWindsurfPostAuthResponse
	var err error
	for attempt := 0; attempt <= retryCount; attempt++ {
		postAuthResp, err = s.WindsurfPostAuth(loginResp.Token, orgID)
		if err == nil {
			break
		}
		if attempt == retryCount || !isPostAuthOrgSyncPending(err) {
			return nil, err
		}
		time.Sleep(time.Duration(attempt+1) * retryDelay)
	}

	requiresOrgSelection := strings.TrimSpace(orgID) == "" && len(postAuthResp.Orgs) > 1
	return &DevinLoginResult{
		SessionToken:         postAuthResp.SessionToken,
		AuthToken:            coalesceString(postAuthResp.AuthToken, loginResp.Token),
		AccountID:            coalesceString(postAuthResp.AccountID, loginResp.UserID),
		PrimaryOrgID:         postAuthResp.PrimaryOrgID,
		Email:                loginResp.Email,
		Orgs:                 postAuthResp.Orgs,
		RequiresOrgSelection: requiresOrgSelection,
	}, nil
}

func parseWindsurfPostAuthResponse(data []byte) (*DevinWindsurfPostAuthResponse, error) {
	result := &DevinWindsurfPostAuthResponse{}
	idx := 0
	for idx < len(data) {
		tag, consumed, ok := decodeVarint(data, idx)
		if !ok {
			return nil, fmt.Errorf("decode WindsurfPostAuth tag failed")
		}
		idx += consumed

		fieldNumber := int(tag >> 3)
		wireType := int(tag & 0x7)
		switch wireType {
		case 0:
			_, width, ok := decodeVarint(data, idx)
			if !ok {
				return nil, fmt.Errorf("decode WindsurfPostAuth field %d failed", fieldNumber)
			}
			idx += width
		case 2:
			length, width, ok := decodeVarint(data, idx)
			if !ok {
				return nil, fmt.Errorf("decode WindsurfPostAuth field %d length failed", fieldNumber)
			}
			idx += width
			end := idx + int(length)
			if end > len(data) {
				return nil, fmt.Errorf("WindsurfPostAuth field %d exceeds payload", fieldNumber)
			}
			payload := data[idx:end]
			switch fieldNumber {
			case 1:
				result.SessionToken = string(payload)
			case 2:
				if org, ok := parseWindsurfOrg(payload); ok {
					result.Orgs = append(result.Orgs, org)
				}
			case 3:
				result.AuthToken = string(payload)
			case 4:
				result.AccountID = string(payload)
			case 5:
				result.PrimaryOrgID = string(payload)
			}
			idx = end
		case 1:
			idx += 8
		case 5:
			idx += 4
		default:
			return nil, fmt.Errorf("unsupported WindsurfPostAuth wire type %d", wireType)
		}
		if idx > len(data) {
			return nil, fmt.Errorf("WindsurfPostAuth parse overflow")
		}
	}

	if strings.TrimSpace(result.SessionToken) == "" {
		return nil, fmt.Errorf("WindsurfPostAuth response missing session_token")
	}
	return result, nil
}

func parseWindsurfOrg(data []byte) (DevinOrg, bool) {
	org := DevinOrg{}
	idx := 0
	for idx < len(data) {
		tag, consumed, ok := decodeVarint(data, idx)
		if !ok {
			return DevinOrg{}, false
		}
		idx += consumed

		fieldNumber := int(tag >> 3)
		wireType := int(tag & 0x7)
		switch wireType {
		case 2:
			length, width, ok := decodeVarint(data, idx)
			if !ok {
				return DevinOrg{}, false
			}
			idx += width
			end := idx + int(length)
			if end > len(data) {
				return DevinOrg{}, false
			}
			payload := string(data[idx:end])
			if fieldNumber == 1 {
				org.ID = payload
			} else if fieldNumber == 2 {
				org.Name = payload
			}
			idx = end
		case 0:
			_, width, ok := decodeVarint(data, idx)
			if !ok {
				return DevinOrg{}, false
			}
			idx += width
		case 1:
			idx += 8
		case 5:
			idx += 4
		default:
			return DevinOrg{}, false
		}
	}
	return org, strings.TrimSpace(org.ID) != ""
}

func isPostAuthOrgSyncPending(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "404") && (strings.Contains(message, "no_eligible_organizations") || strings.Contains(message, "not_found")) {
		return true
	}
	return strings.Contains(message, "org sync") || strings.Contains(message, "organization") && strings.Contains(message, "not found")
}

func coalesceString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func parseDevinError(statusCode int, body string) error {
	bodyLower := toLower(body)
	if contains(bodyLower, "invalid") && (contains(bodyLower, "password") || contains(bodyLower, "credentials")) {
		return fmt.Errorf("邮箱或密码错误")
	}
	if contains(bodyLower, "not found") || contains(bodyLower, "no such") {
		return fmt.Errorf("该邮箱未注册 Devin 账号")
	}
	if contains(bodyLower, "disabled") || contains(bodyLower, "suspended") {
		return fmt.Errorf("账号已被禁用或暂停")
	}
	if contains(bodyLower, "verify") && contains(bodyLower, "email") {
		return fmt.Errorf("请先验证邮箱")
	}
	if contains(bodyLower, "too many") || contains(bodyLower, "rate") || statusCode == 429 {
		return fmt.Errorf("尝试次数过多，请稍后再试")
	}
	return fmt.Errorf("Devin 登录失败(%d): %s", statusCode, body)
}
