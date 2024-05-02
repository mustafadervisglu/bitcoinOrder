package utils

import (
	"bitcoinOrder/internal/domain/entity"
	"gorm.io/gorm"
)

func GetBalance(db *gorm.DB, userID string) (entity.Users, error) {
	var user entity.Users
	if err := db.First(&user, "id = ?", userID).Error; err != nil {
		return entity.Users{}, err
	}
	return user, nil
}
