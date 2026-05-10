package patcher

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

func NormalizeMode(mode Mode) Mode {
	switch mode {
	case ModeConfig, ModeExtension, ModeAll:
		return mode
	default:
		return ModeAll
	}
}

func ResolveEnvironment(configDir, installDir string) Environment {
	configDir = strings.TrimSpace(configDir)
	installDir = strings.TrimSpace(installDir)
	if configDir == "" {
		configDir = defaultConfigDir()
	}
	if installDir == "" {
		installDir = defaultInstallDir()
	}

	settingsPath := filepath.Join(configDir, "User", "settings.json")
	globalStatePath := filepath.Join(configDir, "User", "globalStorage", "state.vscdb")
	extensionPath := filepath.Join(installDir, "extensions", "windsurf", "dist", "extension.js")
	backupRoot := filepath.Join(configDir, "windsurf-gateway-backups")
	patchStatePath := filepath.Join(configDir, filepath.FromSlash(PatchStateRelativePath))

	return Environment{
		ConfigDir:     configDir,
		InstallDir:    installDir,
		SettingsPath:  settingsPath,
		GlobalState:   globalStatePath,
		ExtensionPath: extensionPath,
		PatchState:    patchStatePath,
		BackupRoot:    backupRoot,
	}
}

func defaultConfigDir() string {
	if value := strings.TrimSpace(os.Getenv("WINDSURF_CONFIG_DIR")); value != "" {
		return value
	}
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Windsurf")
	case "windows":
		base := strings.TrimSpace(os.Getenv("APPDATA"))
		if base == "" {
			base = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(base, "Windsurf")
	default:
		return filepath.Join(home, ".config", "Windsurf")
	}
}

func defaultInstallDir() string {
	if value := strings.TrimSpace(os.Getenv("WINDSURF_INSTALL_DIR")); value != "" {
		return value
	}
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return "/Applications/Windsurf.app/Contents/Resources/app"
	case "windows":
		base := strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
		if base == "" {
			base = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(base, "Programs", "Windsurf", "resources", "app")
	default:
		return "/opt/windsurf/resources/app"
	}
}

func loadPatchState(path string) (PatchState, error) {
	var state PatchState
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return PatchState{}, err
	}
	return state, nil
}

func ensurePatchState(path string) (PatchState, error) {
	state, err := loadPatchState(path)
	if err == nil && strings.TrimSpace(state.PlaceholderAPIKey) != "" {
		return state, nil
	}

	state = PatchState{
		PlaceholderAPIKey: generatePlaceholderAPIKey(),
		CreatedAt:         time.Now().Format(time.RFC3339),
	}
	if writeErr := writeJSON(path, state); writeErr != nil {
		return PatchState{}, writeErr
	}
	return state, nil
}

func generatePlaceholderAPIKey() string {
	buf := make([]byte, 18)
	if _, err := rand.Read(buf); err != nil {
		return GatewayPlaceholderPrefix + fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return GatewayPlaceholderPrefix + hex.EncodeToString(buf)
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func summarizeToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if len(token) <= 12 {
		return token
	}
	return token[:8] + "..." + token[len(token)-4:]
}

func classifyAuthMode(token string) string {
	token = strings.TrimSpace(token)
	switch {
	case token == "":
		return "none"
	case strings.HasPrefix(strings.ToLower(token), "ws-"):
		return "gateway-user"
	case strings.HasPrefix(strings.ToLower(token), GatewayPlaceholderPrefix):
		return "per-client-placeholder"
	case strings.Contains(strings.ToLower(token), strings.ToLower(LegacyGatewayPlaceholderAPIKey)):
		return "legacy-shared-placeholder"
	default:
		return "custom"
	}
}

func backupSessionDir(root string) string {
	stamp := time.Now().Format("2006-01-02T15-04-05.000000000")
	return filepath.Join(root, stamp)
}

type backupSession struct {
	dir    string
	copied map[string]struct{}
}

func newBackupSession(root string) *backupSession {
	return &backupSession{dir: backupSessionDir(root), copied: make(map[string]struct{})}
}

func (b *backupSession) Backup(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if _, exists := b.copied[path]; exists {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.IsDir() {
		return nil
	}
	if err := os.MkdirAll(b.dir, 0o755); err != nil {
		return err
	}
	target := filepath.Join(b.dir, filepath.Base(path))
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := os.WriteFile(target, data, info.Mode()); err != nil {
		return err
	}
	b.copied[path] = struct{}{}
	return nil
}

func latestBackupDir(root string) (string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	dirs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		}
	}
	if len(dirs) == 0 {
		return "", os.ErrNotExist
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dirs)))
	return filepath.Join(root, dirs[0]), nil
}
