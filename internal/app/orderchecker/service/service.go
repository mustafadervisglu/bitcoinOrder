package service

import (
	"bitcoinOrder/internal/domain/entity"
	"bitcoinOrder/internal/repository"
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"log"
	"sort"
	"time"
)

type OrderCheckerService struct {
	transactionRepo repository.ITransactionRepository
	lockRepo        repository.ILockRepository
	db              *sql.DB
}

func NewOrderCheckerService(transactionRepo repository.ITransactionRepository, lockRepo repository.ILockRepository, db *sql.DB) *OrderCheckerService {
	return &OrderCheckerService{
		transactionRepo: transactionRepo,
		lockRepo:        lockRepo,
		db:              db,
	}
}

func (s *OrderCheckerService) MatchOrder(ctx context.Context, buyOrders, sellOrders []entity.Order) ([]entity.OrderMatch, error) {
	var orderMatches []entity.OrderMatch
	var ordersToUpdate []*entity.Order

	if len(buyOrders) == 0 || len(sellOrders) == 0 {
		return nil, nil
	}
	sort.Slice(buyOrders, func(i, j int) bool {
		if buyOrders[i].OrderPrice == buyOrders[j].OrderPrice {
			return buyOrders[i].CreatedAt.Before(buyOrders[j].CreatedAt)
		}
		return buyOrders[i].OrderPrice < buyOrders[j].OrderPrice
	})

	sort.Slice(sellOrders, func(i, j int) bool {
		if sellOrders[i].OrderPrice == sellOrders[j].OrderPrice {
			return sellOrders[i].CreatedAt.Before(sellOrders[j].CreatedAt)
		}
		return sellOrders[i].OrderPrice > sellOrders[j].OrderPrice
	})

	i := 0
	j := 0

	for i < len(buyOrders) && j < len(sellOrders) {
		buyOrder := &buyOrders[i]
		sellOrder := &sellOrders[j]

		if buyOrder.OrderPrice >= sellOrder.OrderPrice {
			var matchQuantity float64
			if buyOrder.OrderQuantity < sellOrder.OrderQuantity {
				matchQuantity = buyOrder.OrderQuantity
			} else {
				matchQuantity = sellOrder.OrderQuantity
			}

			orderMatch := entity.OrderMatch{
				ID:            uuid.New(),
				OrderID1:      buyOrder.ID,
				OrderID2:      sellOrder.ID,
				OrderQuantity: matchQuantity,
				MatchedAt:     time.Now(),
			}
			orderMatches = append(orderMatches, orderMatch)

			buyOrder.OrderQuantity -= matchQuantity
			sellOrder.OrderQuantity -= matchQuantity

			if buyOrder.OrderQuantity == 0 {
				buyOrder.OrderStatus = false
				now := time.Now()
				buyOrder.CompletedAt = &now
				i++
			} else {
				buyOrder.CompletedAt = nil
			}
			if sellOrder.OrderQuantity == 0 {
				sellOrder.OrderStatus = false
				now := time.Now()
				sellOrder.CompletedAt = &now
				j++
			} else {
				sellOrder.CompletedAt = nil
			}

			ordersToUpdate = append(ordersToUpdate, buyOrder, sellOrder)
		} else {
			if buyOrder.OrderPrice < sellOrder.OrderPrice {
				i++
			} else {
				j++
			}
		}
	}
	if err := s.transactionRepo.UpdateOrders(ctx, ordersToUpdate); err != nil {
		return nil, fmt.Errorf("failed to update orders: %w", err)
	}
	if err := s.transactionRepo.SaveMatches(ctx, orderMatches); err != nil {
		return nil, fmt.Errorf("failed to save matches: %w", err)
	}

	return orderMatches, nil
}

func (s *OrderCheckerService) UpdateUserBalances(ctx context.Context, orderMatches []entity.OrderMatch) error {
	for _, match := range orderMatches {
		buyOrder, err := s.transactionRepo.FindOrderById(ctx, match.OrderID1)
		if err != nil {
			return fmt.Errorf("failed to find buy order: %w", err)
		}

		sellOrder, err := s.transactionRepo.FindOrderById(ctx, match.OrderID2)
		if err != nil {
			return fmt.Errorf("failed to find sell order: %w", err)
		}
		buyUser := &buyOrder.User
		sellUser := &sellOrder.User

		*buyUser.UsdtBalance -= buyOrder.OrderPrice * match.OrderQuantity
		*buyUser.BtcBalance += match.OrderQuantity
		*sellUser.UsdtBalance += sellOrder.OrderPrice * match.OrderQuantity
		*sellUser.BtcBalance -= match.OrderQuantity

		if err := s.transactionRepo.UpdateBalance(ctx, []*entity.Users{buyUser, sellUser}); err != nil {
			return fmt.Errorf("failed to update user balances: %w", err)
		}

		if err := s.manageLocksAfterMatch(ctx, buyUser, sellUser, buyOrder, match.OrderQuantity); err != nil {
			return fmt.Errorf("failed to manage locks: %w", err)
		}
	}

	return nil
}

func (s *OrderCheckerService) manageLocksAfterMatch(ctx context.Context, buyUser, sellUser *entity.Users, buyOrder entity.Order, matchQuantity float64) error {
	if err := s.manageLockForAsset(ctx, buyUser, "USDT", buyOrder.OrderPrice*matchQuantity); err != nil {
		return fmt.Errorf("failed to manage USDT lock for buyer: %w", err)
	}
	if err := s.manageLockForAsset(ctx, sellUser, "BTC", matchQuantity); err != nil {
		return fmt.Errorf("failed to manage BTC lock for seller: %w", err)
	}
	return nil
}

func (s *OrderCheckerService) manageLockForAsset(ctx context.Context, user *entity.Users, asset string, amount float64) error {
	lockedAmount, err := s.lockRepo.GetLockedAmount(ctx, user.ID, asset)
	if err != nil {
		return fmt.Errorf("failed to get locked %s amount: %w", asset, err)
	}
	if lockedAmount >= amount {
		if err = s.lockRepo.IncreaseUserBalance(ctx, user.ID, asset, amount); err != nil {
			return fmt.Errorf("failed to increase %s balance: %w", asset, err)
		}
		if lockedAmount == amount {
			if err = s.lockRepo.DeleteLock(ctx, user.ID, asset); err != nil {
				return fmt.Errorf("failed to delete %s lock: %w", asset, err)
			}
		} else {
			if err = s.lockRepo.UpdateLockAmount(ctx, user.ID, asset, amount); err != nil {
				return fmt.Errorf("failed to update %s lock amount: %w", asset, err)
			}
		}
	} else {
		return fmt.Errorf("insufficient locked %s usdt_balance: %v, btc_balance: %v, lock_amount:%v", asset, user.UsdtBalance, user.BtcBalance, amount)
	}

	return nil
}

func (s *OrderCheckerService) SoftDeleteOrderMatch(ctx context.Context, orderMatches []entity.OrderMatch) error {
	for _, match := range orderMatches {
		fetchedMatch, err := s.transactionRepo.FetchMatch(ctx, match.OrderID1, match.OrderID2)
		if err != nil {
			return fmt.Errorf("failed to fetch order match: %w", err)
		}
		if fetchedMatch != nil {
			if err := s.transactionRepo.SoftDeleteMatch(ctx, fetchedMatch.ID); err != nil {
				return fmt.Errorf("failed to soft delete order match: %w", err)
			}
		}
	}
	return nil
}

func (s *OrderCheckerService) ProcessTransactions() error {
	ctx := context.Background()
	dbTx, err := s.db.BeginTx(ctx, nil)
	ctx = context.WithValue(ctx, "tx", dbTx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			err := dbTx.Rollback()
			if err != nil {
				log.Printf("Transaction rollback failed during panic recovery: %v\n", err)
				return
			}
			log.Println("transaction rolled back due to panic:", r)
		} else if err != nil {
			err := dbTx.Rollback()
			if err != nil {
				log.Printf("Transaction rollback failed due to error: %v\n", err)
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

	buyOrders, err := s.transactionRepo.FindBuyOrders(ctx)
	if err != nil {
		return fmt.Errorf("buy orders could not be retrieved: %w", err)
	}

	sellOrders, err := s.transactionRepo.FindSellOrders(ctx)
	if err != nil {
		return fmt.Errorf("sell orders could not be retrieved: %w", err)
	}
	orderMatches, err := s.MatchOrder(ctx, buyOrders, sellOrders)
	if err != nil {
		return fmt.Errorf("matching orders failed: %w", err)
	}

	err = s.UpdateUserBalances(ctx, orderMatches)
	if err != nil {
		return fmt.Errorf("user balances could not be updated: %w", err)
	}

	err = s.SoftDeleteOrderMatch(ctx, orderMatches)
	if err != nil {
		return fmt.Errorf("order matches could not be deleted: %w", err)
	}

	return nil
}
