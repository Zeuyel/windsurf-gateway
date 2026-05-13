package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server       ServerConfig
	Database     DatabaseConfig
	Redis        RedisConfig
	Security     SecurityConfig
	Proxy        ProxyConfig
	Frontend     FrontendConfig
	UserAuth     UserAuthConfig
	Telegram     TelegramConfig
	Turnstile    TurnstileConfig
	Subscription SubscriptionConfig
	Log          LogConfig
}

type ServerConfig struct {
	Host         string
	Port         int
	Mode         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

func (s *ServerConfig) GetServerAddr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

type DatabaseConfig struct {
	MySQL MySQLConfig
}

type MySQLConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
}

func (m *MySQLConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		m.Username, m.Password, m.Host, m.Port, m.Database)
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

func (r *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

type SecurityConfig struct {
	JWTSecret           string
	JWTExpiresIn        time.Duration
	JWTRefreshExpiresIn time.Duration
	AdminUsername       string
	AdminPassword       string
	AdminEmail          string
}

type ProxyConfig struct {
	Timeout               time.Duration
	MaxIdleConns          int
	MaxIdleConnsPerHost   int
	IdleConnTimeout       time.Duration
	ForwardDisabled       bool
	EnableCustomUserAgent bool
	PrivacyMode           bool
	BlockTelemetry        bool
	ScheduleTaskEnabled   bool
}

type FrontendConfig struct {
	URL        string
	StaticPath string
	APIPrefix  string
}

type UserAuthConfig struct {
	PasswordMinLength int
	PasswordMaxLength int
}

type TelegramConfig struct {
	BotToken string
	ChatID   string
	Enabled  bool
}

type TurnstileConfig struct {
	SecretKey string
	Enabled   bool
}

type SubscriptionConfig struct {
	UserAgent string
}

type LogConfig struct {
	Enabled bool
	Level   string
	Format  string
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			Port:         getEnvInt("SERVER_PORT", 8080),
			Mode:         getEnv("SERVER_MODE", "debug"),
			ReadTimeout:  time.Duration(getEnvInt("SERVER_READ_TIMEOUT", 30)) * time.Second,
			WriteTimeout: time.Duration(getEnvInt("SERVER_WRITE_TIMEOUT", 300)) * time.Second,
			IdleTimeout:  time.Duration(getEnvInt("SERVER_IDLE_TIMEOUT", 120)) * time.Second,
		},
		Database: DatabaseConfig{
			MySQL: MySQLConfig{
				Host:     getEnv("MYSQL_HOST", "localhost"),
				Port:     getEnvInt("MYSQL_PORT", 3306),
				Username: getEnv("MYSQL_USERNAME", "root"),
				Password: getEnv("MYSQL_PASSWORD", ""),
				Database: getEnv("MYSQL_DATABASE", "windsurf_gateway"),
			},
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 1),
		},
		Security: SecurityConfig{
			JWTSecret:           getEnv("JWT_SECRET", "change-me-to-a-random-secret"),
			JWTExpiresIn:        time.Duration(getEnvInt("JWT_EXPIRES_IN", 86400)) * time.Second,
			JWTRefreshExpiresIn: time.Duration(getEnvInt("JWT_REFRESH_EXPIRES_IN", 604800)) * time.Second,
			AdminUsername:       getEnv("ADMIN_USERNAME", "admin"),
			AdminPassword:       getEnv("ADMIN_PASSWORD", "admin123"),
			AdminEmail:          getEnv("ADMIN_EMAIL", ""),
		},
		Proxy: ProxyConfig{
			Timeout:               time.Duration(getEnvInt("PROXY_TIMEOUT", 300)) * time.Second,
			MaxIdleConns:          getEnvInt("PROXY_MAX_IDLE_CONNS", 100),
			MaxIdleConnsPerHost:   getEnvInt("PROXY_MAX_IDLE_CONNS_PER_HOST", 10),
			IdleConnTimeout:       time.Duration(getEnvInt("PROXY_IDLE_CONN_TIMEOUT", 90)) * time.Second,
			ForwardDisabled:       getEnvBool("FORWARD_DISABLED", false),
			EnableCustomUserAgent: getEnvBool("ENABLE_CUSTOM_USER_AGENT", false),
			PrivacyMode:           getEnvBool("PROXY_PRIVACY_MODE", true),
			BlockTelemetry:        getEnvBool("PROXY_BLOCK_TELEMETRY", true),
			ScheduleTaskEnabled:   getEnvBool("SCHEDULE_TASK_ENABLED", true),
		},
		Frontend: FrontendConfig{
			URL:        getEnv("FRONTEND_URL", ""),
			StaticPath: getEnv("FRONTEND_STATIC_PATH", "./web/dist"),
			APIPrefix:  getEnv("API_PREFIX", "/api"),
		},
		UserAuth: UserAuthConfig{
			PasswordMinLength: getEnvInt("USER_AUTH_PASSWORD_MIN_LENGTH", 6),
			PasswordMaxLength: getEnvInt("USER_AUTH_PASSWORD_MAX_LENGTH", 32),
		},
		Telegram: TelegramConfig{
			BotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
			ChatID:   getEnv("TELEGRAM_CHAT_ID", ""),
			Enabled:  getEnvBool("TELEGRAM_ENABLED", false),
		},
		Turnstile: TurnstileConfig{
			SecretKey: getEnv("TURNSTILE_SECRET_KEY", ""),
			Enabled:   getEnvBool("TURNSTILE_ENABLED", false),
		},
		Subscription: SubscriptionConfig{
			UserAgent: getEnv("SUBSCRIPTION_USER_AGENT", ""),
		},
		Log: LogConfig{
			Enabled: getEnvBool("LOG_ENABLED", true),
			Level:   getEnv("LOG_LEVEL", "info"),
			Format:  getEnv("LOG_FORMAT", "console"),
		},
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		val = strings.ToLower(val)
		return val == "true" || val == "1" || val == "yes"
	}
	return defaultVal
}
