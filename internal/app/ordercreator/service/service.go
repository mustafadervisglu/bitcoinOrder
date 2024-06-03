package service

import (
	"bitcoinOrder/internal/common/dto"
	"bitcoinOrder/internal/domain/entity"
	"bitcoinOrder/internal/repository"
	"bitcoinOrder/pkg/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderCreatorService struct {
	orderRepo repository.IOrderRepository
	userRepo  repository.IUserRepository
	lockRepo  repository.ILockRepository
	gormDB    *gorm.DB
	db        *sql.DB
}

func NewOrderCreatorService(orderRepo repository.IOrderRepository, userRepo repository.IUserRepository, lockRepo repository.ILockRepository, gormDB *gorm.DB, db *sql.DB) *OrderCreatorService {
	return &OrderCreatorService{
		orderRepo: orderRepo,
		userRepo:  userRepo,
		lockRepo:  lockRepo,
		gormDB:    gormDB,
		db:        db,
	}
}

type IOrderCreatorService interface {
	CreateOrder(newOrder dto.OrderDto) error
	CreateUser(newUser dto.UserDto) (entity.Users, error)
	FindAllOrder(ctx context.Context) ([]entity.Order, error)
	GetBalance(id uuid.UUID) (dto.UserDto, error)
	AddBalance(ctx context.Context, balance dto.BalanceDto) error
	FindAllUser() ([]entity.Users, error)
	FindUser(ctx context.Context, userID uuid.UUID) (entity.Users, error)
}

var ErrInvalidOrderPriceOrQuantity = errors.New("invalid order price or quantity")

func (s *OrderCreatorService) CreateOrder(newOrder dto.OrderDto) error {
	ctx := context.Background()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	ctx = context.WithValue(ctx, "tx", tx)

	if err = s.createOrderWithContext(ctx, newOrder); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *OrderCreatorService) createOrderWithContext(ctx context.Context, newOrder dto.OrderDto) error {
	if newOrder.OrderPrice <= 0 || newOrder.OrderQuantity <= 0 {
		return ErrInvalidOrderPriceOrQuantity
	}
	if err := utils.ValidateOrderType(utils.OrderType(newOrder.Type)); err != nil {

		return err
	}
	user, err := s.userRepo.FindUser(ctx, newOrder.UserID)

	if err != nil {

		return fmt.Errorf("user not found: %w", err)
	}

	switch newOrder.Type {
	case "buy":
		if err = s.lockUSDTForBuyOrder(ctx, user, newOrder.OrderPrice*newOrder.OrderQuantity); err != nil {
			return fmt.Errorf("failed to lock USDT for buy order: %w", err)
		}
	case "sell":
		if err = s.lockBTCForSellOrder(ctx, user, newOrder.OrderQuantity); err != nil {

			return fmt.Errorf("failed to lock BTC for sell order: %w", err)
		}
	default:
		return fmt.Errorf("invalid order type: %s", newOrder.Type)
	}

	openOrders, err := s.orderRepo.FindOpenOrdersByUser(ctx, newOrder.UserID)
	if err != nil {

		return fmt.Errorf("open orders not found %w", err)
	}

	existingOrder := s.findExistingOrder(openOrders, newOrder)

	if existingOrder != nil {
		err = s.updateExistingOrder(ctx, user, existingOrder, newOrder)
		if err != nil {

			return fmt.Errorf("failed to update existing order: %w", err)
		}
	} else {
		err = s.createNewOrder(ctx, user, openOrders, newOrder)
		if err != nil {

			return fmt.Errorf("could not create new order: %w", err)
		}
	}

	return nil
}

func (s *OrderCreatorService) fetchUserData(ctx context.Context, userID uuid.UUID) (entity.Users, []entity.Order, error) {
	user, err := s.FindUser(ctx, userID)
	if err != nil {
		return entity.Users{}, nil, err
	}

	openOrders, err := s.orderRepo.FindOpenOrdersByUser(ctx, userID)
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

func (s *OrderCreatorService) updateExistingOrder(ctx context.Context, user entity.Users, existingOrder *entity.Order, newOrder dto.OrderDto) error {
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
	if err := s.orderRepo.UpdateOrder(ctx, *existingOrder); err != nil {
		return err
	}

	return nil
}

func (s *OrderCreatorService) createNewOrder(ctx context.Context, user entity.Users, openOrders []entity.Order, newOrder dto.OrderDto) error {
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

	orderEntity := entity.Order{
		ID:            uuid.New(),
		Asset:         newOrder.Asset,
		OrderPrice:    newOrder.OrderPrice,
		OrderQuantity: newOrder.OrderQuantity,
		OrderStatus:   newOrder.OrderStatus,
		UserID:        newOrder.UserID,
		Type:          newOrder.Type,
		User:          user,
	}
	_, err := s.orderRepo.CreateOrder(ctx, orderEntity)
	if err != nil {
		return err
	}

	return nil

}

func (s *OrderCreatorService) lockUSDTForBuyOrder(ctx context.Context, user entity.Users, amount float64) error {
	currentUSDTBalance, err := s.lockRepo.GetUserBalance(ctx, user.ID, "USDT")
	if err != nil {
		return fmt.Errorf("failed to get USDT balance: %w", err)
	}

	if currentUSDTBalance >= amount {
		if err := s.lockRepo.DecreaseUserBalance(ctx, user.ID, "USDT", amount); err != nil {
			return fmt.Errorf("failed to decrease USDT balance: %w", err)
		}

		newLock := entity.Lock{
			UserID: user.ID,
			Asset:  "USDT",
			Amount: amount,
		}
		if err := s.lockRepo.CreateLock(ctx, newLock); err != nil {
			return fmt.Errorf("failed to create lock: %w", err)
		}
		return nil
	} else {
		return fmt.Errorf("insufficient USDT balance: %v", user.ID)
	}
}

func (s *OrderCreatorService) lockBTCForSellOrder(ctx context.Context, user entity.Users, amount float64) error {
	currentBTCBalance, err := s.lockRepo.GetUserBalance(ctx, user.ID, "BTC")
	if err != nil {
		return fmt.Errorf("failed to get BTC balance: %w", err)
	}

	if currentBTCBalance >= amount {
		if err := s.lockRepo.DecreaseUserBalance(ctx, user.ID, "BTC", amount); err != nil {
			return fmt.Errorf("failed to decrease BTC balance: %w", err)
		}

		newLock := entity.Lock{
			UserID: user.ID,
			Asset:  "BTC",
			Amount: amount,
		}
		if err := s.lockRepo.CreateLock(ctx, newLock); err != nil {
			return fmt.Errorf("failed to create lock: %w", err)
		}
		return nil
	} else {
		return fmt.Errorf("insufficient BTC balance: %v", user.ID)
	}
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

func (s *OrderCreatorService) GetBalance(id uuid.UUID) (dto.UserDto, error) {
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

func (s *OrderCreatorService) AddBalance(ctx context.Context, balance dto.BalanceDto) error {
	user, err := s.userRepo.FindUser(ctx, balance.Id)
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
	return s.userRepo.UpdateUser(ctx, user)
}

func (s *OrderCreatorService) FindAllOrder(ctx context.Context) ([]entity.Order, error) {
	orders, err := s.orderRepo.FindAllOrders(ctx)
	if err != nil {
		return nil, err
	}
	return orders, nil
}

func (s *OrderCreatorService) FindAllUser() ([]entity.Users, error) {
	return s.userRepo.FindAllUser()
}

func (s *OrderCreatorService) FindUser(ctx context.Context, userID uuid.UUID) (entity.Users, error) {
	tx, err := utils.TxFromContext(ctx)
	if err != nil {
		tx, err = s.db.BeginTx(ctx, nil)
		if err != nil {
			return entity.Users{}, fmt.Errorf("failed to start transaction: %w", err)
		}
		defer tx.Rollback()

		ctx = context.WithValue(ctx, "tx", tx)
	}

	user, err := s.userRepo.FindUser(ctx, userID)
	if err != nil {
		return entity.Users{}, fmt.Errorf("user not found: %w", err)
	}

	if ctx.Value("tx") == tx {
		return user, tx.Commit()
	}

	return user, nil

}
