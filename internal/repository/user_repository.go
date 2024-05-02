package repository

import (
	"bitcoinOrder/internal/domain/entity"
	"bitcoinOrder/pkg/utils"
	"errors"
	"gorm.io/gorm"
)

type IUserRepository interface {
	CreateUser(user entity.Users) (entity.Users, error)
	AddBalance(id string, asset string, amount float64) error
}

type UserRepository struct {
	gormDB *gorm.DB
}

var ErrUserExists = errors.New("user already exists")

func (r *UserRepository) CreateUser(user entity.Users) (entity.Users, error) {
	var existingUser entity.Users
	if err := r.gormDB.Where("email = ?", user.Email).First(&existingUser).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		return entity.Users{}, ErrUserExists
	}

	if err := r.gormDB.Create(&user).Error; err != nil {
		return entity.Users{}, err
	}

	return user, nil
}

func (r *UserRepository) AddBalance(id string, asset string, amount float64) error {
	user, err := utils.GetBalance(r.gormDB, id)
	if err != nil {
		return err
	}
	if asset == "USDT" {
		*user.UsdBalance += amount
	} else {
		return errors.New("invalid asset")
	}
	return r.gormDB.Save(&user).Error
}
