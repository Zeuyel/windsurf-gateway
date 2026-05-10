package patcher

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestPatchGlobalStateSeedsPendingAPIKeyMigration(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "User", "globalStorage", "state.vscdb")
	db := mustCreatePatcherStateDB(t, dbPath)
	if _, err := db.Exec(`INSERT INTO ItemTable (key, value) VALUES (?, ?)`, "codeium.apiServerUrl", `"https://server.codeium.com"`); err != nil {
		t.Fatalf("seed ItemTable: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close seed db: %v", err)
	}

	const gatewayURL = "https://gateway.example.com"
	const authToken = "devin-session-token$abcdef1234567890"
	patched, err := patchGlobalState(dbPath, gatewayURL, authToken)
	if err != nil {
		t.Fatalf("patchGlobalState returned error: %v", err)
	}
	if !patched {
		t.Fatal("expected patchGlobalState to report patched")
	}

	verifyDB, err := openSQLite(dbPath)
	if err != nil {
		t.Fatalf("open patched db: %v", err)
	}
	defer verifyDB.Close()

	var rawGateway string
	if err := verifyDB.QueryRow(`SELECT value FROM ItemTable WHERE key = ?`, "codeium.apiServerUrl").Scan(&rawGateway); err != nil {
		t.Fatalf("read patched gateway url: %v", err)
	}
	if rawGateway != `"`+gatewayURL+`"` {
		t.Fatalf("expected patched gateway url %q, got %q", gatewayURL, rawGateway)
	}

	pendingRaw, err := stateValue(verifyDB, PendingAPIKeyMigrationStateKey)
	if err != nil {
		t.Fatalf("read pending migration key: %v", err)
	}
	var pendingToken string
	if err := json.Unmarshal([]byte(pendingRaw), &pendingToken); err != nil {
		t.Fatalf("decode pending migration key: %v", err)
	}
	if pendingToken != authToken {
		t.Fatalf("expected pending migration token %q, got %q", authToken, pendingToken)
	}

	authStatusRaw, err := stateValue(verifyDB, "windsurfAuthStatus")
	if err != nil {
		t.Fatalf("read windsurfAuthStatus: %v", err)
	}
	var authStatus map[string]any
	if err := json.Unmarshal([]byte(authStatusRaw), &authStatus); err != nil {
		t.Fatalf("decode windsurfAuthStatus: %v", err)
	}
	if got, _ := authStatus["apiKey"].(string); got != authToken {
		t.Fatalf("expected windsurfAuthStatus apiKey %q, got %q", authToken, got)
	}
}

func mustCreatePatcherStateDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE ItemTable (key TEXT PRIMARY KEY, value TEXT)`); err != nil {
		db.Close()
		t.Fatalf("create ItemTable: %v", err)
	}
	return db
}
