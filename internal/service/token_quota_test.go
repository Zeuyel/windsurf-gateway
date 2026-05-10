package service

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"testing"
	"time"
)

func TestParseUserStatusQuota(t *testing.T) {
	planInfo := protoMessage(
		protoStringField(planInfoFieldPlanName, "Windsurf Pro"),
		protoVarintField(planInfoFieldMonthlyPromptCredits, 500),
		protoVarintField(planInfoFieldMonthlyFlowCredits, 50),
		protoVarintField(planInfoFieldMonthlyFlexCredits, 10),
		protoVarintField(planInfoFieldHideDailyQuota, 0),
		protoVarintField(planInfoFieldHideWeeklyQuota, 0),
	)
	planStatus := protoMessage(
		protoBytesField(planStatusFieldPlanInfo, planInfo),
		protoVarintField(planStatusFieldAvailablePrompt, 320),
		protoVarintField(planStatusFieldUsedPrompt, 180),
		protoVarintField(planStatusFieldAvailableFlow, 40),
		protoVarintField(planStatusFieldUsedFlow, 10),
		protoVarintField(planStatusFieldAvailableFlex, 8),
		protoVarintField(planStatusFieldUsedFlex, 2),
		protoVarintField(planStatusFieldDailyRemainingPct, 72),
		protoVarintField(planStatusFieldWeeklyRemainingPct, 54),
		protoVarintField(planStatusFieldDailyResetAtUnix, 1715335200),
		protoVarintField(planStatusFieldWeeklyResetAtUnix, 1715594400),
	)
	userStatus := protoMessage(protoBytesField(userStatusFieldPlanStatus, planStatus))
	response := protoMessage(
		protoBytesField(getUserStatusResponseFieldUserStatus, userStatus),
		protoBytesField(getUserStatusResponseFieldPlanInfo, planInfo),
	)

	snapshot, err := parseUserStatusQuota(response)
	if err != nil {
		t.Fatalf("parseUserStatusQuota returned error: %v", err)
	}

	if snapshot.PlanName != "Windsurf Pro" {
		t.Fatalf("unexpected plan name: %q", snapshot.PlanName)
	}
	if snapshot.DailyQuotaRemainingPercent != 72 || snapshot.WeeklyQuotaRemainingPercent != 54 {
		t.Fatalf("unexpected quota percentages: daily=%d weekly=%d", snapshot.DailyQuotaRemainingPercent, snapshot.WeeklyQuotaRemainingPercent)
	}
	if snapshot.AvailablePromptCredits != 320 || snapshot.UsedPromptCredits != 180 {
		t.Fatalf("unexpected prompt credit values: available=%d used=%d", snapshot.AvailablePromptCredits, snapshot.UsedPromptCredits)
	}
	if snapshot.AvailableFlowCredits != 40 || snapshot.UsedFlowCredits != 10 {
		t.Fatalf("unexpected flow credit values: available=%d used=%d", snapshot.AvailableFlowCredits, snapshot.UsedFlowCredits)
	}
	if snapshot.DailyQuotaResetAt == nil || snapshot.WeeklyQuotaResetAt == nil {
		t.Fatal("expected quota reset times to be present")
	}
}

func TestParsePlanStatusQuota(t *testing.T) {
	planInfo := protoMessage(
		protoStringField(planInfoFieldPlanName, "Windsurf Pro"),
		protoVarintField(planInfoFieldMonthlyPromptCredits, 500),
		protoVarintField(planInfoFieldMonthlyFlowCredits, 50),
		protoVarintField(planInfoFieldMonthlyFlexCredits, 10),
	)
	planStatus := protoMessage(
		protoBytesField(planStatusFieldPlanInfo, planInfo),
		protoVarintField(planStatusFieldAvailablePrompt, 320),
		protoVarintField(planStatusFieldDailyRemainingPct, 72),
		protoVarintField(planStatusFieldWeeklyRemainingPct, 54),
	)
	response := protoMessage(
		protoBytesField(getPlanStatusResponseFieldPlanStatus, planStatus),
		protoVarintField(getPlanStatusResponseFieldTeamUsedPromptCreds, 180),
	)

	snapshot, err := parsePlanStatusQuota(response)
	if err != nil {
		t.Fatalf("parsePlanStatusQuota returned error: %v", err)
	}
	if snapshot.PlanName != "Windsurf Pro" {
		t.Fatalf("unexpected plan name: %q", snapshot.PlanName)
	}
	if snapshot.AvailablePromptCredits != 320 || snapshot.UsedPromptCredits != 180 {
		t.Fatalf("unexpected prompt credits: available=%d used=%d", snapshot.AvailablePromptCredits, snapshot.UsedPromptCredits)
	}
	if snapshot.DailyQuotaRemainingPercent != 72 || snapshot.WeeklyQuotaRemainingPercent != 54 {
		t.Fatalf("unexpected quota percentages: daily=%d weekly=%d", snapshot.DailyQuotaRemainingPercent, snapshot.WeeklyQuotaRemainingPercent)
	}
}

func TestDecodeUserStatusResponseBodyHandlesGzip(t *testing.T) {
	payload := protoMessage(protoBytesField(getUserStatusResponseFieldUserStatus, protoMessage()))

	var compressed bytes.Buffer
	zw := gzip.NewWriter(&compressed)
	if _, err := zw.Write(payload); err != nil {
		t.Fatalf("gzip write failed: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("gzip close failed: %v", err)
	}

	headers := make(http.Header)
	headers.Set("Content-Encoding", "gzip")
	decoded, err := decodeUserStatusResponseBody(headers, compressed.Bytes())
	if err != nil {
		t.Fatalf("decodeUserStatusResponseBody returned error: %v", err)
	}
	if !bytes.Equal(decoded, payload) {
		t.Fatal("decoded payload does not match original")
	}
}

func TestParseUserStatusQuotaRejectsEmptySnapshot(t *testing.T) {
	snapshot, err := parseUserStatusQuota(protoMessage())
	if err != nil {
		t.Fatalf("parseUserStatusQuota returned error: %v", err)
	}
	if snapshot.hasQuotaData() {
		t.Fatal("expected empty GetUserStatus response to have no quota data")
	}
}

func protoMessage(fields ...[]byte) []byte {
	var out []byte
	for _, field := range fields {
		out = append(out, field...)
	}
	return out
}

func protoStringField(fieldNumber int, value string) []byte {
	return protoBytesField(fieldNumber, []byte(value))
}

func protoBytesField(fieldNumber int, value []byte) []byte {
	var out []byte
	appendVarint(&out, uint64(fieldNumber<<3|2))
	appendVarint(&out, uint64(len(value)))
	out = append(out, value...)
	return out
}

func protoVarintField(fieldNumber int, value uint64) []byte {
	var out []byte
	appendVarint(&out, uint64(fieldNumber<<3))
	appendVarint(&out, value)
	return out
}

func TestUnixPtr(t *testing.T) {
	got := unixPtr(1715335200)
	if got == nil {
		t.Fatal("expected unixPtr to return value")
	}
	if got.Year() < 2024 || got.Location() != time.Local {
		t.Fatalf("unexpected time conversion: %v", got)
	}
}
