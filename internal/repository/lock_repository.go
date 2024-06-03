package repository

import (
	"bitcoinOrder/internal/domain/entity"
	"bitcoinOrder/pkg/utils"
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
)

type ILockRepository interface {
	GetUserBalance(ctx context.Context, userID uuid.UUID, asset string) (float64, error)
	DecreaseUserBalance(ctx context.Context, userID uuid.UUID, asset string, amount float64) error
	CreateLock(ctx context.Context, lock entity.Lock) error
	GetLockedAmount(ctx context.Context, userID uuid.UUID, asset string) (float64, error)
	IncreaseUserBalance(ctx context.Context, userID uuid.UUID, asset string, amount float64) error
	DeleteLock(ctx context.Context, userID uuid.UUID, asset string) error
	UpdateLockAmount(ctx context.Context, userID uuid.UUID, asset string, amount float64) error
}

type LockRepository struct {
	db *sql.DB
}

// NewLockRepository  constructor
func NewLockRepository(db *sql.DB) *LockRepository {
	return &LockRepository{db: db}
}
func (r *LockRepository) GetUserBalance(ctx context.Context, userID uuid.UUID, asset string) (float64, error) {

	tx, err := utils.TxFromContext(ctx)
	if err != nil {
		return 0, err
	}
	var balance float64
	sqlStatement := fmt.Sprintf("SELECT %s_balance FROM users WHERE id = $1", asset)
	err = tx.QueryRowContext(ctx, sqlStatement, userID).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("error while getting %s balance: %w", asset, err)
	}
	return balance, nil
}

func (r *LockRepository) DecreaseUserBalance(ctx context.Context, userID uuid.UUID, asset string, amount float64) error {
	tx, err := utils.TxFromContext(ctx)
	if err != nil {
		return err
	}

	sqlStatement := fmt.Sprintf(`
        UPDATE users
        SET %s_balance = %s_balance - $1
        WHERE id = $2;
    `, asset, asset)
	_, err = tx.ExecContext(ctx, sqlStatement, amount, userID)
	if err != nil {
		return fmt.Errorf("error while decreasing %s balance: %w", asset, err)
	}
	return nil
}

func (r *LockRepository) CreateLock(ctx context.Context, lock entity.Lock) error {

	tx, err := utils.TxFromContext(ctx)
	if err != nil {
		return err
	}

	err = tx.QueryRowContext(ctx,
		"INSERT INTO locks (user_id, asset, amount) VALUES ($1, $2, $3) RETURNING id",
		lock.UserID, lock.Asset, lock.Amount).Scan(&lock.ID)
	if err != nil {
		return fmt.Errorf("error while creating lock: %w", err)
	}
	return nil
}

// converting NULL COALESCE(SUM(amount), 0) band-aid
func (r *LockRepository) GetLockedAmount(ctx context.Context, userID uuid.UUID, asset string) (float64, error) {
	tx, err := utils.TxFromContext(ctx)
	if err != nil {
		return 0, err
	}

	var lockedAmount float64
	err = tx.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(amount), 0) FROM locks WHERE user_id = $1 AND asset = $2", userID, asset).Scan(&lockedAmount)
	if err != nil {
		return 0, fmt.Errorf("error while getting locked %s balance: %w", asset, err)
	}
	return lockedAmount, nil
}

func (r *LockRepository) IncreaseUserBalance(ctx context.Context, userID uuid.UUID, asset string, amount float64) error {
	tx, err := utils.TxFromContext(ctx)
	if err != nil {
		return err
	}

	sqlStatement := fmt.Sprintf(`
        UPDATE users
        SET %s_balance = %s_balance + $1 
        WHERE id = $2;
    `, asset, asset)
	_, err = tx.ExecContext(ctx, sqlStatement, amount, userID)
	if err != nil {
		return fmt.Errorf("error while increasing %s balance: %w", asset, err)
	}
	return nil
}

func (r *LockRepository) DeleteLock(ctx context.Context, userID uuid.UUID, asset string) error {
	tx, err := utils.TxFromContext(ctx)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx,
		"DELETE FROM locks WHERE user_id = $1 AND asset = $2", userID, asset)
	if err != nil {
		return fmt.Errorf("error while deleting lock: %w", err)
	}
	return nil
}

func (r *LockRepository) UpdateLockAmount(ctx context.Context, userID uuid.UUID, asset string, amount float64) error {
	tx, err := utils.TxFromContext(ctx)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx,
		"UPDATE locks SET amount = amount - $1 WHERE user_id = $2 AND asset = $3",
		amount, userID, asset)
	if err != nil {
		return fmt.Errorf("error while updating locked %s balance: %w", asset, err)
	}
	return nil
}
