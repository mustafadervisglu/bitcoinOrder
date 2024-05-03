package service

import (
	"bitcoinOrder/internal/common/dto"
	"bitcoinOrder/internal/domain/entity"
	"bitcoinOrder/internal/repository"
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
	orderEntity := entity.Order{
		Asset:         newOrder.Asset,
		OrderPrice:    newOrder.OrderPrice,
		OrderQuantity: newOrder.OrderQuantity,
		OrderStatus:   newOrder.OrderStatus,
		UserID:        newOrder.UserID,
		Type:          newOrder.Type,
	}
	_, err := s.orderRepo.CreateOrder(orderEntity)
	if err != nil {
		return err
	}
	return nil
}

func (s *OrderCreatorService) CreateUser(newUser dto.UserDto) (entity.Users, error) {
	userEntity := entity.Users{
		Email:      newUser.Email,
		BtcBalance: &newUser.BtcBalance,
		UsdBalance: &newUser.UsdBalance,
	}
	user, err := s.userRepo.CreateUser(userEntity)
	if err != nil {
		return user, err
	}
	return user, nil
}

func (s *OrderCreatorService) FindAllOrder() ([]entity.Order, error) {
	orders, err := s.orderRepo.FindAllOrder()
	if err != nil {
		return orders, err
	}
	return orders, nil
}

func (s *OrderCreatorService) DeleteOrder(id string) (entity.Order, error) {
	res, err := s.orderRepo.DeleteOrder(id)
	if err != nil {
		return res, err
	}
	return res, nil
}

func (s *OrderCreatorService) GetBalance(id string) (dto.UserDto, error) {
	user, err := s.userRepo.GetBalance(id)
	if err != nil {
		return dto.UserDto{}, err
	}

	balance := dto.UserDto{
		Email:      user.Email,
		UsdBalance: *user.UsdBalance,
		BtcBalance: *user.BtcBalance,
	}
	return balance, nil
}
func (s *OrderCreatorService) AddBalance(balance dto.BalanceDto) error {
	err := s.userRepo.AddBalance(balance.Id, balance.Asset, balance.Amount)
	if err != nil {
		return err
	}
	return nil
}
