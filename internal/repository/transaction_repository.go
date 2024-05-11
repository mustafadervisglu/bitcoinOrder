package repository

import (
	"bitcoinOrder/internal/domain/entity"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type ITransactionRepository interface {
	SaveMatches(orderMatches []entity.OrderMatch) error
	FindBuyOrders() ([]entity.Order, error)
	FindSellOrders() ([]entity.Order, error)
	UpdateBalance(users []*entity.Users) error
	FindUserById(userId uuid.UUID) (*entity.Users, error)
	SoftDeleteOrder(orderId uuid.UUID) error
	UpdateOrders(orders []*entity.Order) error
	FindOrderById(orderId uuid.UUID) (entity.Order, error)
	SoftDeleteMatch(matchID uuid.UUID) error
	FetchMatch(orderID1 uuid.UUID, orderID2 uuid.UUID) (*entity.OrderMatch, error)
}

type TransactionRepository struct {
	gormDB *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) ITransactionRepository {
	return &TransactionRepository{
		gormDB: db,
	}
}

func (o *TransactionRepository) FindSellOrders() ([]entity.Order, error) {
	var sellOrders []entity.Order
	if err := o.gormDB.
		Where("order_status = ? AND type = ?", true, "sell").
		Order("order_price DESC, created_at ASC").
		Find(&sellOrders).Error; err != nil {
		return nil, err
	}
	return sellOrders, nil
}

func (o *TransactionRepository) FindBuyOrders() ([]entity.Order, error) {
	var buyOrders []entity.Order
	if err := o.gormDB.
		Where("order_status = ? AND type = ?", true, "buy").
		Order("order_price ASC , created_at ASC").
		Find(&buyOrders).Error; err != nil {
		return nil, err
	}
	return buyOrders, nil
}

func (o *TransactionRepository) SaveMatches(orderMatches []entity.OrderMatch) error {
	if len(orderMatches) == 0 {
		return nil
	}
	if err := o.gormDB.Create(&orderMatches).Error; err != nil {
		return err
	}
	return nil
}

func (o *TransactionRepository) UpdateBalance(users []*entity.Users) error {
	for _, user := range users {
		updates := map[string]interface{}{
			"usd_balance": user.UsdBalance,
			"btc_balance": user.BtcBalance,
		}
		if err := o.gormDB.Model(user).Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func (o *TransactionRepository) FindOrderById(orderId uuid.UUID) (entity.Order, error) {
	var order entity.Order
	if err := o.gormDB.Take(&order, "id = ?", orderId).Error; err != nil {
		return entity.Order{}, err
	}
	return order, nil
}

func (o *TransactionRepository) FindUserById(userId uuid.UUID) (*entity.Users, error) {
	var user entity.Users
	if err := o.gormDB.Take(&user, "id = ?", userId).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (o *TransactionRepository) SoftDeleteOrder(orderId uuid.UUID) error {
	if err := o.gormDB.Model(&entity.Order{}).Where("id = ?", orderId).Update("deleted_at", time.Now()).Error; err != nil {
		return err
	}
	return nil
}

func (o *TransactionRepository) UpdateOrders(orders []*entity.Order) error {
	for _, order := range orders {
		if err := o.gormDB.Model(order).Updates(map[string]interface{}{
			"order_quantity": order.OrderQuantity,
			"order_status":   order.OrderStatus,
			"completed_at":   order.CompletedAt,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (o *TransactionRepository) SoftDeleteMatch(matchID uuid.UUID) error {
	if err := o.gormDB.Model(&entity.OrderMatch{}).Where("id = ?", matchID).Update("deleted_at", time.Now()).Error; err != nil {
		return err
	}
	return nil
}

func (o *TransactionRepository) FetchMatch(orderID1 uuid.UUID, orderID2 uuid.UUID) (*entity.OrderMatch, error) {
	var match entity.OrderMatch
	err := o.gormDB.Model(&entity.OrderMatch{}).Where("order_id1 = ? AND order_id2 = ?", orderID1, orderID2).First(&match).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &match, nil
}
