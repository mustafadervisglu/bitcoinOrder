package service

import (
	"bitcoinOrder/internal/domain/entity"
	"bitcoinOrder/internal/repository"
	"gorm.io/gorm"
	"log"
	"sort"
	"time"
)

type OrderCheckerService struct {
	transactionRepo repository.ITransactionRepository
	gormDB          *gorm.DB
}

func NewOrderCheckerService(repo repository.ITransactionRepository, gormDB *gorm.DB) *OrderCheckerService {
	return &OrderCheckerService{
		transactionRepo: repo,
		gormDB:          gormDB,
	}
}

func (s *OrderCheckerService) MatchOrder(tx *gorm.DB, buyOrders, sellOrders []entity.Order) ([]entity.OrderMatch, error) {

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
				buyOrder.CompletedAt = &time.Time{}
			}
			if sellOrder.OrderQuantity == 0 {
				sellOrder.OrderStatus = false
				now := time.Now()
				sellOrder.CompletedAt = &now
				j++
			} else {
				sellOrder.CompletedAt = &time.Time{}
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

	if err := s.transactionRepo.UpdateOrders(tx, ordersToUpdate); err != nil {
		log.Println(err)
		return nil, err
	}

	if err := s.transactionRepo.SaveMatches(tx, orderMatches); err != nil {
		return nil, err
	}
	return orderMatches, nil
}
func (s *OrderCheckerService) UpdateUserBalances(tx *gorm.DB, orderMatches []entity.OrderMatch) error {
	for _, match := range orderMatches {
		buyOrder, err := s.transactionRepo.FindOrderById(tx, match.OrderID1)
		if err != nil {
			return err
		}
		sellOrder, err := s.transactionRepo.FindOrderById(tx, match.OrderID2)
		if err != nil {
			return err
		}
		buyUser := &buyOrder.User
		sellUser := &sellOrder.User

		*buyUser.UsdtBalance -= buyOrder.OrderPrice * match.OrderQuantity
		*buyUser.BtcBalance += match.OrderQuantity
		*sellUser.UsdtBalance += sellOrder.OrderPrice * match.OrderQuantity
		*sellUser.BtcBalance -= match.OrderQuantity
		if err := s.transactionRepo.UpdateBalance(tx, []*entity.Users{buyUser, sellUser}); err != nil {
			return err
		}
	}
	return nil
}

func (s *OrderCheckerService) SoftDeleteOrderMatch(tx *gorm.DB, orderMatches []entity.OrderMatch) error {
	for _, match := range orderMatches {
		fetchedMatch, err := s.transactionRepo.FetchMatch(tx, match.OrderID1, match.OrderID2)
		if err != nil {
			return err
		}
		if fetchedMatch != nil {
			if err := s.transactionRepo.SoftDeleteMatch(tx, fetchedMatch.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *OrderCheckerService) ProcessTransactions() error {
	tx := s.gormDB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	buyOrders, err := s.transactionRepo.FindBuyOrders(tx)
	if err != nil {
		tx.Rollback()
		return err
	}

	sellOrders, err := s.transactionRepo.FindSellOrders(tx)
	if err != nil {
		tx.Rollback()
		return err
	}

	orderMatches, err := s.MatchOrder(tx, buyOrders, sellOrders)
	if err != nil {
		tx.Rollback()
		return err
	}
	if err := s.UpdateUserBalances(tx, orderMatches); err != nil {
		tx.Rollback()
		return err
	}

	if err := s.SoftDeleteOrderMatch(tx, orderMatches); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
