package database

import (
	"testing"
	"time"
)

func TestTokenHasGatewayQuotaAvailable(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name  string
		token Token
		want  bool
	}{
		{
			name: "unknown quota does not block",
			token: Token{
				QuotaUpdatedAt: nil,
			},
			want: true,
		},
		{
			name: "available prompt credits allow scheduling",
			token: Token{
				QuotaUpdatedAt:         &now,
				AvailablePromptCredits: 12,
			},
			want: true,
		},
		{
			name: "known credit counters with no remaining credits block",
			token: Token{
				QuotaUpdatedAt:       &now,
				MonthlyPromptCredits: 500,
				UsedPromptCredits:    500,
			},
			want: false,
		},
		{
			name: "known daily quota exhausted blocks when no credit counters exist",
			token: Token{
				QuotaUpdatedAt:              &now,
				DailyQuotaRemainingPercent:  0,
				DailyQuotaResetAt:           &now,
				WeeklyQuotaRemainingPercent: 50,
			},
			want: false,
		},
		{
			name: "missing percentage fields do not block by default",
			token: Token{
				QuotaUpdatedAt:              &now,
				DailyQuotaRemainingPercent:  0,
				WeeklyQuotaRemainingPercent: 0,
			},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.token.HasGatewayQuotaAvailable(); got != tc.want {
				t.Fatalf("HasGatewayQuotaAvailable() = %t, want %t", got, tc.want)
			}
		})
	}
}
