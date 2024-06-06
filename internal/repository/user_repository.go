package repository

import (
	"bitcoinOrder/internal/domain/entity"
	"bitcoinOrder/pkg/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/labstack/gommon/email"
	"gorm.io/gorm"
	"time"
)

type IUserRepository interface {
	CreateUser(user entity.Users) (entity.Users, error)
	UpdateUser(ctx context.Context, user entity.Users) error
	GetBalance(id uuid.UUID) (entity.Users, error)
	FindUser(ctx context.Context, id uuid.UUID) (entity.Users, error)
	FindUserByEmail(email email.Email) (entity.Users, error)
	FindAllUser() ([]entity.Users, error)
}

type UserRepository struct {
	gormDB *gorm.DB
	db     *sql.DB
}

func NewUserRepository(gorm *gorm.DB, db *sql.DB) IUserRepository {
	return &UserRepository{
		gormDB: gorm,
		db:     db,
	}
}

var ErrUserExists = errors.New("user already exists")

func (r *UserRepository) CreateUser(user entity.Users) (entity.Users, error) {
	sqlStatement := `INSERT INTO users (id,email,btc_balance,usdt_balance,created_at)
					 VALUES ($1,$2,$3,$4,$5)
					 RETURNING id, created_at;	
`
	var btcBalance, usdtBalance sql.NullFloat64
	if user.BtcBalance == nil {
		btcBalance.Float64 = *user.BtcBalance
		btcBalance.Valid = true
	}
	if user.UsdtBalance == nil {
		usdtBalance.Float64 = *user.UsdtBalance
		usdtBalance.Valid = true
	}
	err := r.db.QueryRowContext(context.Background(), sqlStatement, user.ID, user.Email,
		user.BtcBalance, user.UsdtBalance, time.Now()).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		return entity.Users{}, err
	}
	return user, nil
}

func (r *UserRepository) GetBalance(id uuid.UUID) (entity.Users, error) {
	return utils.GetBalance(r.gormDB, id)
}

func (r *UserRepository) FindUserByEmail(email email.Email) (entity.Users, error) {
	var existingUser entity.Users
	if err := r.gormDB.Where("email = ?", email).Take(&existingUser).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		return entity.Users{}, ErrUserExists
	}
	return existingUser, nil
}

func (r *UserRepository) FindUser(ctx context.Context, id uuid.UUID) (entity.Users, error) {
	tx, err := utils.TxFromContext(ctx)
	if err != nil {
		return entity.Users{}, err
	}

	var user entity.Users
	sqlStatement := `
        SELECT id, email, btc_balance, usdt_balance, created_at, updated_at, deleted_at
        FROM "users" WHERE id = $1;
    `
	err = tx.QueryRowContext(ctx, sqlStatement, id).
		Scan(&user.ID, &user.Email, &user.BtcBalance, &user.UsdtBalance,
			&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)
	if err != nil {
		return entity.Users{}, fmt.Errorf("user not found: %w", err)
	}
	return user, nil
}

func (r *UserRepository) UpdateUser(ctx context.Context, user entity.Users) error {
	tx, err := utils.TxFromContext(ctx)
	if err != nil {
		return err
	}

	var usdtBalance, btcBalance sql.NullFloat64
	if user.UsdtBalance != nil {
		usdtBalance.Float64 = *user.UsdtBalance
		usdtBalance.Valid = true
	}
	if user.BtcBalance != nil {
		btcBalance.Float64 = *user.BtcBalance
		btcBalance.Valid = true
	}

	sqlStatement := `
        UPDATE users
        SET usdt_balance = $1, btc_balance = $2, updated_at = $3
        WHERE id = $4;
    `
	updateTime := time.Now()
	_, err = tx.ExecContext(ctx, sqlStatement, usdtBalance, btcBalance, updateTime, user.ID)
	if err != nil {
		return fmt.Errorf("error updating user: %w", err)
	}
	return nil
}

func (r *UserRepository) FindAllUser() ([]entity.Users, error) {
	sqlStatement := `SELECT * FROM users;`
	rows, err := r.db.QueryContext(context.Background(), sqlStatement)
	if err != nil {
		return nil, fmt.Errorf("error fetching users: %w", err)
	}
	defer rows.Close()

	var users []entity.Users
	for rows.Next() {
		var user entity.Users
		err = rows.Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt, &user.Email, &user.BtcBalance, &user.UsdtBalance)
		if err != nil {
			return nil, fmt.Errorf("error while scanning row: %w", err)
		}
		users = append(users, user)
	}
	return users, nil
}
