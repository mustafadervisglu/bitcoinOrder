package repository

import (
	"bitcoinOrder/internal/domain/entity"
	"gorm.io/gorm"
	"time"
)

type IOrderRepository interface {
	FindOpenSellOrders() ([]entity.Order, error)
	FindOpenBuyOrders() ([]entity.Order, error)
	CreateOrder(newOrder entity.Order) (entity.Order, error)
	SoftDeleteOrder(orderId string) error
	FindOpenOrdersByUser(userID string) ([]entity.Order, error)
	FindAllOrders() ([]entity.Order, error)
}

type OrderRepository struct {
	gormDB *gorm.DB
}

func NewOrderRepository(db *gorm.DB) IOrderRepository {
	return &OrderRepository{
		gormDB: db,
	}

}
func (o *OrderRepository) CreateOrder(newOrder entity.Order) (entity.Order, error) {
	if err := o.gormDB.Create(&newOrder).Error; err != nil {
		return entity.Order{}, err
	}
	return newOrder, nil
}

func (o *OrderRepository) SoftDeleteOrder(orderId string) error {
	if err := o.gormDB.Model(&entity.Order{}).Where("id = ?", orderId).Update("deleted_at", time.Now()).Error; err != nil {
		return err
	}
	return nil
}

func (o *OrderRepository) FindOpenSellOrders() ([]entity.Order, error) {
	var sellOrders []entity.Order
	if err := o.gormDB.Where("deleted_at IS NULL AND type = ?", "sell").
		Order("created_at ASC").
		Find(&sellOrders).Error; err != nil {
		return nil, err
	}
	return sellOrders, nil
}

func (o *OrderRepository) FindOpenBuyOrders() ([]entity.Order, error) {
	var buyOrders []entity.Order
	if err := o.gormDB.Where("deleted_at IS NULL AND type = ? ", "buy").
		Order("created_at DESC").
		Find(&buyOrders).Error; err != nil {
		return nil, err
	}
	return buyOrders, nil
}

func (o *OrderRepository) FindOpenOrdersByUser(userID string) ([]entity.Order, error) {
	var orders []entity.Order
	if err := o.gormDB.Where("user_id = ? AND deleted_at IS NULL", userID).
		Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

func (o *OrderRepository) FindAllOrders() ([]entity.Order, error) {
	var orders []entity.Order
	if err := o.gormDB.Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}
