package utils

import (
	"bitcoinOrder/internal/domain/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetBalance(db *gorm.DB, userID uuid.UUID) (entity.Users, error) {
	var user entity.Users
	if err := db.First(&user, "id = ?", userID).Error; err != nil {
		return entity.Users{}, err
	}
	return user, nil
}
