package database

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User model
// Gateway users are used for gateway auth/quota, not upstream Windsurf auth.
type User struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	Username           string         `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Password           string         `gorm:"size:255;not null" json:"-"`
	Email              string         `gorm:"size:100" json:"email"`
	Role               string         `gorm:"size:20;default:user" json:"role"`
	Status             string         `gorm:"size:20;default:active" json:"status"`
	ApiToken           string         `gorm:"size:64;uniqueIndex" json:"api_token"`
	TokenStatus        string         `gorm:"size:20;default:active" json:"token_status"`
	MaxRequests        int            `gorm:"default:0" json:"max_requests"`
	UsedRequests       int            `gorm:"default:0" json:"used_requests"`
	UnlimitedAccess    bool           `gorm:"default:false" json:"unlimited_access"`
	RateLimitPerMinute int            `gorm:"default:30" json:"rate_limit_per_minute"`
	CanUseSharedTokens bool           `gorm:"default:true" json:"can_use_shared_tokens"`
	LastLogin          *time.Time     `json:"last_login"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

func (u *User) SetPassword(password string) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashed)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) == nil
}

func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

func (u *User) IsActive() bool {
	return u.Status == "active"
}

func (u *User) IsBanned() bool {
	return u.Status == "banned"
}

func (u *User) IsTokenActive() bool {
	return u.TokenStatus == "active"
}

func (u *User) CanMakeRequest() bool {
	if !u.IsActive() || !u.IsTokenActive() {
		return false
	}
	return true
}

func (u *User) IncrementUsage() {
	u.UsedRequests++
}

// Token model - represents a backend Windsurf account token.
type Token struct {
	ID                          string         `gorm:"primaryKey;size:20" json:"id"`
	Token                       string         `gorm:"size:1024;not null" json:"token"`
	Name                        string         `gorm:"size:100;not null" json:"name"`
	Description                 string         `gorm:"size:500" json:"description"`
	TenantAddress               string         `gorm:"size:255;not null" json:"tenant_address"`
	ProxyURL                    *string        `gorm:"size:255" json:"proxy_url,omitempty"`
	Status                      string         `gorm:"size:20;default:active;index" json:"status"`
	PoolStatus                  string         `gorm:"size:20;default:available;index" json:"pool_status"`
	Weight                      int            `gorm:"default:1" json:"weight"`
	MaxRequests                 int            `gorm:"default:30000" json:"max_requests"`
	UsedRequests                int            `gorm:"default:0" json:"used_requests"`
	PlanName                    string         `gorm:"size:100" json:"plan_name"`
	MonthlyPromptCredits        int            `gorm:"default:0" json:"monthly_prompt_credits"`
	MonthlyFlowCredits          int            `gorm:"default:0" json:"monthly_flow_credits"`
	MonthlyFlexCredits          int            `gorm:"default:0" json:"monthly_flex_credits"`
	AvailablePromptCredits      int            `gorm:"default:0" json:"available_prompt_credits"`
	UsedPromptCredits           int            `gorm:"default:0" json:"used_prompt_credits"`
	AvailableFlowCredits        int            `gorm:"default:0" json:"available_flow_credits"`
	UsedFlowCredits             int            `gorm:"default:0" json:"used_flow_credits"`
	AvailableFlexCredits        int            `gorm:"default:0" json:"available_flex_credits"`
	UsedFlexCredits             int            `gorm:"default:0" json:"used_flex_credits"`
	DailyQuotaRemainingPercent  int            `gorm:"default:0" json:"daily_quota_remaining_percent"`
	WeeklyQuotaRemainingPercent int            `gorm:"default:0" json:"weekly_quota_remaining_percent"`
	HideDailyQuota              bool           `gorm:"default:false" json:"hide_daily_quota"`
	HideWeeklyQuota             bool           `gorm:"default:false" json:"hide_weekly_quota"`
	DailyQuotaResetAt           *time.Time     `json:"daily_quota_reset_at,omitempty"`
	WeeklyQuotaResetAt          *time.Time     `json:"weekly_quota_reset_at,omitempty"`
	QuotaUpdatedAt              *time.Time     `json:"quota_updated_at,omitempty"`
	ActiveRequests              int            `gorm:"default:0" json:"active_requests"`
	ConsecutiveFailures         int            `gorm:"default:0" json:"consecutive_failures"`
	TotalFailures               int            `gorm:"default:0" json:"total_failures"`
	TotalSuccesses              int            `gorm:"default:0" json:"total_successes"`
	LastStatusCode              int            `gorm:"default:0" json:"last_status_code"`
	LastError                   string         `gorm:"size:1000" json:"last_error,omitempty"`
	LastUsedAt                  *time.Time     `json:"last_used_at,omitempty"`
	LastErrorAt                 *time.Time     `json:"last_error_at,omitempty"`
	CooldownUntil               *time.Time     `json:"cooldown_until,omitempty"`
	ExpiresAt                   *time.Time     `json:"expires_at"`
	SubmitterUserID             *uint          `gorm:"index" json:"submitter_user_id,omitempty"`
	SubmitterName               *string        `gorm:"size:50" json:"submitter_name,omitempty"`
	IsShared                    *bool          `gorm:"default:true" json:"is_shared"`
	Email                       *string        `gorm:"size:100" json:"email,omitempty"`
	AllocatedToID               *uint          `gorm:"index" json:"allocated_to_id,omitempty"`
	AllocatedAt                 *time.Time     `json:"allocated_at,omitempty"`
	CreatedAt                   time.Time      `json:"created_at"`
	UpdatedAt                   time.Time      `json:"updated_at"`
	DeletedAt                   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (t *Token) IsExpired() bool {
	return t.ExpiresAt != nil && t.ExpiresAt.Before(time.Now())
}

func (t *Token) IsDisabled() bool {
	return t.Status == "disabled"
}

func (t *Token) IsExhausted() bool {
	return t.MaxRequests > 0 && t.UsedRequests >= t.MaxRequests
}

func (t *Token) IsCoolingDown() bool {
	return t.CooldownUntil != nil && t.CooldownUntil.After(time.Now())
}

func (t *Token) IsActive() bool {
	if t.Status != "active" {
		return false
	}
	if t.IsExpired() || t.IsExhausted() {
		return false
	}
	return true
}

func (t *Token) IsReadyForScheduling(now time.Time) bool {
	if t.Status != "active" {
		return false
	}
	if t.ExpiresAt != nil && t.ExpiresAt.Before(now) {
		return false
	}
	if t.MaxRequests > 0 && t.UsedRequests >= t.MaxRequests {
		return false
	}
	if t.CooldownUntil != nil && t.CooldownUntil.After(now) {
		return false
	}
	return t.PoolStatus == "available"
}

func (t *Token) HasGatewayQuotaAvailable() bool {
	if t.QuotaUpdatedAt == nil {
		return true
	}

	// Prefer concrete credit counters when present. This avoids false blocking
	// when daily/weekly percentages are absent or not populated by Windsurf.
	if t.AvailablePromptCredits > 0 || t.AvailableFlowCredits > 0 || t.AvailableFlexCredits > 0 {
		return true
	}
	if t.MonthlyPromptCredits > 0 || t.MonthlyFlowCredits > 0 || t.MonthlyFlexCredits > 0 ||
		t.UsedPromptCredits > 0 || t.UsedFlowCredits > 0 || t.UsedFlexCredits > 0 {
		return false
	}

	dailyQuotaKnown := t.HideDailyQuota || t.DailyQuotaResetAt != nil || t.DailyQuotaRemainingPercent > 0
	if dailyQuotaKnown && !t.HideDailyQuota && t.DailyQuotaRemainingPercent <= 0 {
		return false
	}

	weeklyQuotaKnown := t.HideWeeklyQuota || t.WeeklyQuotaResetAt != nil || t.WeeklyQuotaRemainingPercent > 0
	if weeklyQuotaKnown && !t.HideWeeklyQuota && t.WeeklyQuotaRemainingPercent <= 0 {
		return false
	}

	return true
}

// InvitationCode model
// Invitation codes gate gateway user registration.
type InvitationCode struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Code      string         `gorm:"uniqueIndex;size:32;not null" json:"code"`
	UsedBy    *uint          `json:"used_by,omitempty"`
	UsedAt    *time.Time     `json:"used_at,omitempty"`
	MaxUses   int            `gorm:"default:1" json:"max_uses"`
	UseCount  int            `gorm:"default:0" json:"use_count"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// RequestLog model
// Request logs are the primary observability source for gateway routing decisions.
type RequestLog struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	TokenID            string    `gorm:"size:20;index" json:"token_id"`
	TokenName          string    `gorm:"size:100" json:"token_name"`
	UserID             *uint     `gorm:"index" json:"user_id,omitempty"`
	Username           string    `gorm:"size:100" json:"username,omitempty"`
	RequestID          string    `gorm:"size:64;index" json:"request_id"`
	Method             string    `gorm:"size:10" json:"method"`
	Path               string    `gorm:"size:500" json:"path"`
	UserAgent          string    `gorm:"size:500" json:"user_agent"`
	ClientIP           string    `gorm:"size:50" json:"client_ip"`
	TenantAddress      string    `gorm:"size:255" json:"tenant_address"`
	StatusCode         int       `json:"status_code"`
	UpstreamStatusCode int       `json:"upstream_status_code"`
	RequestSize        int64     `json:"request_size"`
	ResponseSize       int64     `json:"response_size"`
	Latency            int64     `json:"latency"`
	FailureCategory    string    `gorm:"size:50;index" json:"failure_category,omitempty"`
	ErrorMessage       string    `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt          time.Time `gorm:"index" json:"created_at"`
}

// SystemConfig model
// Used for lightweight runtime configuration such as load balancing strategy.
type SystemConfig struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"uniqueIndex;size:100;not null" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Plugin model - for distributing patched plugins
type Plugin struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	Name          string         `gorm:"size:100;not null" json:"name"`
	Version       string         `gorm:"size:50;not null" json:"version"`
	Description   string         `gorm:"size:500" json:"description"`
	FilePath      string         `gorm:"size:500;not null" json:"file_path"`
	FileSize      int64          `json:"file_size"`
	DownloadCount int            `gorm:"default:0" json:"download_count"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}
