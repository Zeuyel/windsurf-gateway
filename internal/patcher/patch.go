package patcher

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"windsurf-gateway/internal/gatewayuser"
)

var (
	authFallbackRegex          = regexp.MustCompile(`await i\.authentication\.getSession\(n\.WindsurfExtensionMetadata\.getInstance\(\)\.authProviderId,\[[\s\S]*?\],e\)`)
	authFallbackBlockRegex     = regexp.MustCompile(`\?\?\{id:"windsurf-gateway",accessToken:"([^"]+)",account:\{label:"Gateway",id:"windsurf-gateway"\},scopes:\[\]\}`)
	userStatusFallbackSentinel = `allowedCommandModelConfigsProtoBinaryBase64:[],userStatusProtoBinaryBase64:""}`
	userStatusFallbackRegex    = regexp.MustCompile(`([A-Za-z_$][\w$]*)\.StatusBar\.getInstance\(\)\.setAuthStatus\(!1\),([A-Za-z_$][\w$]*)\.windsurfAuth\.setAuthStatus\(null\),\(await\(0,([A-Za-z_$][\w$]*)\.getAuthSession\)\(\)\)\?\.accessToken===([A-Za-z_$][\w$]*)\|\|([A-Za-z_$][\w$]*)\.clearAuthentication\(\),!1`)
)

func Detect(configDir, installDir string) (DetectResult, error) {
	env := ResolveEnvironment(configDir, installDir)
	result := DetectResult{
		Environment: env,
		DetectedAt:  time.Now(),
	}

	state, err := loadPatchState(env.PatchState)
	if err == nil && strings.TrimSpace(state.PlaceholderAPIKey) != "" {
		result.PatchStateExists = true
		result.PatchStatePlaceholder = summarizeToken(state.PlaceholderAPIKey)
		result.PatchStateMode = classifyAuthMode(state.PlaceholderAPIKey)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		result.Messages = append(result.Messages, fmt.Sprintf("patch state unreadable: %v", err))
	}

	result.Settings = detectSettings(env.SettingsPath)
	result.GlobalState = detectGlobalState(env.GlobalState)
	result.Extension = detectExtension(env.ExtensionPath)
	return result, nil
}

func Apply(options ApplyOptions) (ApplyResult, error) {
	options.Mode = NormalizeMode(options.Mode)
	if strings.TrimSpace(options.GatewayURL) == "" {
		return ApplyResult{}, fmt.Errorf("gateway_url is required")
	}
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(options.GatewayURL)), "http://") && !strings.HasPrefix(strings.ToLower(strings.TrimSpace(options.GatewayURL)), "https://") {
		return ApplyResult{}, fmt.Errorf("gateway_url must start with http:// or https://")
	}
	authToken := strings.TrimSpace(options.AuthToken)
	if !gatewayuser.IsToken(authToken) {
		return ApplyResult{}, fmt.Errorf("auth_token is required and must start with %s", gatewayuser.TokenPrefix)
	}

	env := ResolveEnvironment(options.ConfigDir, options.InstallDir)
	backup := newBackupSession(env.BackupRoot)
	_ = backup.Backup(env.PatchState)
	effectiveToken := authToken
	effectiveMode := "gateway-user"

	messages := make([]string, 0, 8)
	if options.Mode == ModeConfig || options.Mode == ModeAll {
		if err := backup.Backup(env.SettingsPath); err != nil {
			return ApplyResult{}, err
		}
		if err := patchSettings(env.SettingsPath, options.GatewayURL, options.RegisterGatewayURL); err != nil {
			return ApplyResult{}, err
		}
		messages = append(messages, "settings.json updated")

		if err := backup.Backup(env.GlobalState); err != nil {
			return ApplyResult{}, err
		}
		patched, err := patchGlobalState(env.GlobalState, options.GatewayURL, effectiveToken)
		if err != nil {
			return ApplyResult{}, err
		}
		if patched {
			messages = append(messages, "state.vscdb updated")
		} else {
			messages = append(messages, "state.vscdb not found; skipped")
		}
	}

	if options.Mode == ModeExtension || options.Mode == ModeAll {
		if err := backup.Backup(env.ExtensionPath); err != nil {
			return ApplyResult{}, err
		}
		patched, err := patchExtension(env.ExtensionPath, options.GatewayURL, options.RegisterGatewayURL, effectiveToken)
		if err != nil {
			return ApplyResult{}, err
		}
		if patched {
			messages = append(messages, "extension.js updated")
		} else {
			messages = append(messages, "extension.js not found or already patched")
		}
	}

	detect, err := Detect(options.ConfigDir, options.InstallDir)
	if err != nil {
		return ApplyResult{}, err
	}
	result := ApplyResult{
		BackupDir:             backup.dir,
		EffectiveTokenSummary: summarizeToken(effectiveToken),
		EffectiveAuthMode:     effectiveMode,
		Detect:                detect,
		Messages:              messages,
	}
	if len(backup.copied) == 0 {
		result.BackupDir = ""
	}
	return result, nil
}

func Restore(configDir, installDir string) (RestoreResult, error) {
	env := ResolveEnvironment(configDir, installDir)
	latest, err := latestBackupDir(env.BackupRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return RestoreResult{}, fmt.Errorf("no backup found")
		}
		return RestoreResult{}, err
	}

	messages := make([]string, 0, 4)
	for _, item := range []struct {
		name   string
		target string
	}{
		{name: filepath.Base(env.SettingsPath), target: env.SettingsPath},
		{name: filepath.Base(env.ExtensionPath), target: env.ExtensionPath},
		{name: filepath.Base(env.GlobalState), target: env.GlobalState},
		{name: filepath.Base(env.PatchState), target: env.PatchState},
	} {
		source := filepath.Join(latest, item.name)
		data, readErr := os.ReadFile(source)
		if readErr != nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(item.target), 0o755); err != nil {
			return RestoreResult{}, err
		}
		if err := os.WriteFile(item.target, data, 0o644); err != nil {
			return RestoreResult{}, err
		}
		messages = append(messages, fmt.Sprintf("restored %s", item.name))
	}

	if cleaned, err := cleanupMockedGlobalState(env.GlobalState); err != nil {
		messages = append(messages, fmt.Sprintf("global state cleanup failed: %v", err))
	} else if cleaned > 0 {
		messages = append(messages, fmt.Sprintf("cleaned %d mocked global-state keys", cleaned))
	}

	detect, err := Detect(configDir, installDir)
	if err != nil {
		return RestoreResult{}, err
	}
	return RestoreResult{RestoredFrom: latest, Detect: detect, Messages: messages}, nil
}

func patchSettings(settingsPath, gatewayURL, registerGatewayURL string) error {
	settings := map[string]any{}
	if data, err := os.ReadFile(settingsPath); err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parse settings.json: %w", err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	settings[ConfigKey] = gatewayURL
	if strings.TrimSpace(registerGatewayURL) != "" {
		settings[RegisterConfigKey] = registerGatewayURL
	}
	enabledAgents := map[string]any{}
	if existing, ok := settings[ACPConfigKey].(map[string]any); ok {
		for key, value := range existing {
			enabledAgents[key] = value
		}
	}
	enabledAgents["devin-cloud"] = false
	settings[ACPConfigKey] = enabledAgents
	return writeJSON(settingsPath, settings)
}

func patchExtension(extensionPath, gatewayURL, registerGatewayURL, effectiveToken string) (bool, error) {
	data, err := os.ReadFile(extensionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	content := string(data)
	before := content
	content = replaceExtensionConstant(content, "DEFAULT_API_SERVER_URL", DefaultAPIURL, gatewayURL)
	if strings.TrimSpace(registerGatewayURL) != "" {
		content = replaceExtensionConstant(content, "DEFAULT_REGISTER_API_SERVER_URL", DefaultRegisterURL, registerGatewayURL)
	}

	fallbackBlock := buildAuthFallbackBlock(effectiveToken)
	if authFallbackBlockRegex.MatchString(content) {
		content = authFallbackBlockRegex.ReplaceAllStringFunc(content, func(string) string {
			return fallbackBlock
		})
	} else {
		content = authFallbackRegex.ReplaceAllStringFunc(content, func(match string) string {
			return match + fallbackBlock
		})
	}
	if !strings.Contains(content, userStatusFallbackSentinel) {
		content = userStatusFallbackRegex.ReplaceAllString(content, `${1}.StatusBar.getInstance().setAuthStatus(!0),${2}.windsurfAuth.setAuthStatus({apiKey:${4},allowedCommandModelConfigsProtoBinaryBase64:[],userStatusProtoBinaryBase64:""}),!0`)
	}

	if content == before {
		return false, nil
	}
	return true, os.WriteFile(extensionPath, []byte(content), 0o644)
}

func buildAuthFallbackBlock(token string) string {
	return fmt.Sprintf(`??{id:"windsurf-gateway",accessToken:"%s",account:{label:"Gateway",id:"windsurf-gateway"},scopes:[]}`, token)
}

func replaceExtensionConstant(content, key, oldValue, newValue string) string {
	quoted := key + `="` + oldValue + `"`
	replaced := key + `="` + newValue + `"`
	escapedQuoted := key + `=\"` + oldValue + `\"`
	escapedReplaced := key + `=\"` + newValue + `\"`
	content = strings.ReplaceAll(content, quoted, replaced)
	content = strings.ReplaceAll(content, escapedQuoted, escapedReplaced)
	return content
}

func detectSettings(settingsPath string) SettingsInfo {
	info := SettingsInfo{}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return info
	}
	info.Exists = true
	var settings map[string]any
	if jsonErr := json.Unmarshal(data, &settings); jsonErr != nil {
		return info
	}
	if gateway, ok := settings[ConfigKey].(string); ok {
		info.GatewayURL = gateway
	}
	if register, ok := settings[RegisterConfigKey].(string); ok {
		info.RegisterGatewayURL = register
	}
	if enabledAgents, ok := settings[ACPConfigKey].(map[string]any); ok {
		if value, ok := enabledAgents["devin-cloud"].(bool); ok {
			info.DevinCloudDisabled = !value
		}
	}
	return info
}

func detectExtension(extensionPath string) ExtensionInfo {
	info := ExtensionInfo{}
	data, err := os.ReadFile(extensionPath)
	if err != nil {
		if !os.IsNotExist(err) {
			info.ReadError = err.Error()
		}
		return info
	}
	info.Exists = true
	content := string(data)
	info.ContainsDefaultAPI = strings.Contains(content, DefaultAPIURL)
	info.ContainsDefaultRegister = strings.Contains(content, DefaultRegisterURL)
	info.HasAuthFallback = authFallbackBlockRegex.MatchString(content)
	info.HasUserStatusFallback = strings.Contains(content, userStatusFallbackSentinel)
	matches := authFallbackBlockRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		info.AuthTokenSummary = summarizeToken(matches[1])
		info.AuthMode = classifyAuthMode(matches[1])
	}
	return info
}
