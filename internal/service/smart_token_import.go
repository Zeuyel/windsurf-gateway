package service

import (
	"fmt"
	"strings"

	"windsurf-gateway/internal/database"
)

// SmartTokenImportRequest 通过邮箱密码导入 backend token 的请求。
type SmartTokenImportRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	OrgID       string `json:"org_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ProxyURL    string `json:"proxy_url"`
	MaxRequests int    `json:"max_requests"`
	Weight      int    `json:"weight"`
	IsShared    bool   `json:"is_shared"`
}

// SmartTokenImportResult 导入结果。
type SmartTokenImportResult struct {
	Recommended          string          `json:"recommended"`
	Reason               string          `json:"reason"`
	BackendTokenKind     string          `json:"backend_token_kind"`
	CreatedToken         *database.Token `json:"created_token,omitempty"`
	RequiresOrgSelection bool            `json:"requires_org_selection,omitempty"`
	Orgs                 []DevinOrg      `json:"orgs,omitempty"`
}

// SmartTokenImportSelectionError 需要用户补选 org。
type SmartTokenImportSelectionError struct {
	Result *SmartTokenImportResult
}

func (e *SmartTokenImportSelectionError) Error() string {
	if e == nil || e.Result == nil || strings.TrimSpace(e.Result.Reason) == "" {
		return "需要选择组织"
	}
	return e.Result.Reason
}

// SmartTokenImportUnsupportedError 当前账号类型不支持直接账密导入。
type SmartTokenImportUnsupportedError struct {
	Result *SmartTokenImportResult
}

func (e *SmartTokenImportUnsupportedError) Error() string {
	if e == nil || e.Result == nil || strings.TrimSpace(e.Result.Reason) == "" {
		return "当前账号类型不支持直接账密导入"
	}
	return e.Result.Reason
}

// SmartTokenImportService 账密换主 token 并导入 token 池。
type SmartTokenImportService struct {
	smart     *SmartLoginService
	firebase  *FirebaseAuthService
	devin     *DevinAuthService
	windsurf  *WindsurfAuthService
	tokenPool *TokenService
}

func NewSmartTokenImportService(
	smart *SmartLoginService,
	firebase *FirebaseAuthService,
	devin *DevinAuthService,
	windsurf *WindsurfAuthService,
	tokenPool *TokenService,
) *SmartTokenImportService {
	return &SmartTokenImportService{
		smart:     smart,
		firebase:  firebase,
		devin:     devin,
		windsurf:  windsurf,
		tokenPool: tokenPool,
	}
}

func (s *SmartTokenImportService) Import(req *SmartTokenImportRequest) (*SmartTokenImportResult, error) {
	email := strings.TrimSpace(req.Email)
	password := strings.TrimSpace(req.Password)
	orgID := strings.TrimSpace(req.OrgID)
	if email == "" || password == "" {
		return nil, fmt.Errorf("email 和 password 不能为空")
	}

	sniff, err := s.smart.SniffLoginMethod(email)
	if err != nil {
		return nil, err
	}

	result := &SmartTokenImportResult{
		Recommended: sniff.Recommended,
		Reason:      sniff.Reason,
	}

	var backendToken string
	var tenantAddress string

	switch sniff.Recommended {
	case "firebase":
		login, err := s.firebase.SignIn(email, password)
		if err != nil {
			return nil, err
		}
		registered, err := s.windsurf.RegisterUser(login.IDToken)
		if err != nil {
			return nil, err
		}
		backendToken = registered.APIKey
		tenantAddress = coalesceString(registered.APIServerURL, DefaultWindsurfAPIURL)
		result.BackendTokenKind = classifyBackendTokenKind(backendToken)
	case "devin":
		login, err := s.devin.LoginWithPassword(email, password, orgID)
		if err != nil {
			return nil, err
		}
		if login.RequiresOrgSelection {
			result.RequiresOrgSelection = true
			result.Orgs = login.Orgs
			return nil, &SmartTokenImportSelectionError{Result: result}
		}
		backendToken = strings.TrimSpace(login.SessionToken)
		if backendToken == "" {
			return nil, fmt.Errorf("Devin WindsurfPostAuth 未返回 session token")
		}
		tenantAddress = DefaultWindsurfAPIURL
		result.BackendTokenKind = classifyBackendTokenKind(backendToken)
	case "devin_native":
		login, err := s.devin.LoginNativeWithPassword(email, password, orgID)
		if err != nil {
			return nil, err
		}
		if login.RequiresOrgSelection {
			result.RequiresOrgSelection = true
			result.Orgs = login.Orgs
			return nil, &SmartTokenImportSelectionError{Result: result}
		}
		backendToken = strings.TrimSpace(login.SessionToken)
		if backendToken == "" {
			return nil, fmt.Errorf("Devin Native WindsurfPostAuth 未返回 session token")
		}
		tenantAddress = DefaultWindsurfAPIURL
		result.BackendTokenKind = classifyBackendTokenKind(backendToken)
	case "sso", "blocked", "no_password", "devin_native_no_password", "not_found":
		result.BackendTokenKind = "unsupported"
		return nil, &SmartTokenImportUnsupportedError{Result: result}
	default:
		result.BackendTokenKind = "unsupported"
		result.Reason = coalesceString(result.Reason, "当前账号类型暂不支持直接导入")
		return nil, &SmartTokenImportUnsupportedError{Result: result}
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = fmt.Sprintf("%s (%s)", email, sniff.Recommended)
	}
	description := strings.TrimSpace(req.Description)
	if description == "" {
		description = fmt.Sprintf("smart login import via %s", sniff.Recommended)
	}
	isShared := req.IsShared
	if !req.IsShared {
		isShared = true
	}

	created, err := s.tokenPool.CreateToken(&TokenCreateRequest{
		Token:         backendToken,
		Name:          name,
		Description:   description,
		TenantAddress: tenantAddress,
		ProxyURL:      strings.TrimSpace(req.ProxyURL),
		MaxRequests:   req.MaxRequests,
		Weight:        req.Weight,
		IsShared:      isShared,
		Email:         email,
	})
	if err != nil {
		return nil, err
	}

	result.CreatedToken = created
	return result, nil
}

func classifyBackendTokenKind(token string) string {
	token = strings.TrimSpace(token)
	switch {
	case strings.HasPrefix(token, "sk-ws-01-"):
		return "sk-ws-01"
	case strings.HasPrefix(token, "devin-session-token$"):
		return "devin-session-token"
	case strings.Count(token, ".") == 2:
		return "jwt"
	case token == "":
		return "unknown"
	default:
		return "api_key"
	}
}
