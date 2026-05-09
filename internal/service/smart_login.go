package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	FirebaseAPIKey          = "AIzaSyDsOl-1XpT5err0Tcnx8FFod1H8gVGIycY"
	DevinAuthBaseURL        = "https://windsurf.com/_devin-auth"
	WindsurfBackendURL      = "https://web-backend.windsurf.com"
	DevinAppAuthBaseURL     = "https://app.devin.ai/api/auth1"
	WindsurfRegisterURL     = "https://register.windsurf.com"
	DefaultWindsurfAPIURL   = "https://server.codeium.com"
	defaultBrowserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36"
)

// LoginMethodSniffResult 智能识别登录结果。
type LoginMethodSniffResult struct {
	Recommended string `json:"recommended"`
	Reason      string `json:"reason"`

	UserExists         bool   `json:"user_exists"`
	IsMigrated         bool   `json:"is_migrated"`
	HasPassword        bool   `json:"has_password"`
	RedirectURL        string `json:"redirect_url"`
	DisallowEnterprise bool   `json:"disallow_enterprise"`

	DevinConnections *json.RawMessage `json:"devin_connections"`
	DevinMethod      string           `json:"devin_method"`
	DevinHasPassword *bool            `json:"devin_has_password"`
	HasSSOConnection bool             `json:"has_sso_connection"`

	DevinNativeConnections *json.RawMessage `json:"devin_native_connections"`
	DevinNativeMethod      string           `json:"devin_native_method"`
	DevinNativeHasPassword *bool            `json:"devin_native_has_password"`
}

// CheckUserLoginMethodResult Firebase 侧登录方式检查结果。
type CheckUserLoginMethodResult struct {
	RedirectURL                 string `json:"redirect_url"`
	DisallowEnterpriseUserLogin bool   `json:"disallow_enterprise_user_login"`
	UserExists                  bool   `json:"user_exists"`
	IsMigrated                  bool   `json:"is_migrated"`
	HasPassword                 bool   `json:"has_password"`
}

// ConnectionsResponse 连接方式响应。
type ConnectionsResponse struct {
	Raw json.RawMessage `json:"-"`
}

// SmartLoginService 智能识别登录服务。
type SmartLoginService struct {
	httpClient *http.Client
}

func NewSmartLoginService() *SmartLoginService {
	return &SmartLoginService{
		httpClient: newExternalHTTPClient(30 * time.Second),
	}
}

func (s *SmartLoginService) SniffLoginMethod(email string) (*LoginMethodSniffResult, error) {
	ws, err := s.checkUserLoginMethod(email)
	if err != nil {
		return nil, err
	}

	bridgeConn, _ := s.checkConnections(email)
	nativeConn, _ := s.devinAppCheckConnections(email)

	devinMethod, devinHasPassword, hasSSOConnection := s.extractConnectionFields(bridgeConn)
	devinNativeMethod, devinNativeHasPassword, _ := s.extractConnectionFields(nativeConn)
	recommended, reason := s.decideLoginMethod(ws, devinMethod, devinHasPassword, hasSSOConnection, devinNativeMethod, devinNativeHasPassword)

	var bridgeRaw, nativeRaw *json.RawMessage
	if bridgeConn != nil {
		bridgeRaw = &bridgeConn.Raw
	}
	if nativeConn != nil {
		nativeRaw = &nativeConn.Raw
	}

	return &LoginMethodSniffResult{
		Recommended:            recommended,
		Reason:                 reason,
		UserExists:             ws.UserExists,
		IsMigrated:             ws.IsMigrated,
		HasPassword:            ws.HasPassword,
		RedirectURL:            ws.RedirectURL,
		DisallowEnterprise:     ws.DisallowEnterpriseUserLogin,
		DevinConnections:       bridgeRaw,
		DevinMethod:            devinMethod,
		DevinHasPassword:       devinHasPassword,
		HasSSOConnection:       hasSSOConnection,
		DevinNativeConnections: nativeRaw,
		DevinNativeMethod:      devinNativeMethod,
		DevinNativeHasPassword: devinNativeHasPassword,
	}, nil
}

func (s *SmartLoginService) checkUserLoginMethod(email string) (*CheckUserLoginMethodResult, error) {
	url := fmt.Sprintf("%s/exa.seat_management_pb.SeatManagementService/CheckUserLoginMethod", WindsurfBackendURL)
	body := encodeProtoStringField(1, email)

	resp, err := doRequestWithRetry(s.httpClient, 3, func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/proto")
		req.Header.Set("Connect-Protocol-Version", "1")
		applyHeaders(req.Header, defaultBrowserHeaders("https://windsurf.com", "https://windsurf.com/account/login"))
		return req, nil
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyText, _ := readResponseBody(resp)
		return nil, fmt.Errorf("CheckUserLoginMethod failed(%d): %s", resp.StatusCode, bodyText)
	}

	payload, err := ioReadAll(resp)
	if err != nil {
		return nil, err
	}
	return parseCheckUserLoginMethodResponse(payload)
}

func (s *SmartLoginService) checkConnections(email string) (*ConnectionsResponse, error) {
	return s.postConnectionsRequest(
		fmt.Sprintf("%s/connections", DevinAuthBaseURL),
		map[string]string{"product": "Windsurf", "email": email},
		"https://windsurf.com",
		"https://windsurf.com/account/login",
	)
}

func (s *SmartLoginService) devinAppCheckConnections(email string) (*ConnectionsResponse, error) {
	return s.postConnectionsRequest(
		fmt.Sprintf("%s/connections", DevinAppAuthBaseURL),
		map[string]string{"email": email},
		"https://app.devin.ai",
		"https://app.devin.ai/",
	)
}

func (s *SmartLoginService) postConnectionsRequest(url string, payload map[string]string, origin, referer string) (*ConnectionsResponse, error) {
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := doRequestWithRetry(s.httpClient, 2, func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBody))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		applyHeaders(req.Header, defaultBrowserHeaders(origin, referer))
		return req, nil
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyText, _ := readResponseBody(resp)
		return nil, fmt.Errorf("connections failed(%d): %s", resp.StatusCode, bodyText)
	}

	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	return &ConnectionsResponse{Raw: raw}, nil
}

func (s *SmartLoginService) extractConnectionFields(conn *ConnectionsResponse) (method string, hasPassword *bool, hasSSOConnection bool) {
	if conn == nil {
		return "", nil, false
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(conn.Raw, &raw); err != nil {
		return "", nil, false
	}

	authMethod, ok := raw["auth_method"].(map[string]interface{})
	if !ok {
		authMethod = raw
	}
	if m, ok := authMethod["method"].(string); ok {
		method = m
	}
	if hp, ok := authMethod["has_password"].(bool); ok {
		hasPassword = boolPtr(hp)
	}
	if sso, ok := authMethod["sso_connections"].([]interface{}); ok {
		hasSSOConnection = len(sso) > 0
	}
	return method, hasPassword, hasSSOConnection
}

func (s *SmartLoginService) decideLoginMethod(
	ws *CheckUserLoginMethodResult,
	devinMethod string,
	devinHasPassword *bool,
	hasSSOConnection bool,
	devinNativeMethod string,
	devinNativeHasPassword *bool,
) (string, string) {
	if ws.DisallowEnterpriseUserLogin {
		return "blocked", "企业用户被限制普通登录，必须走 Devin Enterprise"
	}
	if hasSSOConnection && !ws.IsMigrated {
		return "sso", "该邮箱绑定企业 SSO，请在浏览器完成 SSO 登录后再导入"
	}
	if ws.UserExists && !ws.IsMigrated {
		if ws.HasPassword {
			return "firebase", "老账号已设密码，走 Firebase 邮箱密码登录"
		}
		return "no_password", "老账号未设密码（仅 Google/GitHub），请先使用 OAuth 或重置密码"
	}
	if ws.IsMigrated || devinMethod == "auth1" || (devinHasPassword != nil && devinMethod == "") {
		return "devin", "已迁移或 Auth1 账号，走 Windsurf/Devin 账密登录"
	}
	if devinNativeMethod == "auth1" {
		if devinNativeHasPassword != nil && *devinNativeHasPassword {
			return "devin_native", "Devin 原生账号，走 app.devin.ai 密码登录"
		}
		return "devin_native_no_password", "Devin 原生账号未设密码，请改用邮箱验证码登录"
	}
	if !ws.UserExists && (devinMethod == "" || devinMethod == "not_found") && (devinNativeMethod == "" || devinNativeMethod == "not_found") {
		return "not_found", "该邮箱尚未注册，请先完成注册"
	}
	return "not_found", "无法自动判定登录方式，请手动确认账号类型"
}

func ioReadAll(resp *http.Response) ([]byte, error) {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(resp.Body)
	return buf.Bytes(), err
}
