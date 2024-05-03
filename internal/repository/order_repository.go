package repository

import (
	"bitcoinOrder/internal/domain/entity"
	"bitcoinOrder/pkg/utils"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IOrderRepository interface {
	FindAllOrder() ([]entity.Order, error)
	CreateOrder(newOrder entity.Order) (entity.Order, error)
	DeleteOrder(id string) (entity.Order, error)
}

type OrderRepository struct {
	gormDB *gorm.DB
}

func NewOrderRepository(db *gorm.DB) IOrderRepository {
	return &OrderRepository{
		gormDB: db,
	}

}
func (o *OrderRepository) FindAllOrder() ([]entity.Order, error) {
	var orders []entity.Order
	if err := o.gormDB.Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

func (o *OrderRepository) CreateOrder(newOrder entity.Order) (entity.Order, error) {

	if newOrder.OrderPrice <= 0 && newOrder.OrderQuantity <= 0 {
		return entity.Order{}, errors.New("invalid order price")
	}

	if err := utils.ValidateOrderType(utils.OrderType(newOrder.Type)); err != nil {
		return entity.Order{}, err
	}

	var user entity.Users
	if err := o.gormDB.First(&user, "id = ?", newOrder.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Order{}, errors.New("user not found")
		}
		return entity.Order{}, err
	}
	if newOrder.Type == "buy" {
		if *user.UsdBalance < (newOrder.OrderQuantity * newOrder.OrderPrice) {
			return entity.Order{}, errors.New("insufficient USD balance")
		}
	} else {
		if newOrder.OrderQuantity > *user.BtcBalance {
			return entity.Order{}, errors.New("insufficient BTC balance")
		}
	}

	if err := o.gormDB.Create(&newOrder).Error; err != nil {
		return entity.Order{}, err
	}

	return newOrder, nil
}

func (o *OrderRepository) DeleteOrder(id string) (entity.Order, error) {
	var order entity.Order
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return entity.Order{}, err
	}
	result := o.gormDB.First(&order, "id = ?", uuidID)

	if result.Error != nil {
		return entity.Order{}, result.Error
	}

	if err := o.gormDB.Delete(&order).Error; err != nil {
		return entity.Order{}, err
	}

	return order, nil
}
