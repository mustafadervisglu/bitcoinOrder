package service

import (
	"bitcoinOrder/internal/common/dto"
	"bitcoinOrder/internal/domain/entity"
	"bitcoinOrder/internal/repository"
	"bitcoinOrder/pkg/utils"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderCreatorService struct {
	orderRepo repository.IOrderRepository
	userRepo  repository.IUserRepository
	gormDB    *gorm.DB
}

func NewOrderCreatorService(orderRepo repository.IOrderRepository, userRepo repository.IUserRepository, gormDB *gorm.DB) *OrderCreatorService {
	return &OrderCreatorService{
		orderRepo: orderRepo,
		userRepo:  userRepo,
		gormDB:    gormDB,
	}
}

type IOrderCreatorService interface {
	CreateOrder(newOrder dto.OrderDto) error
	CreateUser(newUser dto.UserDto) (entity.Users, error)
	FindAllOrder() ([]entity.Order, error)
	GetBalance(id string) (dto.UserDto, error)
	AddBalance(balance dto.BalanceDto) error
	FindAllUser() ([]entity.Users, error)
}

var ErrInvalidOrderPriceOrQuantity = errors.New("invalid order price or quantity")

func (s *OrderCreatorService) CreateOrder(newOrder dto.OrderDto) error {

	if newOrder.OrderPrice <= 0 || newOrder.OrderQuantity <= 0 {
		return ErrInvalidOrderPriceOrQuantity
	}

	if err := utils.ValidateOrderType(utils.OrderType(newOrder.Type)); err != nil {
		return err
	}

	tx := s.gormDB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	user, openOrders, err := s.fetchUserData(newOrder.UserID)
	if err != nil {
		tx.Rollback()
		return err
	}

	existingOrder := s.findExistingOrder(openOrders, newOrder)

	if existingOrder != nil {

		err = s.updateExistingOrder(user, existingOrder, newOrder)
		if err != nil {
			tx.Rollback()
			return err
		}
	} else {

		err = s.createNewOrder(user, openOrders, newOrder)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (s *OrderCreatorService) fetchUserData(userID string) (entity.Users, []entity.Order, error) {
	user, err := s.userRepo.FindUser(userID)
	if err != nil {
		return entity.Users{}, nil, err
	}

	openOrders, err := s.orderRepo.FindOpenOrdersByUser(userID)
	if err != nil {
		return entity.Users{}, nil, err
	}

	return user, openOrders, nil
}

func (s *OrderCreatorService) findExistingOrder(openOrders []entity.Order, newOrder dto.OrderDto) *entity.Order {
	for _, order := range openOrders {
		if order.Type == newOrder.Type && order.OrderPrice == newOrder.OrderPrice && order.OrderStatus {
			return &order
		}
	}
	return nil
}

func (s *OrderCreatorService) updateExistingOrder(user entity.Users, existingOrder *entity.Order, newOrder dto.OrderDto) error {
	if newOrder.Type == "buy" {
		if newOrder.OrderQuantity*newOrder.OrderPrice > *user.UsdtBalance-existingOrder.OrderPrice*existingOrder.OrderQuantity {
			return errors.New("insufficient usdt balance for this order")
		}
		*user.UsdtBalance -= newOrder.OrderQuantity * newOrder.OrderPrice
	} else {
		if newOrder.OrderQuantity+existingOrder.OrderQuantity > *user.BtcBalance {
			return errors.New("insufficient BTC balance for this order")
		}
		*user.BtcBalance -= newOrder.OrderQuantity
	}

	existingOrder.OrderQuantity += newOrder.OrderQuantity
	if err := s.orderRepo.UpdateOrder(*existingOrder); err != nil {
		return err
	}
	return nil
}

func (s *OrderCreatorService) createNewOrder(user entity.Users, openOrders []entity.Order, newOrder dto.OrderDto) error {
	if newOrder.Type == "buy" {
		totalOpenBuyValue := 0.0
		for _, order := range openOrders {
			if order.Type == "buy" && order.OrderStatus {
				totalOpenBuyValue += order.OrderPrice * order.OrderQuantity
			}
		}
		if newOrder.OrderQuantity*newOrder.OrderPrice > *user.UsdtBalance-totalOpenBuyValue {
			return errors.New("insufficient usdt balance for this order")
		}
		*user.UsdtBalance -= newOrder.OrderQuantity * newOrder.OrderPrice
	} else {
		if newOrder.OrderQuantity > *user.BtcBalance {
			return errors.New("insufficient BTC balance for this order")
		}
		*user.BtcBalance -= newOrder.OrderQuantity
	}

	userID, err := uuid.Parse(newOrder.UserID)
	if err != nil {
		return err
	}
	orderEntity := entity.Order{
		Asset:         newOrder.Asset,
		OrderPrice:    newOrder.OrderPrice,
		OrderQuantity: newOrder.OrderQuantity,
		OrderStatus:   newOrder.OrderStatus,
		UserID:        userID,
		Type:          newOrder.Type,
		User:          user,
	}
	_, err = s.orderRepo.CreateOrder(orderEntity)
	return err
}

func (s *OrderCreatorService) CreateUser(newUser dto.UserDto) (entity.Users, error) {
	btcBalance := newUser.BtcBalance
	usdtBalance := newUser.UsdtBalance

	userEntity := entity.Users{
		Email:       newUser.Email,
		BtcBalance:  &btcBalance,
		UsdtBalance: &usdtBalance,
	}
	user, err := s.userRepo.CreateUser(userEntity)
	if err != nil {
		return user, err
	}
	return user, nil
}

func (s *OrderCreatorService) GetBalance(id string) (dto.UserDto, error) {
	user, err := s.userRepo.GetBalance(id)
	if err != nil {
		return dto.UserDto{}, err
	}

	balance := dto.UserDto{
		Email:       user.Email,
		UsdtBalance: *user.UsdtBalance,
		BtcBalance:  *user.BtcBalance,
	}
	return balance, nil
}

func (s *OrderCreatorService) AddBalance(balance dto.BalanceDto) error {
	user, err := s.userRepo.FindUser(balance.Id)
	if err != nil {
		return err
	}

	switch balance.Asset {
	case "BTC":
		*user.BtcBalance += balance.Amount
	case "USD":
		*user.UsdtBalance += balance.Amount
	default:
		return errors.New("invalid asset")
	}
	return s.userRepo.UpdateUser(user)
}

func (s *OrderCreatorService) FindAllOrder() ([]entity.Order, error) {
	orders, err := s.orderRepo.FindAllOrders()
	if err != nil {
		return nil, err
	}
	return orders, nil
}

func (s *OrderCreatorService) FindAllUser() ([]entity.Users, error) {
	return s.userRepo.FindAllUser()
}
