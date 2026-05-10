package service

import (
	"testing"
	"time"

	"windsurf-gateway/internal/database"
)

func TestQuotaSyncPassiveMessage(t *testing.T) {
	if QuotaSyncPassiveMessage() == "" {
		t.Fatal("expected passive quota sync message")
	}
}

func TestIsEmptyQuotaSnapshot(t *testing.T) {
	now := time.Now()
	if !isEmptyQuotaSnapshot(&database.Token{QuotaUpdatedAt: &now}) {
		t.Fatal("expected timestamp-only quota snapshot to be empty")
	}
	if isEmptyQuotaSnapshot(&database.Token{QuotaUpdatedAt: &now, PlanName: "Windsurf Pro"}) {
		t.Fatal("expected named quota snapshot to be non-empty")
	}
	if isEmptyQuotaSnapshot(&database.Token{}) {
		t.Fatal("expected unknown quota snapshot to be preserved")
	}
}
