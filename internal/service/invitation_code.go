package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"windsurf-gateway/internal/database"

	"gorm.io/gorm"
)

type InvitationCodeService struct {
	db *gorm.DB
}

func NewInvitationCodeService(db *gorm.DB) *InvitationCodeService {
	return &InvitationCodeService{db: db}
}

func (s *InvitationCodeService) Generate(maxUses int) (*database.InvitationCode, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}

	code := &database.InvitationCode{
		Code:    hex.EncodeToString(bytes),
		MaxUses: maxUses,
	}

	if err := s.db.Create(code).Error; err != nil {
		return nil, err
	}

	return code, nil
}

func (s *InvitationCodeService) Validate(code string) (bool, error) {
	var ic database.InvitationCode
	if err := s.db.Where("code = ?", code).First(&ic).Error; err != nil {
		return false, errors.New("invalid invitation code")
	}

	if ic.MaxUses > 0 && ic.UseCount >= ic.MaxUses {
		return false, errors.New("invitation code has reached max uses")
	}

	return true, nil
}

func (s *InvitationCodeService) MarkUsed(code string, userID uint) error {
	var ic database.InvitationCode
	if err := s.db.Where("code = ?", code).First(&ic).Error; err != nil {
		return err
	}

	now := time.Now()
	ic.UseCount++
	ic.UsedBy = &userID
	ic.UsedAt = &now

	return s.db.Save(&ic).Error
}

func (s *InvitationCodeService) List() ([]database.InvitationCode, error) {
	var codes []database.InvitationCode
	if err := s.db.Order("created_at DESC").Find(&codes).Error; err != nil {
		return nil, err
	}
	return codes, nil
}

func (s *InvitationCodeService) Delete(id uint) error {
	return s.db.Delete(&database.InvitationCode{}, id).Error
}
