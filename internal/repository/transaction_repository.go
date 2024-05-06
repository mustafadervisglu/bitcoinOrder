package repository

import (
	"bitcoinOrder/internal/domain/entity"
	"bitcoinOrder/pkg/utils"
	"gorm.io/gorm"
	"time"
)

type ITransactionRepository interface {
	SaveMatches(orderMatches []entity.OrderMatch) error
	UpdateBalance(orderMatches []entity.OrderMatch) error
	MatchOrder(buyOrders, sellOrders []entity.Order) ([]entity.OrderMatch, error)
	FindBuyOrders() ([]entity.Order, error)
	FindSellOrders() ([]entity.Order, error)
}

type TransactionRepository struct {
	gormDB *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) ITransactionRepository {
	return &TransactionRepository{
		gormDB: db,
	}
}

//func (o *TransactionRepository) CheckOrder() ([]entity.OrderMatch, error) {
//
//	if o.gormDB == nil {
//		return nil, errors.New("nil db connection")
//	}
//
//	var orderMatches []entity.OrderMatch
//	var buyOrders, sellOrders []entity.Order
//	if err := o.gormDB.Where("order_status = ? AND type = ?", true, "buy").Order("order_price ASC, created_at ASC").Find(&buyOrders).Error; err != nil {
//		return nil, err
//	}
//	if err := o.gormDB.Where("order_status = ? AND type = ?", true, "sell").Order("order_price DESC, created_at ASC").Find(&sellOrders).Error; err != nil {
//		return nil, err
//	}
//
//	for i := 0; i < len(buyOrders); i++ {
//		for j := 0; j < len(sellOrders); j++ {
//			buyOrder := buyOrders[i]
//			sellOrder := sellOrders[j]
//			if buyOrder.OrderPrice == sellOrder.OrderPrice {
//				matchQuantity := utils.MinQuantity(buyOrder.OrderQuantity, sellOrder.OrderQuantity)
//
//				orderMatch := entity.OrderMatch{
//					OrderID1:      buyOrder.ID,
//					OrderID2:      sellOrder.ID,
//					OrderQuantity: matchQuantity,
//					MatchedAt:     time.Now(),
//				}
//
//				if err := o.gormDB.Create(&orderMatch).Error; err != nil {
//					return nil, err
//				}
//				orderMatches = append(orderMatches, orderMatch)
//
//				buyOrder.OrderQuantity -= matchQuantity
//				sellOrder.OrderQuantity -= matchQuantity
//
//				if buyOrder.OrderQuantity == 0 {
//					buyOrder.OrderStatus = false
//					now := time.Now()
//					buyOrder.CompletedAt = &now
//				}
//				if sellOrder.OrderQuantity == 0 {
//					sellOrder.OrderStatus = false
//					now := time.Now()
//					sellOrder.CompletedAt = &now
//				}
//
//				o.gormDB.Save(&buyOrder)
//				o.gormDB.Save(&sellOrder)
//
//				if buyOrder.OrderQuantity == 0 || sellOrder.OrderQuantity == 0 {
//					break
//				}
//			}
//		}
//	}
//
//	return orderMatches, nil
//}

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

func (o *TransactionRepository) MatchOrder(buyOrders, sellOrders []entity.Order) ([]entity.OrderMatch, error) {
	var orderMatches []entity.OrderMatch
	for i := 0; i < len(buyOrders); i++ {
		for j := 0; j < len(sellOrders); j++ {
			buyOrder := buyOrders[i]
			sellOrder := sellOrders[j]
			if sellOrder.OrderQuantity == buyOrder.OrderQuantity {
				matchQuantity := utils.MinQuantity(buyOrder.OrderQuantity, sellOrder.OrderQuantity)
				orderMatch := entity.OrderMatch{
					OrderID1:      buyOrder.ID,
					OrderID2:      sellOrder.ID,
					OrderQuantity: matchQuantity,
					MatchedAt:     time.Now(),
				}
				orderMatches = append(orderMatches, orderMatch)
			}
		}
	}
	return orderMatches, nil
}
func (o *TransactionRepository) SaveMatches(orderMatches []entity.OrderMatch) error {
	panic("implement me")
}
func (o *TransactionRepository) UpdateBalance(orderMatches []entity.OrderMatch) error {
	for _, match := range orderMatches {
		var buyOrder, sellOrder entity.Order
		if err := o.gormDB.Take(&buyOrder, "id = ?", match.OrderID1).Error; err != nil {
			return err
		}
		if err := o.gormDB.Take(&sellOrder, "id = ?", match.OrderID2).Error; err != nil {
			return err
		}

		buyUser, err := utils.GetBalance(o.gormDB, buyOrder.UserID.String())
		if err != nil {
			return err
		}
		sellUser, err := utils.GetBalance(o.gormDB, sellOrder.UserID.String())
		if err != nil {
			return err
		}

		*buyUser.UsdBalance -= buyOrder.OrderPrice * match.OrderQuantity
		*buyUser.BtcBalance += match.OrderQuantity
		*sellUser.UsdBalance += sellOrder.OrderPrice * match.OrderQuantity
		*sellUser.BtcBalance -= match.OrderQuantity

		if err := o.gormDB.Save(&buyUser).Error; err != nil {
			return err
		}
		if err := o.gormDB.Save(&sellUser).Error; err != nil {
			return err
		}
	}
	return nil
}
