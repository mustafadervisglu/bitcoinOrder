package service

import (
	"bitcoinOrder/entity"
	"bitcoinOrder/repository"
	"bitcoinOrder/service/dto"
)

type IBitcoinOrderService interface {
	CreateUser(user dto.UserDto) error
	AddBalance(balanceDto dto.BalanceDto) error
	GetBalance(userID string) (dto.UserDto, error)
	FindAllOrder() ([]dto.OrderDto, error)
	FindUserByID(id string) (dto.UserDto, error)
	SaveOrder(order dto.OrderDto) error
	UpdateOrder(order dto.OrderDto) error
	Delete(id string) error
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

func (b *BitcoinOrderService) GetBalance(userID string) (dto.UserDto, error) {
	//TODO implement me
	panic("implement me")
}

func (b *BitcoinOrderService) FindAllOrder() ([]dto.OrderDto, error) {
	//TODO implement me
	panic("implement me")
}

func (b *BitcoinOrderService) FindUserByID(id string) (dto.UserDto, error) {
	res, err := b.bitcoinOrderRepository.FindUserByID(id)
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

func (b *BitcoinOrderService) SaveOrder(order dto.OrderDto) error {
	//TODO implement me
	panic("implement me")
}

func (b *BitcoinOrderService) UpdateOrder(order dto.OrderDto) error {
	//TODO implement me
	panic("implement me")
}

func (b *BitcoinOrderService) Delete(id string) error {
	//TODO implement me
	panic("implement me")
}

func NewBitcoinOrderService(bitcoinOrderRepository repository.IOrderRepository) IBitcoinOrderService {
	return &BitcoinOrderService{
		bitcoinOrderRepository: bitcoinOrderRepository,
	}
}
