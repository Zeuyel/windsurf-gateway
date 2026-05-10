package service

import (
	"fmt"

	"windsurf-gateway/internal/database"
)

const quotaSyncPassiveMessage = "主动构造 GetUserStatus 请求已禁用；Windsurf 配额会在真实客户端经过 Gateway 调用 GetUserStatus 成功后被动更新"

func (s *TokenService) SyncQuotaSnapshot(id string) (*database.Token, error) {
	return s.GetByID(id)
}

func (s *TokenService) SyncAllQuotaSnapshots() (int, int, []string, error) {
	tokens, err := s.GetActiveTokens()
	if err != nil {
		return 0, 0, nil, err
	}

	messages := make([]string, 0, len(tokens))
	for i := range tokens {
		messages = append(messages, fmt.Sprintf("%s: %s", tokens[i].Name, quotaSyncPassiveMessage))
	}
	return len(tokens), 0, messages, nil
}

func QuotaSyncPassiveMessage() string {
	return quotaSyncPassiveMessage
}
