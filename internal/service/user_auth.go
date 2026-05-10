package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"windsurf-gateway/internal/database"
	"windsurf-gateway/internal/logger"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserAuthService struct {
	db                    *gorm.DB
	jwtSecret             string
	passwordMinLength     int
	passwordMaxLength     int
	invitationCodeService *InvitationCodeService
}

func NewUserAuthService(db *gorm.DB, jwtSecret string, cfg *UserAuthConfig) *UserAuthService {
	return &UserAuthService{
		db:                db,
		jwtSecret:         jwtSecret,
		passwordMinLength: cfg.PasswordMinLength,
		passwordMaxLength: cfg.PasswordMaxLength,
	}
}

type UserAuthConfig struct {
	PasswordMinLength int
	PasswordMaxLength int
}

func (s *UserAuthService) SetInvitationCodeService(svc *InvitationCodeService) {
	s.invitationCodeService = svc
}

func (s *UserAuthService) Register(username, password, email, invitationCode string) (*database.User, string, error) {
	if len(username) < 3 || len(username) > 50 {
		return nil, "", errors.New("username must be 3-50 characters")
	}
	if len(password) < s.passwordMinLength || len(password) > s.passwordMaxLength {
		return nil, "", fmt.Errorf("password must be %d-%d characters", s.passwordMinLength, s.passwordMaxLength)
	}

	if s.invitationCodeService != nil {
		valid, err := s.invitationCodeService.Validate(invitationCode)
		if err != nil || !valid {
			return nil, "", errors.New("invalid invitation code")
		}
	}

	var existing database.User
	if err := s.db.Where("username = ?", username).First(&existing).Error; err == nil {
		return nil, "", errors.New("username already exists")
	}

	apiToken, err := generateUserToken()
	if err != nil {
		return nil, "", err
	}

	user := &database.User{
		Username:    username,
		Email:       email,
		Role:        "user",
		Status:      "active",
		ApiToken:    apiToken,
		TokenStatus: "active",
	}

	if err := user.SetPassword(password); err != nil {
		return nil, "", err
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, "", err
	}

	if s.invitationCodeService != nil {
		if err := s.invitationCodeService.MarkUsed(invitationCode, user.ID); err != nil {
			logger.Warnf("Failed to mark invitation code as used: %v", err)
		}
	}

	jwtToken, err := s.generateJWT(user)
	if err != nil {
		return nil, "", err
	}

	logger.Infof("User registered: %s", username)
	return user, jwtToken, nil
}

func (s *UserAuthService) CreateManagedUser(username, password, email, role, status string, unlimitedAccess bool, rateLimitPerMinute int) (*database.User, error) {
	if len(username) < 3 || len(username) > 50 {
		return nil, errors.New("username must be 3-50 characters")
	}
	if len(password) < s.passwordMinLength || len(password) > s.passwordMaxLength {
		return nil, fmt.Errorf("password must be %d-%d characters", s.passwordMinLength, s.passwordMaxLength)
	}
	role = normalizeUserRole(role)
	status = normalizeUserStatus(status)
	if rateLimitPerMinute <= 0 {
		rateLimitPerMinute = 30
	}

	var existing database.User
	if err := s.db.Where("username = ?", username).First(&existing).Error; err == nil {
		return nil, errors.New("username already exists")
	}

	apiToken, err := generateUserToken()
	if err != nil {
		return nil, err
	}

	user := &database.User{
		Username:           username,
		Email:              email,
		Role:               role,
		Status:             status,
		ApiToken:           apiToken,
		TokenStatus:        "active",
		UnlimitedAccess:    unlimitedAccess,
		RateLimitPerMinute: rateLimitPerMinute,
	}

	if err := user.SetPassword(password); err != nil {
		return nil, err
	}
	if err := s.db.Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserAuthService) Login(username, password string) (*database.User, string, error) {
	var user database.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, "", errors.New("invalid credentials")
	}

	if !user.CheckPassword(password) {
		return nil, "", errors.New("invalid credentials")
	}

	if user.IsBanned() {
		return nil, "", errors.New("account is banned")
	}

	if !user.IsActive() {
		return nil, "", errors.New("account is disabled")
	}

	jwtToken, err := s.generateJWT(&user)
	if err != nil {
		return nil, "", err
	}

	now := time.Now()
	user.LastLogin = &now
	s.db.Save(&user)

	return &user, jwtToken, nil
}

func (s *UserAuthService) generateJWT(user *database.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *UserAuthService) ValidateToken(tokenString string) (*jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return &claims, nil
	}

	return nil, errors.New("invalid token")
}

func (s *UserAuthService) GetUserByID(id uint) (*database.User, error) {
	var user database.User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserAuthService) ListUsers(page, pageSize int, username, status string) ([]database.User, int64, error) {
	var users []database.User
	var total int64

	query := s.db.Model(&database.User{})
	if username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (s *UserAuthService) UpdateUser(id uint, updates map[string]interface{}) error {
	if rawRole, ok := updates["role"].(string); ok {
		updates["role"] = normalizeUserRole(rawRole)
	}
	if rawStatus, ok := updates["status"].(string); ok {
		updates["status"] = normalizeUserStatus(rawStatus)
	}
	return s.db.Model(&database.User{}).Where("id = ?", id).Updates(updates).Error
}

func (s *UserAuthService) DeleteUser(id uint) error {
	return s.db.Delete(&database.User{}, id).Error
}

func (s *UserAuthService) BanUser(id uint) error {
	return s.db.Model(&database.User{}).Where("id = ?", id).Update("status", "banned").Error
}

func (s *UserAuthService) UnbanUser(id uint) error {
	return s.db.Model(&database.User{}).Where("id = ?", id).Update("status", "active").Error
}

func (s *UserAuthService) ChangePassword(userID uint, oldPassword, newPassword string) error {
	var user database.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return err
	}

	if !user.CheckPassword(oldPassword) {
		return errors.New("current password is incorrect")
	}

	if len(newPassword) < s.passwordMinLength {
		return fmt.Errorf("password must be at least %d characters", s.passwordMinLength)
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.db.Model(&user).Update("password", string(hashed)).Error
}

func (s *UserAuthService) SetPasswordByAdmin(userID uint, newPassword string) error {
	if len(newPassword) < s.passwordMinLength || len(newPassword) > s.passwordMaxLength {
		return fmt.Errorf("password must be %d-%d characters", s.passwordMinLength, s.passwordMaxLength)
	}

	var user database.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return err
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.db.Model(&user).Update("password", string(hashed)).Error
}

func (s *UserAuthService) GetUserByToken(apiToken string) (*database.User, error) {
	var user database.User
	if err := s.db.Where("api_token = ?", apiToken).First(&user).Error; err != nil {
		return nil, errors.New("invalid api token")
	}
	return &user, nil
}

func (s *UserAuthService) RegenerateAPIToken(userID uint) (string, error) {
	apiToken, err := generateUserToken()
	if err != nil {
		return "", err
	}
	if err := s.db.Model(&database.User{}).Where("id = ?", userID).Update("api_token", apiToken).Error; err != nil {
		return "", err
	}
	return apiToken, nil
}

func generateUserToken() (string, error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "ws-" + hex.EncodeToString(bytes), nil
}

func normalizeUserRole(role string) string {
	switch role {
	case "admin":
		return "admin"
	default:
		return "user"
	}
}

func normalizeUserStatus(status string) string {
	switch status {
	case "disabled":
		return "disabled"
	case "banned":
		return "banned"
	default:
		return "active"
	}
}
