package repository

import (
	"bitcoinOrder/internal/domain/entity"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"log"
	"time"
)

type ITransactionRepository interface {
	FindSellOrders(tx *gorm.DB) ([]entity.Order, error)
	FindBuyOrders(tx *gorm.DB) ([]entity.Order, error)
	SaveMatches(tx *gorm.DB, orderMatches []entity.OrderMatch) error
	UpdateBalance(tx *gorm.DB, users []*entity.Users) error
	FindOrderById(tx *gorm.DB, orderId uuid.UUID) (entity.Order, error)
	FindUserById(tx *gorm.DB, userId uuid.UUID) (*entity.Users, error)
	SoftDeleteOrder(tx *gorm.DB, orderId uuid.UUID) error
	UpdateOrders(tx *gorm.DB, orders []*entity.Order) error
	SoftDeleteMatch(tx *gorm.DB, matchID uuid.UUID) error
	FetchMatch(tx *gorm.DB, orderID1 uuid.UUID, orderID2 uuid.UUID) (*entity.OrderMatch, error)
}

type TransactionRepository struct {
	gormDB *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) ITransactionRepository {
	return &TransactionRepository{
		gormDB: db,
	}
}

func (o *TransactionRepository) FindBuyOrders(tx *gorm.DB) ([]entity.Order, error) {
	var buyOrders []entity.Order
	if err := tx.Joins("User").Where("order_status = ? AND type = ?", true, "buy").
		Order("order_price ASC , created_at ASC").
		Find(&buyOrders).Error; err != nil {
		return nil, err
	}
	return buyOrders, nil
}

func (o *TransactionRepository) FindSellOrders(tx *gorm.DB) ([]entity.Order, error) {
	var sellOrders []entity.Order
	if err := tx.Joins("User").Where("order_status = ? AND type = ?", true, "sell").
		Order("order_price DESC, created_at ASC").
		Find(&sellOrders).Error; err != nil {
		return nil, err
	}
	return sellOrders, nil
}

func (o *TransactionRepository) SaveMatches(tx *gorm.DB, orderMatches []entity.OrderMatch) error {
	if len(orderMatches) == 0 {
		return nil
	}
	if err := tx.Create(&orderMatches).Error; err != nil {
		return err
	}
	return nil
}

func (o *TransactionRepository) UpdateBalance(tx *gorm.DB, users []*entity.Users) error {
	var updates []map[string]interface{}
	for _, user := range users {
		updates = append(updates, map[string]interface{}{
			"usdt_balance": user.UsdtBalance,
			"btc_balance":  user.BtcBalance,
		})
	}

	if err := tx.Model(&entity.Users{}).
		Clauses(clause.OnConflict{DoNothing: true}).
		Where("id IN (?)", o.getIDsFromUsers(users)).
		Updates(updates).Error; err != nil {
		return err
	}

	return nil
}
func (o *TransactionRepository) getIDsFromUsers(users []*entity.Users) []uuid.UUID {
	var ids []uuid.UUID
	for _, user := range users {
		ids = append(ids, user.ID)
	}
	return ids
}

func (o *TransactionRepository) FindOrderById(tx *gorm.DB, orderId uuid.UUID) (entity.Order, error) {
	var order entity.Order
	if err := tx.Preload("User").Take(&order, "id = ?", orderId).Error; err != nil {
		log.Println(order.User)
		return entity.Order{}, err
	}
	log.Println(order.User)
	return order, nil
}

func (o *TransactionRepository) FindUserById(tx *gorm.DB, userId uuid.UUID) (*entity.Users, error) {
	var user entity.Users
	if err := tx.Take(&user, "id = ?", userId).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (o *TransactionRepository) SoftDeleteOrder(tx *gorm.DB, orderId uuid.UUID) error {
	if err := tx.Model(&entity.Order{}).Where("id = ?", orderId).Update("deleted_at", time.Now()).Error; err != nil {
		return err
	}
	return nil
}

func (o *TransactionRepository) UpdateOrders(tx *gorm.DB, orders []*entity.Order) error {
	var updates []map[string]interface{}
	for _, order := range orders {
		updates = append(updates, map[string]interface{}{
			"order_quantity": order.OrderQuantity,
			"order_status":   order.OrderStatus,
			"completed_at":   order.CompletedAt,
		})
	}
	// bylk update
	if err := tx.Model(&entity.Order{}).
		Clauses(clause.OnConflict{DoNothing: true}).
		Where("id IN (?)", o.getIdsFromOrders(orders)).
		Updates(updates).Error; err != nil {

		return err
	}
	return nil
}
func (o *TransactionRepository) getIdsFromOrders(orders []*entity.Order) []uuid.UUID {
	var ids []uuid.UUID

	for _, order := range orders {
		ids = append(ids, order.ID)
	}
	log.Println(ids)
	return ids
}

func (o *TransactionRepository) SoftDeleteMatch(tx *gorm.DB, matchID uuid.UUID) error {
	if err := tx.Model(&entity.OrderMatch{}).Where("id = ?", matchID).Update("deleted_at", time.Now()).Error; err != nil {
		return err
	}
	return nil
}

func (o *TransactionRepository) FetchMatch(tx *gorm.DB, orderID1 uuid.UUID, orderID2 uuid.UUID) (*entity.OrderMatch, error) {
	var match entity.OrderMatch
	err := tx.Model(&entity.OrderMatch{}).Where("order_id1 = ? AND order_id2 = ?", orderID1, orderID2).First(&match).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &match, nil
}
