package database

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User model
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

func (u *User) IsAdmin() bool     { return u.Role == "admin" }
func (u *User) IsActive() bool    { return u.Status == "active" }
func (u *User) IsBanned() bool    { return u.Status == "banned" }
func (u *User) IsTokenActive() bool { return u.TokenStatus == "active" }

func (u *User) CanMakeRequest() bool {
	if !u.IsActive() || !u.IsTokenActive() {
		return false
	}
	if u.MaxRequests == 0 {
		return false
	}
	if u.MaxRequests > 0 && u.UsedRequests >= u.MaxRequests {
		return false
	}
	return true
}

func (u *User) IncrementUsage() { u.UsedRequests++ }

// Token model - represents a Windsurf account token
type Token struct {
	ID              string     `gorm:"primaryKey;size:20" json:"id"`
	Token           string     `gorm:"size:512;not null" json:"token"`
	Name            string     `gorm:"size:100;not null" json:"name"`
	Description     string     `gorm:"size:500" json:"description"`
	TenantAddress   string     `gorm:"size:255;not null" json:"tenant_address"`
	ProxyURL        *string    `gorm:"size:255" json:"proxy_url,omitempty"`
	Status          string     `gorm:"size:20;default:active" json:"status"`
	MaxRequests     int        `gorm:"default:30000" json:"max_requests"`
	UsedRequests    int        `gorm:"default:0" json:"used_requests"`
	ExpiresAt       *time.Time `json:"expires_at"`
	SubmitterUserID *uint      `gorm:"index" json:"submitter_user_id,omitempty"`
	SubmitterName   *string    `gorm:"size:50" json:"submitter_name,omitempty"`
	IsShared        *bool      `gorm:"default:true" json:"is_shared"`
	Email           *string    `gorm:"size:100" json:"email,omitempty"`
	PoolStatus      string     `gorm:"size:20;default:available" json:"pool_status"`
	AllocatedToID   *uint      `gorm:"index" json:"allocated_to_id,omitempty"`
	AllocatedAt     *time.Time `json:"allocated_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// InvitationCode model
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
type RequestLog struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	TokenID       string    `gorm:"index" json:"token_id"`
	UserID        *uint     `gorm:"index" json:"user_id,omitempty"`
	RequestID     string    `gorm:"size:64" json:"request_id"`
	Method        string    `gorm:"size:10" json:"method"`
	Path          string    `gorm:"size:500" json:"path"`
	UserAgent     string    `gorm:"size:500" json:"user_agent"`
	ClientIP      string    `gorm:"size:50" json:"client_ip"`
	TenantAddress string    `gorm:"size:255" json:"tenant_address"`
	StatusCode    int       `json:"status_code"`
	RequestSize   int64     `json:"request_size"`
	ResponseSize  int64     `json:"response_size"`
	Latency       int64     `json:"latency"`
	ErrorMessage  string    `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// SystemConfig model
type SystemConfig struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"uniqueIndex;size:100;not null" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Plugin model - for distributing patched plugins
type Plugin struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:100;not null" json:"name"`
	Version     string         `gorm:"size:50;not null" json:"version"`
	Description string         `gorm:"size:500" json:"description"`
	FilePath    string         `gorm:"size:500;not null" json:"file_path"`
	FileSize    int64          `json:"file_size"`
	DownloadCount int          `gorm:"default:0" json:"download_count"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
