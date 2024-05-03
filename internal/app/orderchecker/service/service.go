package service

import (
	"bitcoinOrder/internal/domain/entity"
	"bitcoinOrder/internal/repository"
)

type OrderCheckerService struct {
	transactionRepo repository.ITransactionRepository
}

func NewOrderCheckerService(repo repository.ITransactionRepository) *OrderCheckerService {
	return &OrderCheckerService{
		transactionRepo: repo,
	}
}

func (s *OrderCheckerService) ProcessTransactions() error {
	orderMatches, err := s.transactionRepo.CheckOrder()
	if err != nil {
		return err
	}
	return s.UpdateBalances(orderMatches)
}

func (s *OrderCheckerService) UpdateBalances(matches []entity.OrderMatch) error {
	if len(matches) == 0 {
		return nil
	}
	return s.transactionRepo.UpdateBalance(matches)
}
