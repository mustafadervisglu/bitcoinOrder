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
	"log"
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

	dbTx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			err := dbTx.Rollback()
			if err != nil {
				return
			}
			log.Println("transaction rolled back due to panic:", r)
		} else if err != nil {
			err := dbTx.Rollback()
			if err != nil {
				return
			}
			log.Println("Transaction was rolled back due to error:", err)
		} else {
			err = dbTx.Commit()
			if err != nil {
				log.Println("An error occurred while processing the transaction:", err)
			}
		}
	}()

	user, err := s.userRepo.FindUser(s.db, newOrder.UserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	switch newOrder.Type {
	case "buy":
		if err := s.lockUSDTForBuyOrder(dbTx, user, newOrder.OrderPrice*newOrder.OrderQuantity); err != nil {
			return fmt.Errorf("failed to lock USDT for buy order: %w", err)
		}
	case "sell":
		if err := s.lockBTCForSellOrder(dbTx, user, newOrder.OrderQuantity); err != nil {
			return fmt.Errorf("failed to lock BTC for sell order: %w", err)
		}
	default:
		return fmt.Errorf("invalid order type: %s", newOrder.Type)
	}

	openOrders, err := s.orderRepo.FindOpenOrdersByUser(s.db, newOrder.UserID)
	if err != nil {
		return fmt.Errorf("open orders not found %w", err)
	}

	existingOrder := s.findExistingOrder(openOrders, newOrder)

	if existingOrder != nil {
		err = s.updateExistingOrder(s.db, user, existingOrder, newOrder)
		if err != nil {
			return fmt.Errorf("failed to update existing order: %w", err)
		}
	} else {
		err = s.createNewOrder(s.db, user, openOrders, newOrder)
		if err != nil {
			return fmt.Errorf("could not create new order: %w", err)
		}
	}

	return nil
}

func (s *OrderCreatorService) fetchUserData(userID string) (entity.Users, []entity.Order, error) {
	user, err := s.userRepo.FindUser(s.db, userID)
	if err != nil {
		return entity.Users{}, nil, err
	}

	openOrders, err := s.orderRepo.FindOpenOrdersByUser(s.db, userID)
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

func (s *OrderCreatorService) updateExistingOrder(tx *sql.DB, user entity.Users, existingOrder *entity.Order, newOrder dto.OrderDto) error {
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
	if err := s.orderRepo.UpdateOrder(tx, *existingOrder); err != nil {
		return err
	}

	return nil
}

func (s *OrderCreatorService) createNewOrder(tx *sql.DB, user entity.Users, openOrders []entity.Order, newOrder dto.OrderDto) error {
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
		ID:            uuid.New(),
		Asset:         newOrder.Asset,
		OrderPrice:    newOrder.OrderPrice,
		OrderQuantity: newOrder.OrderQuantity,
		OrderStatus:   newOrder.OrderStatus,
		UserID:        userID,
		Type:          newOrder.Type,
		User:          user,
	}
	_, err = s.orderRepo.CreateOrder(tx, orderEntity)
	if err != nil {
		return err
	}

	return nil

}

func (s *OrderCreatorService) lockUSDTForBuyOrder(tx *sql.Tx, user entity.Users, amount float64) error {
	currentUSDTBalance, err := s.lockRepo.GetUserBalance(tx, user.ID, "USDT")
	if err != nil {
		return fmt.Errorf("failed to get USDT balance: %w", err)
	}

	if currentUSDTBalance >= amount {
		if err := s.lockRepo.DecreaseUserBalance(tx, user.ID, "USDT", amount); err != nil {
			return fmt.Errorf("failed to decrease USDT balance: %w", err)
		}

		newLock := entity.Lock{
			UserID: user.ID,
			Asset:  "USDT",
			Amount: amount,
		}
		if err := s.lockRepo.CreateLock(tx, newLock); err != nil {
			return fmt.Errorf("failed to create lock: %w", err)
		}
		return nil
	} else {
		return fmt.Errorf("insufficient USDT balance: %v", user.ID)
	}
}

func (s *OrderCreatorService) lockBTCForSellOrder(tx *sql.Tx, user entity.Users, amount float64) error {
	currentBTCBalance, err := s.lockRepo.GetUserBalance(tx, user.ID, "BTC")
	if err != nil {
		return fmt.Errorf("failed to get BTC balance: %w", err)
	}

	if currentBTCBalance >= amount {
		if err := s.lockRepo.DecreaseUserBalance(tx, user.ID, "BTC", amount); err != nil {
			return fmt.Errorf("failed to decrease BTC balance: %w", err)
		}

		newLock := entity.Lock{
			UserID: user.ID,
			Asset:  "BTC",
			Amount: amount,
		}
		if err := s.lockRepo.CreateLock(tx, newLock); err != nil {
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
	user, err := s.userRepo.FindUser(s.db, balance.Id)
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
	return s.userRepo.UpdateUser(s.db, user)
}

func (s *OrderCreatorService) FindAllOrder() ([]entity.Order, error) {
	orders, err := s.orderRepo.FindAllOrders(s.db)
	if err != nil {
		return nil, err
	}
	return orders, nil
}

func (s *OrderCreatorService) FindAllUser() ([]entity.Users, error) {
	return s.userRepo.FindAllUser(s.db)
}
