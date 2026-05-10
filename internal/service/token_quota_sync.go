package service

import (
	"fmt"

	"windsurf-gateway/internal/database"
)

const quotaSyncPassiveMessage = "主动构造 GetUserStatus 请求已禁用；Windsurf 配额会在真实客户端经过 Gateway 调用 GetUserStatus 成功后被动更新"

func (s *TokenService) SyncQuotaSnapshot(id string) (*database.Token, error) {
	token, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if err := s.clearEmptyQuotaSnapshot(token); err != nil {
		return nil, err
	}
	return s.GetByID(id)
}

func (s *TokenService) SyncAllQuotaSnapshots() (int, int, []string, error) {
	tokens, err := s.GetActiveTokens()
	if err != nil {
		return 0, 0, nil, err
	}

	messages := make([]string, 0, len(tokens))
	for i := range tokens {
		if err := s.clearEmptyQuotaSnapshot(&tokens[i]); err != nil {
			return 0, 0, nil, err
		}
		messages = append(messages, fmt.Sprintf("%s: %s", tokens[i].Name, quotaSyncPassiveMessage))
	}
	return len(tokens), 0, messages, nil
}

func QuotaSyncPassiveMessage() string {
	return quotaSyncPassiveMessage
}

func (s *TokenService) clearEmptyQuotaSnapshot(token *database.Token) error {
	if token == nil || !isEmptyQuotaSnapshot(token) {
		return nil
	}
	return s.db.Model(&database.Token{}).Where("id = ?", token.ID).Updates(map[string]interface{}{
		"plan_name":                      "",
		"monthly_prompt_credits":         0,
		"monthly_flow_credits":           0,
		"monthly_flex_credits":           0,
		"available_prompt_credits":       0,
		"used_prompt_credits":            0,
		"available_flow_credits":         0,
		"used_flow_credits":              0,
		"available_flex_credits":         0,
		"used_flex_credits":              0,
		"daily_quota_remaining_percent":  0,
		"weekly_quota_remaining_percent": 0,
		"hide_daily_quota":               false,
		"hide_weekly_quota":              false,
		"daily_quota_reset_at":           nil,
		"weekly_quota_reset_at":          nil,
		"quota_updated_at":               nil,
	}).Error
}

func isEmptyQuotaSnapshot(token *database.Token) bool {
	if token == nil || token.QuotaUpdatedAt == nil {
		return false
	}
	return token.PlanName == "" &&
		token.MonthlyPromptCredits == 0 &&
		token.MonthlyFlowCredits == 0 &&
		token.MonthlyFlexCredits == 0 &&
		token.AvailablePromptCredits == 0 &&
		token.UsedPromptCredits == 0 &&
		token.AvailableFlowCredits == 0 &&
		token.UsedFlowCredits == 0 &&
		token.AvailableFlexCredits == 0 &&
		token.UsedFlexCredits == 0 &&
		token.DailyQuotaRemainingPercent == 0 &&
		token.WeeklyQuotaRemainingPercent == 0 &&
		!token.HideDailyQuota &&
		!token.HideWeeklyQuota &&
		token.DailyQuotaResetAt == nil &&
		token.WeeklyQuotaResetAt == nil
}
