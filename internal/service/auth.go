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
	"gorm.io/gorm"
)

type AuthService struct {
	db        *gorm.DB
	jwtSecret string
}

func NewAuthService(db *gorm.DB, jwtSecret string) *AuthService {
	return &AuthService{db: db, jwtSecret: jwtSecret}
}

func (s *AuthService) CreateDefaultAdmin() error {
	var count int64
	s.db.Model(&database.User{}).Where("role = ?", "admin").Count(&count)
	if count > 0 {
		return nil
	}

	admin := &database.User{
		Username: "admin",
		Role:     "admin",
		Status:   "active",
	}
	if err := admin.SetPassword("admin123"); err != nil {
		return err
	}

	if err := s.db.Create(admin).Error; err != nil {
		return err
	}

	logger.Info("Default admin user created (admin/admin123)")
	return nil
}

func (s *AuthService) Login(username, password string) (*database.User, string, error) {
	var user database.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, "", errors.New("invalid credentials")
	}

	if !user.CheckPassword(password) {
		return nil, "", errors.New("invalid credentials")
	}

	if !user.IsActive() {
		return nil, "", errors.New("account is disabled")
	}

	token, err := s.generateJWT(&user)
	if err != nil {
		return nil, "", err
	}

	now := time.Now()
	user.LastLogin = &now
	s.db.Save(&user)

	return &user, token, nil
}

func (s *AuthService) generateJWT(user *database.User) (string, error) {
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

func (s *AuthService) ValidateToken(tokenString string) (*jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
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

func (s *AuthService) GetUserByID(id uint) (*database.User, error) {
	var user database.User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *AuthService) UpdateUser(userID uint, updates map[string]interface{}) error {
	return s.db.Model(&database.User{}).Where("id = ?", userID).Updates(updates).Error
}

func generateAPIKey() (string, error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "ws-" + hex.EncodeToString(bytes), nil
}

func (s *AuthService) RegenerateAPIToken(userID uint) (string, error) {
	var user database.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return "", err
	}

	token, err := generateAPIKey()
	if err != nil {
		return "", err
	}

	user.ApiToken = token
	user.TokenStatus = "active"
	return token, s.db.Save(&user).Error
}
