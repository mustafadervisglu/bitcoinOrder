package repository

import (
	"bitcoinOrder/internal/domain/entity"
	"bitcoinOrder/pkg/utils"
	"errors"
	"github.com/labstack/gommon/email"
	"gorm.io/gorm"
)

type IUserRepository interface {
	CreateUser(user entity.Users) (entity.Users, error)
	UpdateUser(user entity.Users) error
	GetBalance(id string) (entity.Users, error)
	FindUser(id string) (entity.Users, error)
	FindUserByEmail(email email.Email) (entity.Users, error)
}

type UserRepository struct {
	gormDB *gorm.DB
}

func NewUserRepository(db *gorm.DB) IUserRepository {
	return &UserRepository{
		gormDB: db,
	}
}

var ErrUserExists = errors.New("user already exists")

func (r *UserRepository) CreateUser(user entity.Users) (entity.Users, error) {

	if err := r.gormDB.Create(&user).Error; err != nil {
		return entity.Users{}, err
	}

	return user, nil
}

func (r *UserRepository) UpdateUser(user entity.Users) error {
	return r.gormDB.Save(&user).Error
}

func (r *UserRepository) GetBalance(id string) (entity.Users, error) {
	return utils.GetBalance(r.gormDB, id)
}

func (r *UserRepository) FindUser(id string) (entity.Users, error) {
	var user entity.Users
	if err := r.gormDB.Where("id = ?", id).
		Take(&user).Error; err != nil {
		return entity.Users{}, err
	}
	return user, nil
}

func (r *UserRepository) FindUserByEmail(email email.Email) (entity.Users, error) {
	var existingUser entity.Users
	if err := r.gormDB.Where("email = ?", email).Take(&existingUser).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		return entity.Users{}, ErrUserExists
	}
	return existingUser, nil
}
