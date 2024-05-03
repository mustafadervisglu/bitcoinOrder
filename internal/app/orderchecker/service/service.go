package service

import (
	"bitcoinOrder/internal/domain/entity"
	"bitcoinOrder/internal/repository"
)

type TransactionService struct {
	transactionRepo repository.ITransactionRepository
}

func NewTransactionService(repo repository.ITransactionRepository) *TransactionService {
	return &TransactionService{
		transactionRepo: repo,
	}
}

func (s *TransactionService) ProcessTransactions() error {
	orderMatches, err := s.transactionRepo.CheckOrder()
	if err != nil {
		return err
	}
	return s.UpdateBalances(orderMatches)
}

func (s *TransactionService) UpdateBalances(matches []entity.OrderMatch) error {
	if len(matches) == 0 {
		return nil
	}
	return s.transactionRepo.UpdateBalance(matches)
}
