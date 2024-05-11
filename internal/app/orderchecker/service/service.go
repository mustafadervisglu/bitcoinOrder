package service

import (
	"bitcoinOrder/internal/domain/entity"
	"bitcoinOrder/internal/repository"
	"bitcoinOrder/pkg/utils"
	"time"
)

type OrderCheckerService struct {
	transactionRepo repository.ITransactionRepository
}

func NewOrderCheckerService(repo repository.ITransactionRepository) *OrderCheckerService {
	return &OrderCheckerService{
		transactionRepo: repo,
	}
}

func (s *OrderCheckerService) MatchOrder(buyOrders, sellOrders []entity.Order) ([]entity.OrderMatch, error) {
	var orderMatches []entity.OrderMatch
	var ordersToUpdate []*entity.Order

	for i := 0; i < len(buyOrders); i++ {
		for j := 0; j < len(sellOrders); j++ {
			buyOrder := &buyOrders[i]
			sellOrder := &sellOrders[j]

			if buyOrder.OrderPrice == sellOrder.OrderPrice {
				matchQuantity := utils.MinQuantity(buyOrder.OrderQuantity, sellOrder.OrderQuantity)
				orderMatch := entity.OrderMatch{
					OrderID1:      buyOrder.ID,
					OrderID2:      sellOrder.ID,
					OrderQuantity: matchQuantity,
					MatchedAt:     time.Now(),
				}
				orderMatches = append(orderMatches, orderMatch)

				buyOrder.OrderQuantity -= matchQuantity
				sellOrder.OrderQuantity -= matchQuantity

				ordersToUpdate = append(ordersToUpdate, buyOrder, sellOrder)

				now := time.Now()

				if buyOrder.OrderQuantity == 0 {
					buyOrder.OrderStatus = false
					buyOrder.CompletedAt = &now

				}
				if sellOrder.OrderQuantity == 0 {
					sellOrder.OrderStatus = false
					sellOrder.CompletedAt = &now

				}
			}
		}
	}

	if err := s.transactionRepo.UpdateOrders(ordersToUpdate); err != nil {
		return nil, err
	}

	if err := s.transactionRepo.SaveMatches(orderMatches); err != nil {
		return nil, err
	}

	return orderMatches, nil
}

func (s *OrderCheckerService) UpdateUserBalances(orderMatches []entity.OrderMatch) error {
	for _, match := range orderMatches {
		buyOrder, err := s.transactionRepo.FindOrderById(match.OrderID1)
		if err != nil {
			return err
		}

		sellOrder, err := s.transactionRepo.FindOrderById(match.OrderID2)
		if err != nil {
			return err
		}

		buyUser, err := s.transactionRepo.FindUserById(buyOrder.UserID)
		if err != nil {
			return err
		}

		sellUser, err := s.transactionRepo.FindUserById(sellOrder.UserID)
		if err != nil {
			return err
		}

		*buyUser.UsdBalance -= buyOrder.OrderPrice * match.OrderQuantity
		*buyUser.BtcBalance += match.OrderQuantity
		*sellUser.UsdBalance += sellOrder.OrderPrice * match.OrderQuantity
		*sellUser.BtcBalance -= match.OrderQuantity

		if err := s.transactionRepo.UpdateBalance([]*entity.Users{buyUser, sellUser}); err != nil {
			return err
		}
	}

	return nil
}

func (s *OrderCheckerService) SoftDeleteOrderMatch(orderMatches []entity.OrderMatch) error {
	for _, match := range orderMatches {
		fetchedMatch, err := s.transactionRepo.FetchMatch(match.OrderID1, match.OrderID2)
		if err != nil {
			return err
		}
		if fetchedMatch != nil {
			if err := s.transactionRepo.SoftDeleteMatch(fetchedMatch.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *OrderCheckerService) ProcessTransactions() error {
	buyOrders, err := s.transactionRepo.FindBuyOrders()
	if err != nil {
		return err
	}
	sellOrders, err := s.transactionRepo.FindSellOrders()
	if err != nil {
		return err
	}

	orderMatches, err := s.MatchOrder(buyOrders, sellOrders)
	if err != nil {
		return err
	}

	if err := s.UpdateUserBalances(orderMatches); err != nil {
		return err
	}

	if err := s.SoftDeleteOrderMatch(orderMatches); err != nil {
		return err
	}

	return nil
}
