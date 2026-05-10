package patcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPatchExtensionUsesGatewayUserToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "extension.js")
	content := `const DEFAULT_API_SERVER_URL="https://server.codeium.com";const DEFAULT_REGISTER_API_SERVER_URL="https://register.windsurf.com";async function run(e){return await i.authentication.getSession(n.WindsurfExtensionMetadata.getInstance().authProviderId,[o.LoginScope.LOGIN],e)};a.StatusBar.getInstance().setAuthStatus(!1),b.windsurfAuth.setAuthStatus(null),(await(0,c.getAuthSession)())?.accessToken===d||e.clearAuthentication(),!1;`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	patched, err := patchExtension(path, "https://gateway.example.com", "", "ws-test-user-token")
	if err != nil {
		t.Fatalf("patchExtension returned error: %v", err)
	}
	if !patched {
		t.Fatal("expected extension to be patched")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "https://gateway.example.com") {
		t.Fatalf("expected patched gateway URL, got: %s", got)
	}
	if !strings.Contains(got, `accessToken:"ws-test-user-token"`) {
		t.Fatalf("expected custom gateway user token in auth fallback, got: %s", got)
	}
	if !strings.Contains(got, userStatusFallbackSentinel) {
		t.Fatalf("expected user-status fallback to be injected, got: %s", got)
	}
}

func TestApplyAnonymousModeCreatesPerClientPlaceholder(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), "config")
	installDir := filepath.Join(t.TempDir(), "install")
	settingsPath := filepath.Join(configDir, "User", "settings.json")
	extensionPath := filepath.Join(installDir, "extensions", "windsurf", "dist", "extension.js")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(extensionPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(extensionPath, []byte(`const DEFAULT_API_SERVER_URL="https://server.codeium.com";async function run(e){return await i.authentication.getSession(n.WindsurfExtensionMetadata.getInstance().authProviderId,[o.LoginScope.LOGIN],e)};a.StatusBar.getInstance().setAuthStatus(!1),b.windsurfAuth.setAuthStatus(null),(await(0,c.getAuthSession)())?.accessToken===d||e.clearAuthentication(),!1;`), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Apply(ApplyOptions{
		ConfigDir:  configDir,
		InstallDir: installDir,
		GatewayURL: "https://gateway.example.com",
		Mode:       ModeAll,
	})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if result.EffectiveAuthMode != "per-client-placeholder" {
		t.Fatalf("expected per-client-placeholder mode, got %s", result.EffectiveAuthMode)
	}
	if result.Detect.PatchStateMode != "per-client-placeholder" {
		t.Fatalf("expected detected patch state mode to be per-client-placeholder, got %s", result.Detect.PatchStateMode)
	}
	if result.Detect.Settings.GatewayURL != "https://gateway.example.com" {
		t.Fatalf("expected settings gateway URL to be patched, got %s", result.Detect.Settings.GatewayURL)
	}
	if result.Detect.Extension.AuthMode != "per-client-placeholder" {
		t.Fatalf("expected extension auth mode to use per-client placeholder, got %s", result.Detect.Extension.AuthMode)
	}
}
