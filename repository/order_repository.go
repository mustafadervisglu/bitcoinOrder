package repository

import (
	"bitcoinOrder/entity"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IOrderRepository interface {
	CreateUser(user entity.Users) (entity.Users, error)
	AddBalance(id string, asset string, amount float64) error
	GetBalance(userID string) (entity.Users, error)
	FindAllOrder() ([]entity.Order, error)
	FindUserByID(id string) (entity.Users, error)
	SaveOrder(order entity.Order) (entity.Order, error)
	UpdateOrder(order entity.Order) (entity.Order, error)
	Delete(id string) (entity.Order, error)
}

type OrderRepository struct {
	gormDB *gorm.DB
}

var ErrUserExists = errors.New("user already exists")

func (o *OrderRepository) CreateUser(user entity.Users) (entity.Users, error) {
	var existingUser entity.Users
	if err := o.gormDB.Where("email = ?", user.Email).First(&existingUser).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		return entity.Users{}, ErrUserExists
	}

	if err := o.gormDB.Create(&user).Error; err != nil {
		return entity.Users{}, err
	}

	return user, nil
}

func (o *OrderRepository) AddBalance(id string, asset string, amount float64) error {
	user, err := o.FindUserByID(id)
	if err != nil {
		return err
	}
	if asset == "BTC" {
		*user.BtcBalance += amount
	} else if asset == "USDT" {
		*user.UsdBalance += amount
	}
	err2 := o.gormDB.Save(&user).Error
	if err2 != nil {
		return err2
	}
	return nil
}

func (o *OrderRepository) GetBalance(userID string) (entity.Users, error) {
	//TODO implement me
	panic("implement me")
}

func (o *OrderRepository) FindAllOrder() ([]entity.Order, error) {
	//TODO implement me
	panic("implement me")
}

func (o *OrderRepository) FindUserByID(id string) (entity.Users, error) {
	var user entity.Users
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return entity.Users{}, err
	}
	result := o.gormDB.First(&user, "id = ?", uuidID)

	if result.Error != nil {
		return entity.Users{}, result.Error
	}
	return user, nil
}

func (o *OrderRepository) SaveOrder(order entity.Order) (entity.Order, error) {
	//TODO implement me
	panic("implement me")
}

func (o *OrderRepository) UpdateOrder(order entity.Order) (entity.Order, error) {
	//TODO implement me
	panic("implement me")
}

func (o *OrderRepository) Delete(id string) (entity.Order, error) {
	//TODO implement me
	panic("implement me")
}

// for testing purposes
func NewOrderRepository(db *gorm.DB) IOrderRepository {
	return &OrderRepository{
		gormDB: db,
	}
}
