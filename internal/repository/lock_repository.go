package repository

import (
	"bitcoinOrder/internal/domain/entity"
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
)

type ILockRepository interface {
	GetUserBalance(tx *sql.Tx, userID uuid.UUID, asset string) (float64, error)
	DecreaseUserBalance(tx *sql.Tx, userID uuid.UUID, asset string, amount float64) error
	CreateLock(tx *sql.Tx, lock entity.Lock) error
	GetLockedAmount(tx *sql.Tx, userID uuid.UUID, asset string) (float64, error)
	IncreaseUserBalance(tx *sql.Tx, userID uuid.UUID, asset string, amount float64) error
	DeleteLock(tx *sql.Tx, userID uuid.UUID, asset string) error
	UpdateLockAmount(tx *sql.Tx, userID uuid.UUID, asset string, amount float64) error
}

type LockRepository struct {
	db *sql.DB
}

// NewLockRepository  constructor
func NewLockRepository(db *sql.DB) *LockRepository {
	return &LockRepository{db: db}
}
func (r *LockRepository) GetUserBalance(tx *sql.Tx, userID uuid.UUID, asset string) (float64, error) {
	var balance float64
	sqlStatement := fmt.Sprintf("SELECT %s_balance FROM users WHERE id = $1", asset)
	err := tx.QueryRowContext(context.Background(), sqlStatement, userID).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("error while getting %s balance: %w", asset, err)
	}
	return balance, nil
}

func (r *LockRepository) DecreaseUserBalance(tx *sql.Tx, userID uuid.UUID, asset string, amount float64) error {
	sqlStatement := fmt.Sprintf(`
        UPDATE users
        SET %s_balance = %s_balance - $1
        WHERE id = $2;
    `, asset, asset)
	_, err := tx.ExecContext(context.Background(), sqlStatement, amount, userID)
	if err != nil {
		return fmt.Errorf("error while decreasing %s balance: %w", asset, err)
	}
	return nil
}

func (r *LockRepository) CreateLock(tx *sql.Tx, lock entity.Lock) error {
	err := tx.QueryRowContext(context.Background(),
		"INSERT INTO locks (user_id, asset, amount) VALUES ($1, $2, $3) RETURNING id",
		lock.UserID, lock.Asset, lock.Amount).Scan(&lock.ID)
	if err != nil {
		return fmt.Errorf("error while creating lock: %w", err)
	}
	return nil
}

func (r *LockRepository) GetLockedAmount(tx *sql.Tx, userID uuid.UUID, asset string) (float64, error) {
	var lockedAmount float64
	err := tx.QueryRowContext(context.Background(),
		"SELECT SUM(amount) FROM locks WHERE user_id = $1 AND asset = $2", userID, asset).Scan(&lockedAmount)
	if err != nil {
		return 0, fmt.Errorf("error while getting locked %s balance: %w", asset, err)
	}
	return lockedAmount, nil
}

func (r *LockRepository) IncreaseUserBalance(tx *sql.Tx, userID uuid.UUID, asset string, amount float64) error {
	sqlStatement := fmt.Sprintf(`
        UPDATE users
        SET %s_balance = %s_balance + $1 
        WHERE id = $2;
    `, asset, asset)
	_, err := tx.ExecContext(context.Background(), sqlStatement, amount, userID)
	if err != nil {
		return fmt.Errorf("error while increasing %s balance: %w", asset, err)
	}
	return nil
}

func (r *LockRepository) DeleteLock(tx *sql.Tx, userID uuid.UUID, asset string) error {
	_, err := tx.ExecContext(context.Background(),
		"DELETE FROM locks WHERE user_id = $1 AND asset = $2", userID, asset)
	if err != nil {
		return fmt.Errorf("error while deleting lock: %w", err)
	}
	return nil
}

func (r *LockRepository) UpdateLockAmount(tx *sql.Tx, userID uuid.UUID, asset string, amount float64) error {
	_, err := tx.ExecContext(context.Background(),
		"UPDATE locks SET amount = amount - $1 WHERE user_id = $2 AND asset = $3",
		amount, userID, asset)
	if err != nil {
		return fmt.Errorf("error while updating locked %s balance: %w", asset, err)
	}
	return nil
}
