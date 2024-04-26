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
	if err := o.gormDB.Create(&newOrder).Error; err != nil {
		return entity.Order{}, err
	}

	return newOrder, nil
}

func (o *OrderRepository) DeleteOrder(id string) (entity.Order, error) {
	//TODO implement me
	panic("implement me")
}

func (o *OrderRepository) CheckOrder() ([]entity.OrderMatch, error) {
	var buyOrders, sellOrders []entity.Order
	var orderMatches []entity.OrderMatch
	if err := o.gormDB.Where("order_status = ?", true).Order("order_price ASC,created_at ASC").Find(&buyOrders, "type = ?", "buy").Error; err != nil {
		return orderMatches, err
	}
	if err := o.gormDB.Where("order_status = ?", true).Order("order_price DESC,created_at ASC").Find(&sellOrders, "type = ?", "sell").Error; err != nil {
		return orderMatches, err
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

				o.gormDB.Save(&buyOrder)
				o.gormDB.Save(&sellOrder)

				if buyOrder.OrderQuantity == 0 || sellOrder.OrderQuantity == 0 {
					break
				}
			}
		}
	}
	if len(orderMatches) > 0 {
		//update balance
	}

	return orderMatches, nil
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
