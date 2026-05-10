package service

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"windsurf-gateway/internal/database"
)

const (
	userStatusFieldPlanStatus         = 13
	planStatusFieldPlanInfo           = 1
	planStatusFieldAvailableFlex      = 4
	planStatusFieldUsedFlow           = 5
	planStatusFieldUsedPrompt         = 6
	planStatusFieldUsedFlex           = 7
	planStatusFieldAvailablePrompt    = 8
	planStatusFieldAvailableFlow      = 9
	planStatusFieldDailyRemainingPct  = 14
	planStatusFieldWeeklyRemainingPct = 15
	planStatusFieldDailyResetAtUnix   = 17
	planStatusFieldWeeklyResetAtUnix  = 18
	planInfoFieldPlanName             = 2
	planInfoFieldMonthlyPromptCredits = 12
	planInfoFieldMonthlyFlowCredits   = 13
	planInfoFieldMonthlyFlexCredits   = 14
	planInfoFieldHideDailyQuota       = 36
	planInfoFieldHideWeeklyQuota      = 37
)

type TokenQuotaSnapshot struct {
	PlanName                    string
	MonthlyPromptCredits        int
	MonthlyFlowCredits          int
	MonthlyFlexCredits          int
	AvailablePromptCredits      int
	UsedPromptCredits           int
	AvailableFlowCredits        int
	UsedFlowCredits             int
	AvailableFlexCredits        int
	UsedFlexCredits             int
	DailyQuotaRemainingPercent  int
	WeeklyQuotaRemainingPercent int
	HideDailyQuota              bool
	HideWeeklyQuota             bool
	DailyQuotaResetAt           *time.Time
	WeeklyQuotaResetAt          *time.Time
}

func (s *TokenService) UpdateQuotaFromGetUserStatusResponse(tokenID string, headers http.Header, payload []byte) error {
	if strings.TrimSpace(tokenID) == "" || len(payload) == 0 {
		return nil
	}

	decoded, err := decodeUserStatusResponseBody(headers, payload)
	if err != nil {
		return err
	}
	snapshot, err := parseUserStatusQuota(decoded)
	if err != nil {
		return err
	}

	now := time.Now()
	updates := map[string]interface{}{
		"plan_name":                      snapshot.PlanName,
		"monthly_prompt_credits":         snapshot.MonthlyPromptCredits,
		"monthly_flow_credits":           snapshot.MonthlyFlowCredits,
		"monthly_flex_credits":           snapshot.MonthlyFlexCredits,
		"available_prompt_credits":       snapshot.AvailablePromptCredits,
		"used_prompt_credits":            snapshot.UsedPromptCredits,
		"available_flow_credits":         snapshot.AvailableFlowCredits,
		"used_flow_credits":              snapshot.UsedFlowCredits,
		"available_flex_credits":         snapshot.AvailableFlexCredits,
		"used_flex_credits":              snapshot.UsedFlexCredits,
		"daily_quota_remaining_percent":  snapshot.DailyQuotaRemainingPercent,
		"weekly_quota_remaining_percent": snapshot.WeeklyQuotaRemainingPercent,
		"hide_daily_quota":               snapshot.HideDailyQuota,
		"hide_weekly_quota":              snapshot.HideWeeklyQuota,
		"daily_quota_reset_at":           snapshot.DailyQuotaResetAt,
		"weekly_quota_reset_at":          snapshot.WeeklyQuotaResetAt,
		"quota_updated_at":               &now,
	}
	return s.db.Model(&database.Token{}).Where("id = ?", tokenID).Updates(updates).Error
}

func decodeUserStatusResponseBody(headers http.Header, payload []byte) ([]byte, error) {
	body := payload
	if strings.Contains(strings.ToLower(headers.Get("Content-Encoding")), "gzip") {
		reader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("gunzip GetUserStatus response failed: %w", err)
		}
		defer reader.Close()

		decoded, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("read gzipped GetUserStatus response failed: %w", err)
		}
		body = decoded
	}

	if unwrapped, ok := unwrapConnectProtoMessage(body); ok {
		return unwrapped, nil
	}
	return body, nil
}

func unwrapConnectProtoMessage(body []byte) ([]byte, bool) {
	if len(body) < 5 {
		return nil, false
	}

	idx := 0
	var message []byte
	for idx < len(body) {
		if len(body)-idx < 5 {
			return nil, false
		}
		flags := body[idx]
		length := int(uint32(body[idx+1])<<24 | uint32(body[idx+2])<<16 | uint32(body[idx+3])<<8 | uint32(body[idx+4]))
		frameStart := idx + 5
		frameEnd := frameStart + length
		if frameEnd > len(body) {
			return nil, false
		}

		if flags&0x02 == 0 && flags&0x01 == 0 && message == nil {
			message = append([]byte(nil), body[frameStart:frameEnd]...)
		}
		idx = frameEnd
	}

	if idx != len(body) || len(message) == 0 {
		return nil, false
	}
	return message, true
}

func parseUserStatusQuota(data []byte) (*TokenQuotaSnapshot, error) {
	snapshot := &TokenQuotaSnapshot{}
	idx := 0
	for idx < len(data) {
		tag, consumed, ok := decodeVarint(data, idx)
		if !ok {
			return nil, fmt.Errorf("decode UserStatus tag failed")
		}
		idx += consumed

		fieldNumber := int(tag >> 3)
		wireType := int(tag & 0x7)
		switch wireType {
		case 0:
			_, width, ok := decodeVarint(data, idx)
			if !ok {
				return nil, fmt.Errorf("decode UserStatus field %d failed", fieldNumber)
			}
			idx += width
		case 2:
			length, width, ok := decodeVarint(data, idx)
			if !ok {
				return nil, fmt.Errorf("decode UserStatus field %d length failed", fieldNumber)
			}
			idx += width
			end := idx + int(length)
			if end > len(data) {
				return nil, fmt.Errorf("UserStatus field %d exceeds payload", fieldNumber)
			}
			if fieldNumber == userStatusFieldPlanStatus {
				if err := parsePlanStatus(data[idx:end], snapshot); err != nil {
					return nil, err
				}
			}
			idx = end
		case 1:
			idx += 8
		case 5:
			idx += 4
		default:
			return nil, fmt.Errorf("unsupported UserStatus wire type %d", wireType)
		}
		if idx > len(data) {
			return nil, fmt.Errorf("UserStatus parse overflow")
		}
	}
	return snapshot, nil
}

func parsePlanStatus(data []byte, snapshot *TokenQuotaSnapshot) error {
	idx := 0
	for idx < len(data) {
		tag, consumed, ok := decodeVarint(data, idx)
		if !ok {
			return fmt.Errorf("decode PlanStatus tag failed")
		}
		idx += consumed

		fieldNumber := int(tag >> 3)
		wireType := int(tag & 0x7)
		switch wireType {
		case 0:
			value, width, ok := decodeVarint(data, idx)
			if !ok {
				return fmt.Errorf("decode PlanStatus field %d failed", fieldNumber)
			}
			switch fieldNumber {
			case planStatusFieldAvailableFlex:
				snapshot.AvailableFlexCredits = int(value)
			case planStatusFieldUsedFlow:
				snapshot.UsedFlowCredits = int(value)
			case planStatusFieldUsedPrompt:
				snapshot.UsedPromptCredits = int(value)
			case planStatusFieldUsedFlex:
				snapshot.UsedFlexCredits = int(value)
			case planStatusFieldAvailablePrompt:
				snapshot.AvailablePromptCredits = int(value)
			case planStatusFieldAvailableFlow:
				snapshot.AvailableFlowCredits = int(value)
			case planStatusFieldDailyRemainingPct:
				snapshot.DailyQuotaRemainingPercent = int(value)
			case planStatusFieldWeeklyRemainingPct:
				snapshot.WeeklyQuotaRemainingPercent = int(value)
			case planStatusFieldDailyResetAtUnix:
				snapshot.DailyQuotaResetAt = unixPtr(int64(value))
			case planStatusFieldWeeklyResetAtUnix:
				snapshot.WeeklyQuotaResetAt = unixPtr(int64(value))
			}
			idx += width
		case 2:
			length, width, ok := decodeVarint(data, idx)
			if !ok {
				return fmt.Errorf("decode PlanStatus field %d length failed", fieldNumber)
			}
			idx += width
			end := idx + int(length)
			if end > len(data) {
				return fmt.Errorf("PlanStatus field %d exceeds payload", fieldNumber)
			}
			if fieldNumber == planStatusFieldPlanInfo {
				if err := parsePlanInfo(data[idx:end], snapshot); err != nil {
					return err
				}
			}
			idx = end
		case 1:
			idx += 8
		case 5:
			idx += 4
		default:
			return fmt.Errorf("unsupported PlanStatus wire type %d", wireType)
		}
		if idx > len(data) {
			return fmt.Errorf("PlanStatus parse overflow")
		}
	}
	return nil
}

func parsePlanInfo(data []byte, snapshot *TokenQuotaSnapshot) error {
	idx := 0
	for idx < len(data) {
		tag, consumed, ok := decodeVarint(data, idx)
		if !ok {
			return fmt.Errorf("decode PlanInfo tag failed")
		}
		idx += consumed

		fieldNumber := int(tag >> 3)
		wireType := int(tag & 0x7)
		switch wireType {
		case 0:
			value, width, ok := decodeVarint(data, idx)
			if !ok {
				return fmt.Errorf("decode PlanInfo field %d failed", fieldNumber)
			}
			switch fieldNumber {
			case planInfoFieldMonthlyPromptCredits:
				snapshot.MonthlyPromptCredits = int(value)
			case planInfoFieldMonthlyFlowCredits:
				snapshot.MonthlyFlowCredits = int(value)
			case planInfoFieldMonthlyFlexCredits:
				snapshot.MonthlyFlexCredits = int(value)
			case planInfoFieldHideDailyQuota:
				snapshot.HideDailyQuota = value != 0
			case planInfoFieldHideWeeklyQuota:
				snapshot.HideWeeklyQuota = value != 0
			}
			idx += width
		case 2:
			length, width, ok := decodeVarint(data, idx)
			if !ok {
				return fmt.Errorf("decode PlanInfo field %d length failed", fieldNumber)
			}
			idx += width
			end := idx + int(length)
			if end > len(data) {
				return fmt.Errorf("PlanInfo field %d exceeds payload", fieldNumber)
			}
			if fieldNumber == planInfoFieldPlanName {
				snapshot.PlanName = string(data[idx:end])
			}
			idx = end
		case 1:
			idx += 8
		case 5:
			idx += 4
		default:
			return fmt.Errorf("unsupported PlanInfo wire type %d", wireType)
		}
		if idx > len(data) {
			return fmt.Errorf("PlanInfo parse overflow")
		}
	}
	return nil
}

func unixPtr(value int64) *time.Time {
	if value <= 0 {
		return nil
	}
	tm := time.Unix(value, 0).In(time.Local)
	return &tm
}
