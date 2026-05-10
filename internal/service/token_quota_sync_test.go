package service

import (
	"bytes"
	"testing"
	"time"

	"windsurf-gateway/internal/database"
)

func TestBuildPlanStatusRequestBody(t *testing.T) {
	body := buildPlanStatusRequestBody("backend-token")
	if !bytes.Contains(body, []byte("backend-token")) {
		t.Fatal("expected GetPlanStatus body to contain auth token")
	}
	if !bytes.Contains(body, []byte{0x10, 0x01}) {
		t.Fatal("expected GetPlanStatus body to request top-up status")
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
