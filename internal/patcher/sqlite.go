package patcher

import (
	"database/sql"
	"encoding/json"
	"os"
	"strings"

	_ "modernc.org/sqlite"
)

func openSQLite(path string) (*sql.DB, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func patchGlobalState(path, gatewayURL, authToken string) (bool, error) {
	db, err := openSQLite(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT key, value FROM ItemTable WHERE key LIKE ? OR key LIKE ?`, "%apiServerUrl%", "%BASE_API_SERVER_URL%")
	if err != nil {
		return false, err
	}

	keys := make([]string, 0, 4)
	for rows.Next() {
		var key string
		var ignored string
		if err := rows.Scan(&key, &ignored); err != nil {
			rows.Close()
			return false, err
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return false, err
	}
	if err := rows.Close(); err != nil {
		return false, err
	}
	for _, key := range keys {
		if _, err := db.Exec(`UPDATE ItemTable SET value = ? WHERE key = ?`, marshalJSONString(gatewayURL), key); err != nil {
			return false, err
		}
	}

	mockAuthStatus := map[string]any{
		"apiKey": authToken,
		"allowedCommandModelConfigsProtoBinaryBase64": []any{},
		"userStatusProtoBinaryBase64":                 "",
	}
	if err := upsertStateValue(db, "windsurfAuthStatus", mockAuthStatus); err != nil {
		return false, err
	}
	// Let Windsurf's native startup migration path replace any stale stored session
	// token with the injected gateway user token on next launch.
	if err := upsertStateValue(db, PendingAPIKeyMigrationStateKey, authToken); err != nil {
		return false, err
	}
	if err := upsertStateValue(db, "windsurfOnboarding", true); err != nil {
		return false, err
	}
	if err := upsertStateValue(db, "windsurfProductEducation", map[string]any{
		"onboardingState": 2,
		"onboardingItems": []map[string]any{{
			"id":        "windsurf.prioritized.chat.open",
			"title":     "Code with Cascade",
			"completed": true,
			"command":   "windsurf.prioritized.chat.open",
		}},
	}); err != nil {
		return false, err
	}
	return true, nil
}

func detectGlobalState(path string) GlobalStateInfo {
	info := GlobalStateInfo{}
	db, err := openSQLite(path)
	if err != nil {
		if !os.IsNotExist(err) {
			info.ReadError = err.Error()
		}
		return info
	}
	defer db.Close()
	info.Exists = true

	_ = db.QueryRow(`SELECT COUNT(1) FROM ItemTable WHERE key LIKE ? OR key LIKE ?`, "%apiServerUrl%", "%BASE_API_SERVER_URL%").Scan(&info.GatewayURLKeys)

	if raw, err := stateValue(db, "windsurfAuthStatus"); err == nil {
		var payload map[string]any
		if jsonErr := json.Unmarshal([]byte(raw), &payload); jsonErr == nil {
			if token, ok := payload["apiKey"].(string); ok {
				info.AuthTokenSummary = summarizeToken(token)
				info.AuthMode = classifyAuthMode(token)
			}
		}
	}
	if raw, err := stateValue(db, "windsurfOnboarding"); err == nil {
		var value bool
		if jsonErr := json.Unmarshal([]byte(raw), &value); jsonErr == nil {
			info.OnboardingPatched = value
		}
	}
	if raw, err := stateValue(db, "windsurfProductEducation"); err == nil {
		info.EducationPatched = strings.Contains(raw, `"onboardingState":2`) || strings.Contains(raw, `"onboardingState": 2`)
	}
	return info
}

func cleanupMockedGlobalState(path string) (int, error) {
	db, err := openSQLite(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	defer db.Close()

	count := 0
	for _, key := range []string{"windsurfAuthStatus", PendingAPIKeyMigrationStateKey, "windsurfOnboarding", "windsurfProductEducation"} {
		result, err := db.Exec(`DELETE FROM ItemTable WHERE key = ?`, key)
		if err != nil {
			return count, err
		}
		affected, _ := result.RowsAffected()
		count += int(affected)
	}
	return count, nil
}

func upsertStateValue(db *sql.DB, key string, value any) error {
	_, err := db.Exec(`
		INSERT INTO ItemTable (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, marshalJSON(value))
	return err
}

func stateValue(db *sql.DB, key string) (string, error) {
	var value string
	if err := db.QueryRow(`SELECT value FROM ItemTable WHERE key = ?`, key).Scan(&value); err != nil {
		return "", err
	}
	return value, nil
}

func marshalJSON(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func marshalJSONString(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}
