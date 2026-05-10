package patcher

import "time"

type Mode string

const (
	ModeConfig    Mode = "config"
	ModeExtension Mode = "extension"
	ModeAll       Mode = "all"
)

const (
	DefaultAPIURL                  = "https://server.codeium.com"
	ConfigKey                      = "codeium.apiServerUrl"
	RegisterConfigKey              = "codeium.registerApiServerUrl"
	DefaultRegisterURL             = "https://register.windsurf.com"
	LegacyGatewayPlaceholderAPIKey = "sk-ws-01-gateway-placeholder"
	GatewayPlaceholderPrefix       = "sk-ws-01-client-"
	ACPConfigKey                   = "windsurf.acp.enabledAgents"
	PendingAPIKeyMigrationStateKey = "windsurf.pendingApiKeyMigration"
	PatchStateRelativePath         = "User/globalStorage/windsurf-gateway-patch.json"
)

type Environment struct {
	ConfigDir     string `json:"config_dir"`
	InstallDir    string `json:"install_dir"`
	SettingsPath  string `json:"settings_path"`
	GlobalState   string `json:"global_state_path"`
	ExtensionPath string `json:"extension_path"`
	PatchState    string `json:"patch_state_path"`
	BackupRoot    string `json:"backup_root"`
}

type PatchState struct {
	PlaceholderAPIKey string `json:"placeholderApiKey"`
	CreatedAt         string `json:"createdAt,omitempty"`
}

type SettingsInfo struct {
	Exists             bool   `json:"exists"`
	GatewayURL         string `json:"gateway_url,omitempty"`
	RegisterGatewayURL string `json:"register_gateway_url,omitempty"`
	DevinCloudDisabled bool   `json:"devin_cloud_disabled"`
}

type GlobalStateInfo struct {
	Exists            bool   `json:"exists"`
	GatewayURLKeys    int    `json:"gateway_url_keys"`
	AuthTokenSummary  string `json:"auth_token_summary,omitempty"`
	AuthMode          string `json:"auth_mode,omitempty"`
	OnboardingPatched bool   `json:"onboarding_patched"`
	EducationPatched  bool   `json:"education_patched"`
	ReadError         string `json:"read_error,omitempty"`
}

type ExtensionInfo struct {
	Exists                  bool   `json:"exists"`
	ContainsDefaultAPI      bool   `json:"contains_default_api"`
	ContainsDefaultRegister bool   `json:"contains_default_register"`
	HasAuthFallback         bool   `json:"has_auth_fallback"`
	HasUserStatusFallback   bool   `json:"has_user_status_fallback"`
	AuthTokenSummary        string `json:"auth_token_summary,omitempty"`
	AuthMode                string `json:"auth_mode,omitempty"`
	ReadError               string `json:"read_error,omitempty"`
}

type DetectResult struct {
	Environment           Environment     `json:"environment"`
	PatchStateExists      bool            `json:"patch_state_exists"`
	PatchStatePlaceholder string          `json:"patch_state_placeholder,omitempty"`
	PatchStateMode        string          `json:"patch_state_mode,omitempty"`
	Settings              SettingsInfo    `json:"settings"`
	GlobalState           GlobalStateInfo `json:"global_state"`
	Extension             ExtensionInfo   `json:"extension"`
	Messages              []string        `json:"messages,omitempty"`
	DetectedAt            time.Time       `json:"detected_at"`
}

type ApplyOptions struct {
	ConfigDir          string `json:"config_dir,omitempty"`
	InstallDir         string `json:"install_dir,omitempty"`
	GatewayURL         string `json:"gateway_url"`
	RegisterGatewayURL string `json:"register_gateway_url,omitempty"`
	AuthToken          string `json:"auth_token,omitempty"`
	Mode               Mode   `json:"mode,omitempty"`
}

type ApplyResult struct {
	BackupDir             string       `json:"backup_dir,omitempty"`
	EffectiveTokenSummary string       `json:"effective_token_summary,omitempty"`
	EffectiveAuthMode     string       `json:"effective_auth_mode,omitempty"`
	Detect                DetectResult `json:"detect"`
	Messages              []string     `json:"messages,omitempty"`
}

type RestoreResult struct {
	RestoredFrom string       `json:"restored_from"`
	Detect       DetectResult `json:"detect"`
	Messages     []string     `json:"messages,omitempty"`
}
