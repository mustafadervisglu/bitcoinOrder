package repository

import (
	"bitcoinOrder/internal/domain/entity"
	"bitcoinOrder/pkg/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/labstack/gommon/email"
	"gorm.io/gorm"
	"time"
)

type IUserRepository interface {
	CreateUser(user entity.Users) (entity.Users, error)
	UpdateUser(tx *sql.DB, user entity.Users) error
	GetBalance(id string) (entity.Users, error)
	FindUser(tx *sql.DB, id string) (entity.Users, error)
	FindUserByEmail(email email.Email) (entity.Users, error)
	FindAllUser(tx *sql.DB) ([]entity.Users, error)
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

	if err := r.gormDB.Create(&user).Error; err != nil {
		return entity.Users{}, err
	}

	return user, nil
}

func (r *UserRepository) GetBalance(id string) (entity.Users, error) {
	return utils.GetBalance(r.gormDB, id)
}

func (r *UserRepository) FindUserByEmail(email email.Email) (entity.Users, error) {
	var existingUser entity.Users
	if err := r.gormDB.Where("email = ?", email).Take(&existingUser).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		return entity.Users{}, ErrUserExists
	}
	return existingUser, nil
}

func (r *UserRepository) FindUser(tx *sql.DB, id string) (entity.Users, error) {
	var user entity.Users
	sqlStatement := `
        SELECT id, email, btc_balance, usdt_balance, created_at, updated_at, deleted_at
        FROM "users" WHERE id = $1;
    `
	// Sütun sıralamasını ve tiplerini Users struct'ına göre ayarlayın
	err := tx.QueryRowContext(context.Background(), sqlStatement, id).
		Scan(&user.ID, &user.Email, &user.BtcBalance, &user.UsdtBalance,
			&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)
	if err != nil {
		return entity.Users{}, fmt.Errorf("kullanıcı bulunamadı: %w", err)
	}
	return user, nil
}

func (r *UserRepository) UpdateUser(tx *sql.DB, user entity.Users) error {
	sqlStatement := `
        UPDATE users
        SET usdt_balance = $1, btc_balance = $2, updated_at = $3
        WHERE id = $4;
    `
	var usdtBalance, btcBalance interface{}
	if user.UsdtBalance != nil {
		usdtBalance = *user.UsdtBalance
	}
	if user.BtcBalance != nil {
		btcBalance = *user.BtcBalance
	}
	updateTime := time.Now()
	_, err := tx.ExecContext(context.Background(), sqlStatement, usdtBalance, btcBalance, updateTime, user.ID)
	if err != nil {
		return fmt.Errorf("error updating user: %w", err)
	}
	return nil
}

func (r *UserRepository) FindAllUser(tx *sql.DB) ([]entity.Users, error) {
	sqlStatement := `
        SELECT * FROM users;
    `
	rows, err := tx.QueryContext(context.Background(), sqlStatement)
	if err != nil {
		return nil, fmt.Errorf("error fetching users: %w", err)
	}
	defer rows.Close()

	var users []entity.Users
	for rows.Next() {
		var user entity.Users
		err := rows.Scan(&user.ID, &user.Email, &user.BtcBalance, &user.UsdtBalance,
			&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)
		if err != nil {
			return nil, fmt.Errorf("error while scanning row: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}
