package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// FirebaseSignInRequest Firebase 登录请求
type FirebaseSignInRequest struct {
	ReturnSecureToken bool   `json:"returnSecureToken"`
	Email             string `json:"email"`
	Password          string `json:"password"`
	ClientType        string `json:"clientType"`
}

// FirebaseSignInResponse Firebase 登录响应
type FirebaseSignInResponse struct {
	IDToken      string `json:"idToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    string `json:"expiresIn"`
	LocalID      string `json:"localId"`
	Email        string `json:"email"`
	DisplayName  string `json:"displayName"`
}

// FirebaseRefreshTokenRequest Firebase 刷新 Token 请求
type FirebaseRefreshTokenRequest struct {
	GrantType    string `json:"grant_type"`
	RefreshToken string `json:"refresh_token"`
}

// FirebaseRefreshTokenResponse Firebase 刷新 Token 响应
type FirebaseRefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    string `json:"expires_in"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	UserID       string `json:"user_id"`
	ProjectID    string `json:"project_id"`
}

// FirebaseAuthService Firebase 认证服务
type FirebaseAuthService struct {
	httpClient *http.Client
}

func NewFirebaseAuthService() *FirebaseAuthService {
	return &FirebaseAuthService{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SignIn 使用邮箱密码登录
func (s *FirebaseAuthService) SignIn(email, password string) (*FirebaseSignInResponse, error) {
	url := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=%s", FirebaseAPIKey)

	request := FirebaseSignInRequest{
		ReturnSecureToken: true,
		Email:             email,
		Password:          password,
		ClientType:        "CLIENT_TYPE_WEB",
	}

	jsonBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Origin", "https://windsurf.com")
	req.Header.Set("Referer", "https://windsurf.com/")
	req.Header.Set("X-Client-Version", "Chrome/JsCore/11.0.0/FirebaseCore-web")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if !isSuccessStatus(resp.StatusCode) {
		body, _ := readResponseBody(resp)
		return nil, parseFirebaseError(resp.StatusCode, body)
	}

	var response FirebaseSignInResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &response, nil
}

// RefreshToken 刷新 Token
func (s *FirebaseAuthService) RefreshToken(refreshToken string) (*FirebaseRefreshTokenResponse, error) {
	url := fmt.Sprintf("https://securetoken.googleapis.com/v1/token?key=%s", FirebaseAPIKey)

	body := fmt.Sprintf("grant_type=refresh_token&refresh_token=%s", refreshToken)

	req, err := http.NewRequest("POST", url, bytes.NewReader([]byte(body)))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Origin", "https://windsurf.com")
	req.Header.Set("Referer", "https://windsurf.com/")
	req.Header.Set("X-Client-Version", "Chrome/JsCore/11.0.0/FirebaseCore-web")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if !isSuccessStatus(resp.StatusCode) {
		body, _ := readResponseBody(resp)
		return nil, parseFirebaseError(resp.StatusCode, body)
	}

	var response FirebaseRefreshTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &response, nil
}

// GetAccountInfo 获取账号信息
func (s *FirebaseAuthService) GetAccountInfo(idToken string) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:lookup?key=%s", FirebaseAPIKey)

	body := map[string]string{
		"idToken": idToken,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Version", "Chrome/JsCore/11.0.0/FirebaseCore-web")
	req.Header.Set("Origin", "https://windsurf.com")
	req.Header.Set("Referer", "https://windsurf.com/")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if !isSuccessStatus(resp.StatusCode) {
		body, _ := readResponseBody(resp)
		return nil, parseFirebaseError(resp.StatusCode, body)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return result, nil
}

// parseFirebaseError 解析 Firebase 错误
func parseFirebaseError(statusCode int, body string) error {
	bodyLower := toLower(body)

	if contains(bodyLower, "too_many_attempts_try_later") {
		return fmt.Errorf("登录尝试次数过多，请15-30分钟后再试")
	}
	if contains(bodyLower, "invalid_login_credentials") || contains(bodyLower, "invalid_password") {
		return fmt.Errorf("邮箱或密码错误，请检查后重试")
	}
	if contains(bodyLower, "email_not_found") {
		return fmt.Errorf("该邮箱未注册")
	}
	if contains(bodyLower, "user_disabled") {
		return fmt.Errorf("该账号已被禁用")
	}
	if contains(bodyLower, "token_expired") || contains(bodyLower, "invalid_refresh_token") {
		return fmt.Errorf("token 已过期，请重新登录")
	}

	return fmt.Errorf("Firebase 认证失败(%d): %s", statusCode, body)
}

// Helper functions
func isSuccessStatus(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}

func readResponseBody(resp *http.Response) (string, error) {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(resp.Body)
	return buf.String(), err
}

func toLower(s string) string {
	lower := ""
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			lower += string(c + 32)
		} else {
			lower += string(c)
		}
	}
	return lower
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
