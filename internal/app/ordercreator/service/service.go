package service

import (
	"bitcoinOrder/internal/common/dto"
	"bitcoinOrder/internal/domain/entity"
	"bitcoinOrder/internal/repository"
	"bitcoinOrder/pkg/utils"
	"errors"
	"github.com/google/uuid"
)

type OrderCreatorService struct {
	orderRepo repository.IOrderRepository
	userRepo  repository.IUserRepository
}

func NewOrderCreatorService(orderRepo repository.IOrderRepository, userRepo repository.IUserRepository) *OrderCreatorService {
	return &OrderCreatorService{
		orderRepo: orderRepo,
		userRepo:  userRepo,
	}
}

type IOrderCreatorService interface {
	CreateOrder(newOrder dto.OrderDto) error
	CreateUser(newUser dto.UserDto) (entity.Users, error)
	FindAllOrder() ([]entity.Order, error)
	DeleteOrder(id string) (entity.Order, error)
	GetBalance(id string) (dto.UserDto, error)
	AddBalance(balance dto.BalanceDto) error
}

func (s *OrderCreatorService) CreateOrder(newOrder dto.OrderDto) error {
	if newOrder.OrderPrice <= 0 || newOrder.OrderQuantity <= 0 {
		return errors.New("invalid order price or quantity")
	}

	if err := utils.ValidateOrderType(utils.OrderType(newOrder.Type)); err != nil {
		return err
	}

	user, err := s.userRepo.FindUser(newOrder.UserID)
	if err != nil {
		return err
	}

	openOrders, err := s.orderRepo.FindOpenOrdersByUser(newOrder.UserID)
	if err != nil {
		return err
	}
	if newOrder.Type == "buy" {
		totalOpenBuyValue := 0.0
		for _, order := range openOrders {
			if order.Type == "buy" {
				totalOpenBuyValue += order.OrderPrice * order.OrderQuantity
			}
		}
		if newOrder.OrderQuantity*newOrder.OrderPrice > *user.UsdtBalance-totalOpenBuyValue {
			return errors.New("insufficient usdt balance for this order")
		}
	} else if newOrder.Type == "sell" {
		totalOpenSellValue := 0.0
		for _, order := range openOrders {
			if order.Type == "sell" {
				totalOpenSellValue += order.OrderPrice * order.OrderQuantity
			}
		}
		if newOrder.OrderQuantity > *user.BtcBalance-totalOpenSellValue {
			return errors.New("insufficient BTC balance for this order")
		}

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
	}
	_, err = s.orderRepo.CreateOrder(orderEntity)

	if err != nil {
		return err
	}

	if newOrder.Type == "buy" {
		*user.UsdtBalance -= newOrder.OrderQuantity * newOrder.OrderPrice
	} else {
		*user.BtcBalance -= newOrder.OrderQuantity
	}
	return s.userRepo.UpdateUser(user)
}

func (s *OrderCreatorService) CreateUser(newUser dto.UserDto) (entity.Users, error) {
	userEntity := entity.Users{
		Email:       newUser.Email,
		BtcBalance:  &newUser.BtcBalance,
		UsdtBalance: &newUser.UsdtBalance,
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
