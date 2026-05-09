package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// WindsurfRegisterUserResponse RegisterUser 返回的主 token 结果。
type WindsurfRegisterUserResponse struct {
	APIKey       string                 `json:"api_key"`
	Name         string                 `json:"name,omitempty"`
	APIServerURL string                 `json:"api_server_url,omitempty"`
	RawData      map[string]interface{} `json:"raw_data,omitempty"`
}

// WindsurfPrimaryAPIKeyResponse Devin session 换取主 token 的结果。
type WindsurfPrimaryAPIKeyResponse struct {
	APIKey string `json:"api_key"`
}

// WindsurfAuthService Windsurf 注册与 token 交换服务。
type WindsurfAuthService struct {
	httpClient *http.Client
}

func NewWindsurfAuthService() *WindsurfAuthService {
	return &WindsurfAuthService{httpClient: newExternalHTTPClient(30 * time.Second)}
}

func (s *WindsurfAuthService) RegisterUser(firebaseIDToken string) (*WindsurfRegisterUserResponse, error) {
	url := fmt.Sprintf("%s/exa.seat_management_pb.SeatManagementService/RegisterUser", WindsurfRegisterURL)
	attempts := []map[string]string{
		{"firebase_id_token": firebaseIDToken},
		{"firebaseIdToken": firebaseIDToken},
	}

	var lastErr error
	for _, payload := range attempts {
		resp, err := s.postRegisterRequest(url, payload)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("RegisterUser failed")
	}
	return nil, lastErr
}

func (s *WindsurfAuthService) postRegisterRequest(url string, payload map[string]string) (*WindsurfRegisterUserResponse, error) {
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	applyHeaders(req.Header, defaultBrowserHeaders("https://windsurf.com", "https://windsurf.com/"))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyText, err := readResponseBody(resp)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}
	if !isSuccessStatus(resp.StatusCode) {
		return nil, fmt.Errorf("RegisterUser failed(%d): %s", resp.StatusCode, bodyText)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(bodyText), &raw); err != nil {
		return nil, fmt.Errorf("decode RegisterUser response failed: %w", err)
	}

	response := &WindsurfRegisterUserResponse{RawData: raw}
	response.APIKey = firstNonEmptyString(raw, "api_key", "apiKey", "token")
	response.Name = firstNonEmptyString(raw, "name")
	response.APIServerURL = firstNonEmptyString(raw, "api_server_url", "apiServerUrl")
	if strings.TrimSpace(response.APIKey) == "" {
		return nil, fmt.Errorf("RegisterUser response missing api key: %s", bodyText)
	}
	return response, nil
}

func (s *WindsurfAuthService) GetPrimaryAPIKeyForDevs(sessionToken string) (*WindsurfPrimaryAPIKeyResponse, error) {
	url := fmt.Sprintf("%s/exa.seat_management_pb.SeatManagementService/GetPrimaryApiKeyForDevsOnly", WindsurfBackendURL)
	body := encodeProtoStringField(1, sessionToken)

	resp, err := doRequestWithRetry(s.httpClient, 3, func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("create request failed: %w", err)
		}
		req.Header.Set("Content-Type", "application/proto")
		req.Header.Set("Connect-Protocol-Version", "1")
		applyHeaders(req.Header, defaultBrowserHeaders("https://windsurf.com", "https://windsurf.com/"))
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
		return nil, fmt.Errorf("GetPrimaryApiKeyForDevsOnly failed(%d): %s", resp.StatusCode, string(payload))
	}

	apiKey, err := parseLengthDelimitedStringField(payload, 1)
	if err != nil {
		return nil, fmt.Errorf("parse primary api key failed: %w", err)
	}
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("GetPrimaryApiKeyForDevsOnly response missing api key")
	}
	return &WindsurfPrimaryAPIKeyResponse{APIKey: apiKey}, nil
}

func firstNonEmptyString(raw map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := raw[key].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
