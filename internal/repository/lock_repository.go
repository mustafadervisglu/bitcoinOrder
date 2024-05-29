package repository

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
)

type ILockRepository interface {
	LockUSDT(tx *sql.DB, userID uuid.UUID, amount float64) error
	LockBTC(tx *sql.DB, userID uuid.UUID, amount float64) error
	UnlockUSDT(tx *sql.DB, userID uuid.UUID, amount float64) error
	UnlockBTC(tx *sql.DB, userID uuid.UUID, amount float64) error
}

type LockRepository struct {
	db *sql.DB
}

// NewLockRepository  constructor
func NewLockRepository(db *sql.DB) *LockRepository {
	return &LockRepository{db: db}
}

func (r *LockRepository) LockUSDT(tx *sql.DB, userID uuid.UUID, amount float64) error {

	sqlStatement := `
        UPDATE users
        SET usdt_balance = usdt_balance - $1
        WHERE id = $2 AND usdt_balance >= $1; 
    `
	result, err := tx.ExecContext(context.Background(), sqlStatement, amount, userID)
	if err != nil {
		return fmt.Errorf("error while locking USDT: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get number of affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("insufficient USDT balance: %v", userID)
	}

	return nil
}

func (r *LockRepository) LockBTC(tx *sql.DB, userID uuid.UUID, amount float64) error {

	sqlStatement := `
        UPDATE users
        SET btc_balance = btc_balance - $1
        WHERE id = $2 AND btc_balance >= $1;
    `
	result, err := tx.ExecContext(context.Background(), sqlStatement, amount, userID)
	if err != nil {
		return fmt.Errorf("error while locking BTC: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get number of affected rows:c %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("insufficient BTC balance: %v", userID)
	}

	return nil
}

func (r *LockRepository) UnlockUSDT(tx *sql.DB, userID uuid.UUID, amount float64) error {

	sqlStatement := `
        UPDATE users
        SET usdt_balance = usdt_balance + $1
        WHERE id = $2; 
    `
	_, err := tx.ExecContext(context.Background(), sqlStatement, amount, userID)
	if err != nil {
		return fmt.Errorf("error while removing USDT lock: %w", err)
	}
	return nil
}

func (r *LockRepository) UnlockBTC(tx *sql.DB, userID uuid.UUID, amount float64) error {

	sqlStatement := `
        UPDATE users
        SET btc_balance = btc_balance + $1
        WHERE id = $2; 
    `
	_, err := tx.ExecContext(context.Background(), sqlStatement, amount, userID)
	if err != nil {
		return fmt.Errorf("error while removing BTC lock: %w", err)
	}
	return nil
}
