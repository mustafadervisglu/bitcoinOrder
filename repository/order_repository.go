package repository

import (
	"bitcoinOrder/entity"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type IOrderRepository interface {
	CreateUser(user entity.Users) (entity.Users, error)
	AddBalance(id string, asset string, amount float64) error
	FindAllOrder() []entity.Order
	GetBalance(id string) (entity.Users, error)
	CreateOrder(newOrder entity.Order) (entity.Order, error)
	DeleteOrder(id string) (entity.Order, error)
	CheckOrder() ([]entity.OrderMatch, error)
	UpdateBalance(orderMatches []entity.OrderMatch) error
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
	user, err := o.GetBalance(id)
	if err != nil {
		return err
	}
	if asset == "USDT" {
		*user.UsdBalance += amount
	} else {
		return errors.New("invalid asset")
	}
	err2 := o.gormDB.Save(&user).Error
	if err2 != nil {
		return err2
	}
	return nil
}

func (o *OrderRepository) FindAllOrder() []entity.Order {
	var orders []entity.Order
	o.gormDB.Find(&orders)
	return orders
}

func (o *OrderRepository) GetBalance(id string) (entity.Users, error) {
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

func (o *OrderRepository) CreateOrder(newOrder entity.Order) (entity.Order, error) {

	if newOrder.OrderPrice <= 0 && newOrder.OrderQuantity <= 0 {
		return entity.Order{}, errors.New("invalid order price")
	}

	if err := ValidateOrderType(OrderType(newOrder.Type)); err != nil {
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

func (o *OrderRepository) CheckOrder() ([]entity.OrderMatch, error) {

	if o.gormDB == nil {
		return nil, errors.New("nil db connection")
	}

	var orderMatches []entity.OrderMatch
	var buyOrders, sellOrders []entity.Order
	if err := o.gormDB.Where("order_status = ? AND type = ?", true, "buy").Order("order_price ASC, created_at ASC").Find(&buyOrders).Error; err != nil {
		return nil, err
	}
	if err := o.gormDB.Where("order_status = ? AND type = ?", true, "sell").Order("order_price DESC, created_at ASC").Find(&sellOrders).Error; err != nil {
		return nil, err
	}

	for i := 0; i < len(buyOrders); i++ {
		for j := 0; j < len(sellOrders); j++ {
			buyOrder := buyOrders[i]
			sellOrder := sellOrders[j]
			if buyOrder.OrderPrice == sellOrder.OrderPrice {
				matchQuantity := minQuantity(buyOrder.OrderQuantity, sellOrder.OrderQuantity)

				orderMatch := entity.OrderMatch{
					OrderID1:      buyOrder.ID,
					OrderID2:      sellOrder.ID,
					OrderQuantity: matchQuantity,
					MatchedAt:     time.Now(),
				}

				if err := o.gormDB.Create(&orderMatch).Error; err != nil {
					return nil, err
				}
				orderMatches = append(orderMatches, orderMatch)

				buyOrder.OrderQuantity -= matchQuantity
				sellOrder.OrderQuantity -= matchQuantity

				if buyOrder.OrderQuantity == 0 {
					buyOrder.OrderStatus = false
					now := time.Now()
					buyOrder.CompletedAt = &now
				}
				if sellOrder.OrderQuantity == 0 {
					sellOrder.OrderStatus = false
					now := time.Now()
					sellOrder.CompletedAt = &now
				}

				o.gormDB.Save(&buyOrder)
				o.gormDB.Save(&sellOrder)

				if buyOrder.OrderQuantity == 0 || sellOrder.OrderQuantity == 0 {
					break
				}
			}
		}
	}
	if len(orderMatches) > 0 {
		if err := o.UpdateBalance(orderMatches); err != nil {
			return nil, err
		}
	}

	return orderMatches, nil
}

func (o *OrderRepository) UpdateBalance(orderMatches []entity.OrderMatch) error {
	for _, match := range orderMatches {
		var buyOrder, sellOrder entity.Order
		if err := o.gormDB.First(&buyOrder, "id = ?", match.OrderID1).Error; err != nil {
			return err
		}
		if err := o.gormDB.First(&sellOrder, "id = ?", match.OrderID2).Error; err != nil {
			return err
		}

		buyUser, err := o.GetBalance(buyOrder.UserID.String())
		if err != nil {
			return err
		}
		sellUser, err := o.GetBalance(sellOrder.UserID.String())
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
func minQuantity(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// for testing purposes
func NewOrderRepository(db *gorm.DB) IOrderRepository {
	return &OrderRepository{
		gormDB: db,
	}
}
