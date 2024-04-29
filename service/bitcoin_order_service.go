package service

import (
	"bitcoinOrder/entity"
	"bitcoinOrder/repository"
	"bitcoinOrder/service/dto"
)

type IBitcoinOrderService interface {
	CreateUser(user dto.UserDto) error
	AddBalance(balanceDto dto.BalanceDto) error
	FindAllOrder() ([]dto.OrderDto, error)
	GetBalance(id string) (dto.UserDto, error)
	CreateOrder(order dto.OrderDto) error
	DeleteOrder(id string) error
}

type BitcoinOrderService struct {
	bitcoinOrderRepository repository.IOrderRepository
}

func (b *BitcoinOrderService) CreateUser(userDto dto.UserDto) error {
	userEntity := entity.Users{
		Email:      userDto.Email,
		BtcBalance: &userDto.BtcBalance,
		UsdBalance: &userDto.UsdBalance,
	}

	user, err := b.bitcoinOrderRepository.CreateUser(userEntity)
	if err != nil {
		return err
	}

	_ = user
	return nil
}

func (b *BitcoinOrderService) AddBalance(balanceDto dto.BalanceDto) error {
	err := b.bitcoinOrderRepository.AddBalance(balanceDto.Id, balanceDto.Asset, balanceDto.Amount)
	if err != nil {
		return err
	}
	return nil
}

func (b *BitcoinOrderService) FindAllOrder() ([]dto.OrderDto, error) {
	orders := b.bitcoinOrderRepository.FindAllOrder()
	var orderDtos []dto.OrderDto
	for _, order := range orders {
		orderDto := dto.OrderDto{
			Asset:         order.Asset,
			OrderPrice:    order.OrderPrice,
			OrderQuantity: order.OrderQuantity,
			OrderStatus:   order.OrderStatus,
			UserID:        order.UserID,
			Type:          order.Type,
		}
		orderDtos = append(orderDtos, orderDto)
	}
	return orderDtos, nil
}

func (b *BitcoinOrderService) GetBalance(id string) (dto.UserDto, error) {
	res, err := b.bitcoinOrderRepository.GetBalance(id)
	if err != nil {
		return dto.UserDto{}, err
	}
	user := dto.UserDto{
		Email:      res.Email,
		BtcBalance: *res.BtcBalance,
		UsdBalance: *res.UsdBalance,
	}
	return user, nil
}

func (b *BitcoinOrderService) CreateOrder(order dto.OrderDto) error {
	//TODO implement me
	panic("implement me")
}

func (b *BitcoinOrderService) DeleteOrder(id string) error {
	//TODO implement me
	panic("implement me")
}

func NewBitcoinOrderService(bitcoinOrderRepository repository.IOrderRepository) IBitcoinOrderService {
	return &BitcoinOrderService{
		bitcoinOrderRepository: bitcoinOrderRepository,
	}
}
